// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"
)

func TestCompareVer(t *testing.T) {
	cases := []struct {
		ver1   string
		ver2   string
		expect int
	}{
		{"", "", 0},
		{"", "0", 0},
		{"", "1", -1},
		{"2", "", 1},
		{"0.1", "0.1", 0},
		{"0.1", "0.1.0", 0},
		{"0.1", "0.1.1", -1},
		{"0.2", "0.1.1", 1},
		{"0.10.0", "0.9.9", 1},
		{"0.10.0", "0.009.9", 1},
		{"0.1.1", "0.1.9", -1},
		{"git.0.2", "git.0.1.1", 1},
	}

	for _, c := range cases {
		res := CompareVer(c.ver1, c.ver2)

		if res != c.expect {

			t.Errorf(`Result mismatch
------------------------
ver1: %#v
ver2: %#v
expected: %v
got:      %v`,
				c.ver1,
				c.ver2,
				c.expect, res)
			return
		}

	}
}
