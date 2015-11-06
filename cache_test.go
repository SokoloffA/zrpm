// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"code.google.com/p/lzma"
	"compress/gzip"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

var (
	testDir string

	saxpath_103 = Package{
		Filename:  "saxpath-1.0-3-rosa2014.1.noarch",
		Disttag:   "rosa",
		Sourcerpm: "saxpath-1.0-3.src.rpm",
		URL:       "http://sourceforge.net/projects/saxpath/",
		License:   "Saxpath",
		Description: "The SAXPath project is a Simple API for XPath. SAXPath is analogous to SAX\n" +
			"in that the API abstracts away the details of parsing and provides a simple\n" +
			"event based callback interface.",
	}

	flacon_x64_120 = Package{
		Filename:  "flacon-1.2.0-1-rosa2014.1.x86_64",
		Disttag:   "rosa",
		Sourcerpm: "flacon-1.2.0-1.src.rpm",
		URL:       "http://sourceforge.net/projects/saxpath/",
		License:   "GPLv3",
		Description: "Flacon extracts individual tracks from one big audio file containing the\n" +
			"entire album of music and saves them as separate audio files. To do this, it\n" +
			"uses information from the appropriate CUE file. Flacon also makes it possible\n" +
			"to conveniently revise or specify tags both for all tracks at once or for each\n" +
			"tag separately.",
	}

	flacon_x32_120 = Package{
		Filename:  "flacon-1.2.0-1-rosa2014.1.i586",
		Disttag:   "rosa",
		Sourcerpm: "flacon-1.2.0-1.src.rpm",
		URL:       "http://sourceforge.net/projects/saxpath/",
		License:   "GPLv3",
		Description: "Flacon extracts individual tracks from one big audio file containing the\n" +
			"entire album of music and saves them as separate audio files. To do this, it\n" +
			"uses information from the appropriate CUE file. Flacon also makes it possible\n" +
			"to conveniently revise or specify tags both for all tracks at once or for each\n" +
			"tag separately.",
	}

	boomaga_x64_060 = Package{

		Filename:    "boomaga-0.6.0-1-rosa2014.1.x86_64",
		Disttag:     "rosa",
		Sourcerpm:   "boomaga-0.6.0-1.src.rpm",
		URL:         "http://sourceforge.net/projects/saxpath/",
		License:     "LGPLv2.1+",
		Description: `Boomaga (BOOklet MAnager) is a virtual printer for viewing a document before printing it out using the physical printer.`,
	}

	boomaga_x64_071 = Package{
		Filename:    "boomaga-0.7.1-1-rosa2014.1.x86_64",
		Disttag:     "rosa",
		Sourcerpm:   "boomaga-0.7.1-1.src.rpm",
		URL:         "http://sourceforge.net/projects/saxpath/",
		License:     "LGPLv2.1+",
		Description: `Boomaga (BOOklet MAnager) is a virtual printer for viewing a document before printing it out using the physical printer.`,
	}

	boomaga_x32_060 = Package{

		Filename:    "boomaga-0.6.0-1-rosa2014.1.i586",
		Disttag:     "rosa",
		Sourcerpm:   "boomaga-0.6.0-1.src.rpm",
		URL:         "http://sourceforge.net/projects/saxpath/",
		License:     "LGPLv2.1+",
		Description: `Boomaga (BOOklet MAnager) is a virtual printer for viewing a document before printing it out using the physical printer.`,
	}

	boomaga_x32_071 = Package{
		Filename:    "boomaga-0.7.1-1-rosa2014.1.i586",
		Disttag:     "rosa",
		Sourcerpm:   "boomaga-0.7.1-1.src.rpm",
		URL:         "http://sourceforge.net/projects/saxpath/",
		License:     "LGPLv2.1+",
		Description: `Boomaga (BOOklet MAnager) is a virtual printer for viewing a document before printing it out using the physical printer.`,
	}
)

const (
	TMPL_ERROR = `Error: "%v"
key: %#v`

	TMPL_MISMATCH = `Result mismatch
------------------------
key: %#v
expected: %#v
got:      %#v`
)

func createDirs() (string, error) {
	dir, err := ioutil.TempDir("", "zrpm_test")
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir+"/etc", 0777); err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir+"/var", 0777); err != nil {
		return "", err
	}

	return dir, nil
}

func createUrpmiConfig(t *testing.T, tmpDir string, repos []Repository) {
	dir := tmpDir + "/etc"
	if err := os.MkdirAll(dir, 0777); err != nil {
		t.Error("Can't ceate urpmi config file:", err)
	}

	f, err := os.Create(dir + "/urpmi.cfg")
	if err != nil {
		t.Error("Can't ceate urpmi config file:", err)
	}
	defer f.Close()

	for _, r := range repos {
		f.WriteString(r.Name + " ")
		f.WriteString(r.URL + " {\n")
		if r.Ignore {
			f.WriteString("  ignore\n")
		}

		f.WriteString("}\n\n")
	}
}

func createPkgFiles(t *testing.T, tmpDir string, repoName string, pkgs []Package) {
	dir := tmpDir + "/var/" + repoName
	if err := os.MkdirAll(dir, 0777); err != nil {
		t.Error("Can't ceate info.xml.lzma file:", err)
		t.Fail()
	}

	f, err := os.Create(dir + "/info.xml.lzma")
	if err != nil {
		t.Error("Can't ceate info.xml.lzma file:", err)
		t.Fail()
	}
	defer f.Close()

	lz := lzma.NewWriter(f)
	defer lz.Close()

	lz.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>` + "\n"))
	lz.Write([]byte("<media_info>"))

	for _, p := range pkgs {
		lz.Write([]byte("<info "))
		lz.Write([]byte(fmt.Sprintf("fn='%s' ", p.Filename)))
		lz.Write([]byte(fmt.Sprintf("disttag='%s' ", p.Disttag)))
		lz.Write([]byte(fmt.Sprintf("distepoch='%s' ", p.Distepoch)))
		lz.Write([]byte(fmt.Sprintf("sourcerpm='%s' ", p.Sourcerpm)))
		lz.Write([]byte(fmt.Sprintf("url='%s' ", p.URL)))
		lz.Write([]byte(fmt.Sprintf("license='%s' ", p.License)))
		lz.Write([]byte(">"))
		lz.Write([]byte(p.Description))
		lz.Write([]byte("\n</info>"))
	}

	lz.Write([]byte("</media_info>"))

	f, err = os.Create(dir + "/synthesis.hdlist.cz")
	if err != nil {
		t.Error("Can't ceate info.xml.lzma file:", err)
		t.Fail()
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()
	for _, p := range pkgs {
		gz.Write([]byte(fmt.Sprintf("@summary@%s\n", p.Summary)))
		gz.Write([]byte(fmt.Sprintf("@filesize@%d\n", p.Size)))
		gz.Write([]byte(fmt.Sprintf("@info@%s@0@12345@%s@2014.1\n", p.Filename, p.Group)))
	}
}

func tableRowCount(t *testing.T, dbFile string, table string) int {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		t.Error("Can't open cache db: ", err)
		t.Fail()
	}

	var res int
	ddl := fmt.Sprintf("SELECT COUNT(id) FROM %s;", table)

	if err := db.QueryRow(ddl).Scan(&res); err != nil {
		t.Errorf("DB error: %#v", err)
		t.Fail()
	}

	return res
}

func TestCacheCreate(t *testing.T) {
	dir, err := createDirs()
	if err != nil {
		t.Error("Can't ceate tmp dir:", err)
	}
	defer os.RemoveAll(dir)

	dbFile := dir + "/test.db"
	EtcDir = dir + "/etc"
	VarDir = dir + "/var"

	var expect int
	var res int

	// ******************************************
	createUrpmiConfig(t, dir, []Repository{
		{Name: "Test", URL: "http://test.com/test"},
		{Name: "Test\\ 2", URL: "http://test.com/test2"},
	})

	createPkgFiles(t, dir, "Test", []Package{
		saxpath_103,
		flacon_x64_120,
		boomaga_x64_060,
	})

	createPkgFiles(t, dir, "Test 2", []Package{
		boomaga_x64_071,
	})

	_ = NewCache(dbFile)

	expect = 2
	res = tableRowCount(t, dbFile, "repos")
	if expect != res {
		t.Errorf("Result mismatch: expected: %v, real: %v ", expect, res)
	}

	expect = 4
	res = tableRowCount(t, dbFile, "pkgs")
	if expect != res {
		t.Errorf("Result mismatch: expected: %v, real: %v ", expect, res)
	}

	// ******************************************
	createUrpmiConfig(t, dir, []Repository{
		{Name: "Test", URL: "http://test.com/test", Ignore: true},
		{Name: "Test\\ 2", URL: "http://test.com/test2"},
	})

	_ = NewCache(dbFile)

	expect = 1
	res = tableRowCount(t, dbFile, "repos")
	if expect != res {
		t.Errorf("Result mismatch: expected: %v, real: %v ", expect, res)
	}

	expect = 1
	res = tableRowCount(t, dbFile, "pkgs")
	if expect != res {
		t.Errorf("Result mismatch: expected: %v, real: %v ", expect, res)
	}

	// ******************************************
	createUrpmiConfig(t, dir, []Repository{
		{Name: "Test", URL: "http://test.com/test"},
		{Name: "Test\\ 2", URL: "http://test.com/test2"},
	})

	createPkgFiles(t, dir, "Test", []Package{
		saxpath_103,
		flacon_x64_120,
		boomaga_x64_060,
	})

	createPkgFiles(t, dir, "Test 2", []Package{
		boomaga_x64_071,
	})

	_ = NewCache(dbFile)

	expect = 2
	res = tableRowCount(t, dbFile, "repos")
	if expect != res {
		t.Errorf("Result mismatch: expected: %v, real: %v ", expect, res)
	}

	expect = 4
	res = tableRowCount(t, dbFile, "pkgs")
	if expect != res {
		t.Errorf("Result mismatch: expected: %v, real: %v ", expect, res)
	}

	// ******************************************
	createUrpmiConfig(t, dir, []Repository{
		{Name: "Test", URL: "http://test.com/test", Ignore: false},
	})

	_ = NewCache(dbFile)

	expect = 1
	res = tableRowCount(t, dbFile, "repos")
	if expect != res {
		t.Errorf("Result mismatch: expected: %v, real: %v ", expect, res)
	}

	expect = 3
	res = tableRowCount(t, dbFile, "pkgs")
	if expect != res {
		t.Errorf("Result mismatch: expected: %v, real: %v ", expect, res)
	}

}

func TestSearch(t *testing.T) {

	cases := []struct {
		query    []string
		arch     int
		onlyLast bool
		expect   []Package
	}{
		{
			[]string{""}, Arch_64, true,
			[]Package{},
		},

		{
			[]string{"*"}, Arch_64, true,
			[]Package{boomaga_x64_071, flacon_x64_120, saxpath_103},
		},

		{
			[]string{"boomaga"}, Arch_64, true,
			[]Package{boomaga_x64_071},
		},

		{
			[]string{"b?omaga"}, Arch_64, true,
			[]Package{boomaga_x64_071},
		},

		{
			[]string{"boomaga"}, Arch_64, false,
			[]Package{boomaga_x64_071, boomaga_x64_060},
		},

		{
			[]string{"boomaga"}, Arch_32, true,
			[]Package{boomaga_x32_071},
		},

		{
			[]string{"boomaga"}, Arch_32, false,
			[]Package{boomaga_x32_071, boomaga_x32_060},
		},

		{
			[]string{"boomaga"}, Arch_All, false,
			[]Package{boomaga_x32_071, boomaga_x64_071, boomaga_x32_060, boomaga_x64_060},
		},
	}

	dir, err := createDirs()
	if err != nil {
		t.Error("Can't ceate tmp dir:", err)
	}
	defer os.RemoveAll(dir)

	dbFile := dir + "/test.db"
	EtcDir = dir + "/etc"
	VarDir = dir + "/var"

	// ******************************************
	createUrpmiConfig(t, dir, []Repository{
		{Name: "Test", URL: "http://test.com/test"},
		{Name: "Test\\ Updates", URL: "http://test.com/test2"},
		{Name: "Test32", URL: "http://test.com/test"},
	})

	createPkgFiles(t, dir, "Test", []Package{
		saxpath_103,
		flacon_x64_120,
		boomaga_x64_060,
	})

	createPkgFiles(t, dir, "Test Updates", []Package{
		boomaga_x64_071,
	})

	createPkgFiles(t, dir, "Test32", []Package{
		flacon_x32_120,
		boomaga_x32_060,
		boomaga_x32_071,
	})

	cache := NewCache(dbFile)

	for _, c := range cases {
		out := cache.SearchByName(c.query, c.arch, c.onlyLast)

		exp := []string{}
		for _, p := range c.expect {
			exp = append(exp, p.Filename)
		}

		res := []string{}
		for p := range out {
			res = append(res, p.Filename)
		}

		if !reflect.DeepEqual(res, exp) {
			t.Errorf(TMPL_MISMATCH, c.query, exp, res)
		}
	}

}
