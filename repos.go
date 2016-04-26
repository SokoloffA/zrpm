// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"os"
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
	Ignore     bool
	Dir        string
	LastUpdate time.Time
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

		rep := Repository{}
		rep.Name = strings.Replace(head[:n], "\\", "", -1)
		rep.URL = head[n:]
		rep.Ignore = strings.Contains(body, "ignore")

		rep.Dir = VarDir + "/" + rep.Name
		if !rep.Ignore {
			//rep.Md5, err = readMd5(rep.Dir + "/MD5SUM")

			if stat, err := os.Lstat(rep.Dir + "/MD5SUM"); err == nil {
				rep.LastUpdate = stat.ModTime()
			}
		}
		res = append(res, rep)
	}

	err = nil
	return
}
