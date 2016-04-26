// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strings"
)

const (
	PACKAGE_NOTINSATALLED = 0
	PACKAGE_INSATALLED    = 1
	PACKAGE_UPDATE        = 3
)

type Package struct {
	FileName    string // from synthesis
	Name        string // from synthesis
	Disttag     string // from synthesis
	Distepoch   string // from synthesis
	Sourcerpm   string // from info
	URL         string // from info
	License     string // from info
	Description string // from info
	Arch        string // from synthesis
	Version     string // from synthesis
	Summary     string // from synthesis
	Size        int    // from synthesis
	RPMSize     int    // from synthesis
	Group       string // from synthesis
	Repository  string // from synthesis

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
	ver1 += "."
	ver2 += "."
	res := 0
	s1 := ""
	s2 := ""
	for res == 0 {

		n1 := strings.IndexAny(ver1, ".-")
		n2 := strings.IndexAny(ver2, ".-")

		if n1 > -1 {
			s1 = fmt.Sprintf("%010s", ver1[:n1])
			ver1 = ver1[n1+1:]
		} else {
			s1 = "0000000000"
		}

		if n2 > -1 {
			s2 = fmt.Sprintf("%010s", ver2[:n2])
			ver2 = ver2[n2+1:]
		} else {

			s2 = "0000000000"
		}

		res = strings.Compare(s1, s2)
		if n1 < 0 && n2 < 0 {
			break
		}
	}

	return res
}

type Packages []Package

func (p Packages) Len() int {
	return len(p)
}

func (p Packages) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Packages) Less(i, j int) bool {
	return p[i].Name < p[j].Name
}
