// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	PACKAGE_NOTINSATALLED = 0
	PACKAGE_INSATALLED    = 1
	PACKAGE_UPDATE        = 3
)

type Package struct {
	Filename    string
	Name        string
	Disttag     string
	Sourcerpm   string
	URL         string
	License     string
	Description string
	Arch        string
	Distepoch   string
	Version     string
	Summary     string
	Size        int
	Group       string

	CacheID      int64
	InstalledVer string
}

func NewPackage() Package {
	return Package{}
}

func (p Package) State() int {
	if p.InstalledVer == "" {
		return PACKAGE_NOTINSATALLED
	}

	if CompareVer(p.InstalledVer, p.Version) < 0 {
		return PACKAGE_UPDATE
	}

	return PACKAGE_INSATALLED
}

func CompareVer(ver1, ver2 string) int {
	re := regexp.MustCompile(`[\.\-]`)
	v1 := re.Split(ver1, -1)
	v2 := re.Split(ver2, -1)

	cnt := len(v1)
	if cnt < len(v2) {
		cnt = len(v2)
	}

	res := 0
	for i := 0; res == 0 && i < cnt; i++ {
		s1 := "0000000000"
		s2 := "0000000000"

		if i < len(v1) {
			s1 = fmt.Sprintf("%010s", v1[i])
		}

		if i < len(v2) {
			s2 = fmt.Sprintf("%010s", v2[i])
		}

		res = strings.Compare(s1, s2)
	}

	return res
}
