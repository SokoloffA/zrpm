// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"code.google.com/p/lzma"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"testing"
)

var (
	testDir string

	saxpath_103 = Package{
		FileName:  "saxpath-1.0-3-rosa2014.1.noarch",
		Disttag:   "rosa",
		Sourcerpm: "saxpath-1.0-3.src.rpm",
		URL:       "http://sourceforge.net/projects/saxpath/",
		License:   "Saxpath",
		Description: "The SAXPath project is a Simple API for XPath. SAXPath is analogous to SAX\n" +
			"in that the API abstracts away the details of parsing and provides a simple\n" +
			"event based callback interface.",
	}

	flacon_x64_120 = Package{
		FileName:  "flacon-1.2.0-1-rosa2014.1.x86_64",
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
		FileName:  "flacon-1.2.0-1-rosa2014.1.i586",
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

		FileName:    "boomaga-0.6.0-1-rosa2014.1.x86_64",
		Disttag:     "rosa",
		Sourcerpm:   "boomaga-0.6.0-1.src.rpm",
		URL:         "http://sourceforge.net/projects/saxpath/",
		License:     "LGPLv2.1+",
		Description: `Boomaga (BOOklet MAnager) is a virtual printer for viewing a document before printing it out using the physical printer.`,
	}

	boomaga_x64_071 = Package{
		FileName:    "boomaga-0.7.1-1-rosa2014.1.x86_64",
		Disttag:     "rosa",
		Sourcerpm:   "boomaga-0.7.1-1.src.rpm",
		URL:         "http://sourceforge.net/projects/saxpath/",
		License:     "LGPLv2.1+",
		Description: `Boomaga (BOOklet MAnager) is a virtual printer for viewing a document before printing it out using the physical printer.`,
	}

	boomaga_x32_060 = Package{

		FileName:    "boomaga-0.6.0-1-rosa2014.1.i586",
		Disttag:     "rosa",
		Sourcerpm:   "boomaga-0.6.0-1.src.rpm",
		URL:         "http://sourceforge.net/projects/saxpath/",
		License:     "LGPLv2.1+",
		Description: `Boomaga (BOOklet MAnager) is a virtual printer for viewing a document before printing it out using the physical printer.`,
	}

	boomaga_x32_071 = Package{
		FileName:    "boomaga-0.7.1-1-rosa2014.1.i586",
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
		lz.Write([]byte(fmt.Sprintf("fn='%s' ", p.FileName)))
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
		gz.Write([]byte(fmt.Sprintf("@info@%s@0@12345@%s@rosa@2014.1\n", p.FileName, p.Group)))
	}
}

func TestCompareName(t *testing.T) {
	cases := []struct {
		words  []string
		pkg    Package
		expect bool
	}{
		{
			[]string{""},
			Package{Name: "boomaga"},
			false,
		},

		{
			[]string{"*"},
			Package{Name: "boomaga"},
			true,
		},

		{
			[]string{"boo"},
			Package{Name: "boomaga"},
			true,
		},

		{
			[]string{"boomaga"},
			Package{Name: "boomaga"},
			true,
		},
		{
			[]string{"oo"},
			Package{Name: "boomaga"},
			true,
		},

		{
			[]string{"o"},
			Package{Name: "boomaga"},
			true,
		},
		{
			[]string{"aga"},
			Package{Name: "boomaga"},
			true,
		},

		{
			[]string{"BooMaGa"},
			Package{Name: "boomaga"},
			true,
		},
		{
			[]string{"boomagaa"},
			Package{Name: "boomaga"},
			false,
		},
		{
			[]string{"bomaga"},
			Package{Name: "boomaga"},
			false,
		},
	}

	for _, c := range cases {
		res := compareName(c.words, c.pkg)
		if res != c.expect {
			t.Errorf(`Result mismatch
------------------------
words: %#v
package name: %v
expected: %#v
got:      %#v`,
				c.words,
				c.pkg.Name,
				c.expect,
				res)
		}
	}
}

func TestSearch(t *testing.T) {

	cases := []struct {
		query    []string
		arch     []string
		onlyLast bool
		expect   []string
	}{
		{
			[]string{""}, []string{"x86_64", "noarch"}, true,
			[]string{},
		},

		{
			[]string{"*"}, []string{"x86_64", "noarch"}, true,
			[]string{boomaga_x64_071.FileName, flacon_x64_120.FileName, saxpath_103.FileName},
		},

		{
			[]string{"boomaga"}, []string{"x86_64", "noarch"}, true,
			[]string{boomaga_x64_071.FileName},
		},

		{
			[]string{"b?omaga"}, []string{"x86_64", "noarch"}, true,
			[]string{boomaga_x64_071.FileName},
		},

		{
			[]string{"boomaga"}, []string{"x86_64", "noarch"}, false,
			[]string{boomaga_x64_071.FileName, boomaga_x64_060.FileName},
		},

		{
			[]string{"boomaga"}, []string{"i586", "noarch"}, true,
			[]string{boomaga_x32_071.FileName},
		},

		{
			[]string{"boomaga"}, []string{"i586", "noarch"}, false,
			[]string{boomaga_x32_071.FileName, boomaga_x32_060.FileName},
		},

		{
			[]string{"boomaga"}, []string{"noarch", "x86_64", "i586"}, false,
			[]string{boomaga_x32_071.FileName, boomaga_x64_071.FileName, boomaga_x32_060.FileName, boomaga_x64_060.FileName},
		},
	}

	dir, err := createDirs()
	if err != nil {
		t.Error("Can't ceate tmp dir:", err)
	}
	defer os.RemoveAll(dir)

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

	cache := NewCache()

	for _, c := range cases {
		out := cache.SearchByName(c.query, c.arch, c.onlyLast)

		res := []string{}
		for p := range out {
			res = append(res, p.FileName)
		}

		sort.Strings(res)
		sort.Strings(c.expect)

		if !reflect.DeepEqual(res, c.expect) {
			t.Errorf(TMPL_MISMATCH, c.query, c.expect, res)
		}
	}

}
