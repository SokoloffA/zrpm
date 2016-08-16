// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"io"
	"log"
	"os/exec"
	"path"
	"sort"
	"strings"
	"sync"
)

type Cache struct {
	packages Packages
}

func NewCache() *Cache {
	c := &Cache{}

	info := map[string]InfoRecord{}
	installed := map[string]string{}

	var wg sync.WaitGroup

	// Get info about repositories
	repos, err := GetRepositories()
	if err != nil {
		log.Fatal("Can't read urpi.cfg file: ", err)
	}

	wg.Add(1)

	go func() {
		defer wg.Done()
		installed = fillInstalledInfo()
	}()

	syntesisChan := make(chan Package, 9999)
	var synthWg sync.WaitGroup

	infoChan := make(chan InfoRecord, 9999)
	var infoWg sync.WaitGroup

	for _, repo := range repos {

		if repo.Ignore {
			continue
		}

		// **************************************
		// Parse synthesis.hdlist.cz file
		synthWg.Add(1)
		go func(r Repository) {
			defer synthWg.Done()
			err = ReadSynthesisFile(r, r.Dir+"/synthesis.hdlist.cz", syntesisChan)
			if err != nil {
				log.Fatal("Can't read synthesis file ", r.Dir+"/synthesis.hdlist.cz", ": ", err)
			}
		}(repo)

		// **************************************
		// Parse info.xml.lzma file
		infoWg.Add(1)
		go func(r Repository) {
			defer infoWg.Done()
			err = ReadInfoFile(r.Dir+"/info.xml.lzma", infoChan)
			if err != nil {
				log.Fatal("Can't read info file ", r.Dir+"/info.xml.lzma", ": ", err)
			}
		}(repo)
	}

	go func() {
		synthWg.Wait()
		close(syntesisChan)
	}()

	go func() {
		infoWg.Wait()
		close(infoChan)
	}()

	// **************************************
	// Save data from synthesis.hdlist.cz file

	wg.Add(1)
	go func() {
		defer wg.Done()
		for pkg := range syntesisChan {
			c.packages = append(c.packages, pkg)

		}
	}()

	// **************************************
	// Save data from info.xml.lzma file
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range infoChan {
			info[i.Filename] = i
		}
	}()

	wg.Wait()

	sort.Sort(c.packages)
	n := 0
	for i, pkg := range c.packages {
		if n < len(pkg.Name) {
			n = len(pkg.Name)
		}
		inf, ok := info[pkg.FileName]
		if ok {
			pkg.Sourcerpm = inf.Sourcerpm
			pkg.URL = inf.URL
			pkg.License = inf.License
			pkg.Description = inf.Description

			c.packages[i] = pkg
		}

		ver, ok := installed[pkg.Name]
		if ok {
			pkg.InstalledVer = ver
			c.packages[i] = pkg
		}
	}

	return c
}

func fillInstalledInfo() map[string]string {
	res := map[string]string{}
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
			log.Fatal("can't get installed versions:", err)
		}

		line = strings.Trim(line, "\n\r")
		items := strings.Split(line, "\t")
		res[items[0]] = items[1]
	}
	return res
}

type packagesVerSorted []Package

func (p packagesVerSorted) Len() int {
	return len(p)
}

func (p packagesVerSorted) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p packagesVerSorted) Less(i, j int) bool {
	return CompareVer(p[i].Version, p[j].Version) > 0
}

func compareName(names []string, pkg Package) bool {
	for _, word := range names {
		if word == "" {
			continue
		}

		if word == "*" {
			return true
		}

		word = "*" + strings.ToLower(word) + "*"
		ok, _ := path.Match(word, strings.ToLower(pkg.Name))
		if ok {
			return true
		}
	}
	return false
}

func compareArch(arch []string, pkg Package) bool {
	for _, a := range arch {
		if strings.EqualFold(pkg.Arch, a) {
			return true
		}
	}

	return false
}

func (c Cache) SearchByName(names []string, arch []string, onlyLast bool) <-chan Package {
	out := make(chan Package)

	prevName := ""
	prog := packagesVerSorted{}

	go func() {
		defer close(out)

		for _, pkg := range c.packages {
			if !compareArch(arch, pkg) {
				continue
			}

			if !compareName(names, pkg) {
				continue
			}

			if prevName != pkg.Name {
				prog.processOneProgPkgs(names, arch, onlyLast, out)

				prevName = pkg.Name
				prog = packagesVerSorted{}
			}

			prog = append(prog, pkg)
		}

		prog.processOneProgPkgs(names, arch, onlyLast, out)
	}()

	return out
}

func (pkgs packagesVerSorted) processOneProgPkgs(names []string, arch []string, onlyLast bool, out chan<- Package) {
	if len(pkgs) == 0 {
		return
	}

	sort.Sort(pkgs)

	if onlyLast {
		out <- pkgs[0]
		return
	}

	for _, p := range pkgs {
		out <- p
	}
}
