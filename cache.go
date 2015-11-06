// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"crypto/md5"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

const (
	Arch_noarch = 1
	Arch_i586   = 2
	Arch_x86_64 = 4

	Arch_32  = Arch_noarch | Arch_i586
	Arch_64  = Arch_noarch | Arch_x86_64
	Arch_All = Arch_noarch | Arch_i586 | Arch_x86_64
)

const (
	DB_DDL = `
	CREATE TABLE IF NOT EXISTS sys (
		id integer UNIQUE,
		dbid string
	);

	INSERT OR IGNORE INTO sys (id) VALUES (0);

	CREATE TABLE IF NOT EXISTS repos (
		id integer not null primary key,
		name string UNIQUE,
		file string,
		md5 string
	);

	CREATE TABLE  IF NOT EXISTS pkgs (
		id integer not null primary key,
 				
		filename    string  NOT NULL,
		name        string  NOT NULL,
		summary	    string  DEFAULT "",
		size        integer DEFAULT 0,
		disttag     string  DEFAULT "",
		sourcerpm   string  DEFAULT "",
		url         string  DEFAULT "",
		license     string  DEFAULT "",
		description string  DEFAULT "",
		arch        string  DEFAULT "",
		distepoch   string  DEFAULT "",
		version     string  DEFAULT "",
		grp         string  DEFAULT "",
		last		boolean DEFAULT false,
		repoid integer NOT NULL, 
		FOREIGN KEY(repoid)	REFERENCES repos(id) ON DELETE CASCADE
				
	);`
)

type Cache struct {
	dbFile string
	rpmDB  map[string]string
}

func NewCache(cacheFile string) *Cache {

	c := &Cache{
		dbFile: cacheFile,
		rpmDB:  map[string]string{},
	}

	// Fill info about installed packages
	var rpmWg sync.WaitGroup
	rpmWg.Add(1)
	go func() {
		defer rpmWg.Done()
		c.fillRpmDB()
	}()

	db := c.openDB()

	// Remove outdated DB .............
	dbID := fmt.Sprintf("%x", md5.Sum([]byte(DB_DDL)))
	var oldID string
	_ = db.QueryRow("SELECT dbid FROM sys").Scan(&oldID)
	if oldID != dbID {
		db.Close()
		os.Remove(cacheFile)
		db = c.openDB()
	}

	// Create DB ......................
	if _, err := db.Exec(DB_DDL); err != nil {
		log.Fatal("Can't create DB:", err, "\n", DB_DDL)
	}

	if _, err := db.Exec("UPDATE sys SET dbID = ?;", dbID); err != nil {
		log.Fatal("Can't create DB:", err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal("Can't update cache:", err)
	}
	defer tx.Rollback()

	c.update(tx)
	tx.Commit()

	rpmWg.Wait()
	return c
}

func (c Cache) openDB() *sql.DB {
	db, err := sql.Open("sqlite3", c.dbFile)
	if err != nil {
		log.Fatal("Can't open cache db:", err)
	}

	ddl := `PRAGMA foreign_keys = ON;
	 		PRAGMA journal_mode = MEMORY;
	 		PRAGMA temp_store = MEMORY;
		`
	// 	PRAGMA automatic_index = ON;
	//        PRAGMA cache_size = 32768;
	//        PRAGMA cache_spill = OFF;

	//        PRAGMA journal_size_limit = 67110000;
	//        PRAGMA locking_mode = NORMAL;
	//        PRAGMA page_size = 4096;
	//        PRAGMA recursive_triggers = ON;
	//        PRAGMA secure_delete = ON;
	//        PRAGMA synchronous = NORMAL;
	//        PRAGMA temp_store = MEMORY;
	//
	//        PRAGMA wal_autocheckpoint = 16384;
	// `
	if _, err := db.Exec(ddl); err != nil {
		log.Fatal(err)
	}

	return db
}

func (c *Cache) update(tx *sql.Tx) {

	reps, err := GetRepositories()
	if err != nil {
		log.Fatal("Can't read urpi.cfg file: ", err)
	}

	// Remove outdated repos ..........
	outdated := map[string]bool{}
	rows, err := tx.Query("SELECT name FROM repos")
	if err != nil {
		log.Fatal("Can't update cache:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatal("Can't update cache:", err)
		}

		outdated[name] = true
	}

	for _, rep := range reps {
		outdated[rep.Name] = rep.Ignore
	}

	for n, out := range outdated {
		if out {
			_, err = tx.Exec(`delete from repos where name = ?`, n)
		}
	}

	// Add packages ...................
	changedCnt := 0
	var wg sync.WaitGroup
	for _, rep := range reps {
		if rep.Ignore {
			continue
		}

		var repoID int64
		var md5 string
		err = tx.QueryRow("SELECT id, md5 FROM repos WHERE name = ?", rep.Name).Scan(&repoID, &md5)
		if err != nil && err != sql.ErrNoRows {
			log.Fatal("DB error:", err)
		}

		if md5 == "" || md5 != rep.Md5 {
			if changedCnt == 0 {
				fmt.Printf("Updating cache ... ")
				changedCnt++
			}

			wg.Add(1)
			go func(r Repository, rID int64) {
				defer wg.Done()
				updatePkgs(tx, r, rID)
			}(rep, repoID)
		}
	}

	wg.Wait()

	if changedCnt > 0 {
		fmt.Printf(" Done\n")
	}

	if changedCnt > 0 {
		// Set last mark ..................
		_, err = tx.Exec(`UPDATE pkgs SET last = 0;
			UPDATE pkgs SET last = 1 
			WHERE id in (
				SELECT p1.id FROM pkgs p1 
				WHERE p1.version = (
					SELECT max(p2.version) FROM pkgs p2 
					WHERE p2.Name = p1.name AND 
						  p2.arch = p1.arch
					)
			);`)
	}
}

func updatePkgs(tx *sql.Tx, rep Repository, repoID int64) {
	if repoID == 0 {
		res, err := tx.Exec(`INSERT INTO repos (name, md5) VALUES(?,?)`, rep.Name, rep.Md5)
		if err != nil {
			log.Fatal("DB error:", err)
		}

		if repoID, err = res.LastInsertId(); err != nil {
			log.Fatal("DB error:", err)
		}

	} else {
		_, err := tx.Exec(`delete from pkgs where repoid = ?`, repoID)
		if err != nil {
			log.Fatal("DB error:", err)
		}
	}

	stmt, err := tx.Prepare(`
	 	INSERT INTO pkgs
	 		(repoid,
	 		filename,
	 		name,
	 		disttag,
	 		sourcerpm,
	 		url,
	 		license,
	 		description,
	 		arch,
	 		distepoch,
	 		version,
	 		summary,
	 		size,
	 		grp)
	 	VALUES
	 		(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	 	`)

	for pkg := range rep.ReadPackages() {
		_, err = stmt.Exec(
			repoID,
			pkg.Filename,
			pkg.Name,
			pkg.Disttag,
			pkg.Sourcerpm,
			pkg.URL,
			pkg.License,
			pkg.Description,
			pkg.Arch,
			pkg.Distepoch,
			pkg.Version,
			pkg.Summary,
			pkg.Size,
			pkg.Group)

		if err != nil {
			log.Fatal("DB error:", err)
		}
	}

	if rep.Error != nil {
		log.Fatal("Can't read package info:", rep.Error)
	}
}

func (c *Cache) fillRpmDB() {
	cmd := exec.Command("rpm", "-q", "-a", "--qf", "%{NAME}\t%{VERSION}-%{RELEASE}\n")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	r := bufio.NewReader(stdout)
	for err == nil {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal("can't read synthesis file:", err)
		}

		line = strings.Trim(line, "\n\r")
		items := strings.Split(line, "\t")
		c.rpmDB[items[0]] = items[1]
	}
}

func (c Cache) search(query string, args ...interface{}) <-chan Package {
	out := make(chan Package)

	query = strings.Replace(query,
		"PKG_FIELDS",
		`id,
			filename,
 			name,
 			disttag,
 			sourcerpm,
 			URL,
			license,
			description,
			arch,
			distepoch,
			version,
			summary,
			size,
			grp`,
		-1)

	go func() {
		db := c.openDB()
		defer db.Close()

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Fatal("DB error:", err, "\n", query)
		}
		defer rows.Close()

		var pkg Package
		for rows.Next() {
			err := rows.Scan(
				&pkg.CacheID,
				&pkg.Filename,
				&pkg.Name,
				&pkg.Disttag,
				&pkg.Sourcerpm,
				&pkg.URL,
				&pkg.License,
				&pkg.Description,
				&pkg.Arch,
				&pkg.Distepoch,
				&pkg.Version,
				&pkg.Summary,
				&pkg.Size,
				&pkg.Group,
			)

			if err != nil {
				log.Fatal("DB error:", err, "\n", query)
			}
			pkg.InstalledVer = c.rpmDB[pkg.Name]
			out <- pkg
		}

		close(out)

	}()
	return out
}

func (c Cache) SearchByName(names []string, arch int, onlyLast bool) <-chan Package {
	var args []interface{}
	query := "SELECT PKG_FIELDS FROM pkgs WHERE "

	// Name ...........................
	query += "("
	for i, name := range names {
		if i == 0 {
			query += "(name like ? ) "
		} else {
			query += "OR (name like ? ) "
		}

		name = strings.Replace(name, "*", "%", -1)
		name = strings.Replace(name, "?", "_", -1)
		args = append(args, name)
	}
	query += ") "

	// Arch ...........................
	query += "AND arch IN ("
	comma := ""
	if arch&Arch_noarch > 0 {
		query += comma + "?"
		comma = ","
		args = append(args, "noarch")
	}

	if arch&Arch_i586 > 0 {
		query += comma + "?"
		args = append(args, "i586")
	}

	if arch&Arch_x86_64 > 0 {
		query += comma + "?"
		args = append(args, "x86_64")
	}
	query += ") "

	if onlyLast {
		query += " AND last = 1 "
	}

	query += "ORDER BY name ASC, version DESC, arch ASC"
	return c.search(query, args...)
}
