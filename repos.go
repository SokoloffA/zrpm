// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"code.google.com/p/lzma"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	EtcDir = "/etc/urpmi"
	VarDir = "/var/lib/urpmi"
)

type Repository struct {
	Name       string
	URL        string
	Md5        string
	Ignore     bool
	ID         int64
	Dir        string
	LastUpdate time.Time
	Error      error

	f         *os.File
	lz        io.ReadCloser
	xml       *xml.Decoder
	synthesis map[string]synthesis
}

type synthesis struct {
	summary string
	size    int
	group   string
}

func NewRepository() Repository {
	return Repository{
		synthesis: map[string]synthesis{},
	}
}

func GetRepositories() (res []Repository, err error) {
	f, err := os.Open(EtcDir + "/urpmi.cfg")
	if err != nil {
		return
	}
	defer f.Close()

	r := bufio.NewReader(f)

	var head, body string
	var ferr error
	for ferr == nil {

		head, ferr = r.ReadString('{')
		head = strings.Trim(head, " \t\r\n{")

		body, ferr = r.ReadString('}')
		body = strings.Trim(body, " \t\r\n}")

		if len(head) == 0 {
			continue
		}

		n := strings.LastIndexAny(head, " \t")
		if n < 0 {
			err = fmt.Errorf("incorrect media line '%s' in urpmi.cfg", head)
			return
		}

		rep := NewRepository()
		rep.Name = strings.Replace(head[:n], "\\", "", -1)
		rep.URL = head[n:]
		rep.Ignore = strings.Contains(body, "ignore")

		rep.Dir = VarDir + "/" + rep.Name
		if !rep.Ignore {
			rep.Md5, err = readMd5(rep.Dir + "/MD5SUM")

			if stat, err := os.Lstat(rep.Dir + "/MD5SUM"); err == nil {
				rep.LastUpdate = stat.ModTime()
			}
		}
		res = append(res, rep)
	}

	err = nil
	return
}

func readMd5(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("can't read file: %s", err)
	}

	r := bufio.NewReader(f)
	var line string
	for err == nil {
		line, err = r.ReadString('\n')
		if strings.Contains(line, "info.xml.lzma") {
			return line[:strings.Index(line, " ")], nil
		}
	}

	return "", fmt.Errorf("can't get MD5 for info.xml.lzma from: %s", file)
}

func (rep Repository) readSynthesisHdlist() (map[string]synthesis, error) {
	res := map[string]synthesis{}

	f, err := os.Open(rep.Dir + "/synthesis.hdlist.cz")
	if err != nil {
		return res, err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return res, err
	}
	defer gz.Close()

	r := bufio.NewReader(gz)

	cur := synthesis{}
	for err == nil {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}

		if err != nil {
			return res, fmt.Errorf("can't read synthesis file: %v", err)
		}

		line = strings.Trim(line, "\n\r")

		if strings.HasPrefix(line, "@info") {
			items := strings.Split(line, "@")
			if len(items) < 3 {
				return res, fmt.Errorf("can't read synthesis file: package filename not found %v", line)
			}

			if len(items) > 5 {
				cur.group = items[5]
			}

			res[items[2]] = cur
			cur = synthesis{}
			continue
		}

		if strings.HasPrefix(line, "@summary@") {
			cur.summary = line[9:]
			continue
		}

		if strings.HasPrefix(line, "@filesize@") {
			items := strings.Split(line, "@")
			if len(items) < 3 {
				return res, fmt.Errorf("can't read synthesis file: package filesize not found %v", line)
			}

			cur.size, err = strconv.Atoi(items[2])
			if err != nil {
				return res, fmt.Errorf("can't read synthesis file: incorrect filesize %v: %v", items[2], err)
			}
			continue
		}
	}

	return res, nil
}

func (rep *Repository) ReadPackages() <-chan Package {
	out := make(chan Package)

	go func() {
		var err error
		defer close(out)
		defer func() { rep.Error = err }()

		synthesis, err := rep.readSynthesisHdlist()
		if err != nil {
			return
		}

		f, err := os.Open(rep.Dir + "/info.xml.lzma")
		if err != nil {
			return
		}
		defer f.Close()

		lz := lzma.NewReader(f)
		defer lz.Close()
		x := xml.NewDecoder(lz)

		var token xml.Token
		for {
			token, err = x.Token()

			if err == io.EOF {
				err = nil
				return
			}

			if token == nil {
				return
			}

			switch se := token.(type) {
			case xml.StartElement:
				if se.Name.Local == "info" {
					pkg, err := packageFromXML(x, se)
					if err != nil {
						return
					}
					synt := synthesis[pkg.Filename]
					pkg.Summary = synt.summary
					pkg.Size = synt.size
					pkg.Group = synt.group
					out <- pkg
				}
			}
		}

	}()

	return out
}

func packageFromXML(xml *xml.Decoder, se xml.StartElement) (Package, error) {
	record := struct {
		Fn          string `xml:"fn,attr"`
		Distepoch   string `xml:"distepoch"`
		Disttag     string `xml:"disttag,attr"`
		Sourcerpm   string `xml:"sourcerpm,attr"`
		URL         string `xml:"url,attr"`
		License     string `xml:"license,attr"`
		Description string `xml:",chardata"`
	}{}

	err := xml.DecodeElement(&record, &se)

	if err != nil {
		return Package{}, err
	}

	pkg := Package{}
	pkg.Filename = record.Fn
	pkg.Disttag = record.Disttag
	pkg.Sourcerpm = record.Sourcerpm
	pkg.URL = record.URL
	pkg.License = record.License
	pkg.Description = record.Description
	pkg.Distepoch = record.Distepoch

	s := ""
	if pkg.Disttag != "" {
		s += "-" + pkg.Disttag + pkg.Distepoch + ".*"
	}

	re := regexp.MustCompile(`^(.*)-([^\-]*)-([^\-]*)` + s + `\.([^\.\-]*)$`)
	parts := re.FindStringSubmatch(pkg.Filename)
	if parts == nil {
		return pkg, fmt.Errorf("can't parse pckage filename %s", pkg.Filename)
	}

	pkg.Name = parts[1]
	pkg.Version = parts[2] + "-" + parts[3]
	pkg.Arch = parts[4]

	return pkg, err
}
