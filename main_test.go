// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"
)

func TestColorizeResultStringr(t *testing.T) {
	cases := []struct {
		query  []string
		from   string
		expect string
	}{
		{[]string{""}, "", ""},
		{[]string{"qt"}, "libqt", "lib[B]qt[/B]"},
		{[]string{"qt"}, "libqtconfig", "lib[B]qt[/B]config"},
		{[]string{"Qt"}, "libqt", "lib[B]qt[/B]"},
		{[]string{"Qt"}, "libqtconfig", "lib[B]qt[/B]config"},
		{[]string{"qt"}, "libQt", "lib[B]Qt[/B]"},
		{[]string{"qt"}, "libQtconfig", "lib[B]Qt[/B]config"},
		{[]string{"qt"}, "libQtqt", "lib[B]Qt[/B][B]qt[/B]"},
		{[]string{"qt"}, "libQtconfigqt", "lib[B]Qt[/B]config[B]qt[/B]"},

		{[]string{"qt", "lib"}, "libQtconfigqt", "[B]lib[/B][B]Qt[/B]config[B]qt[/B]"},
	}

	colorBold = "[B]"
	colorNorm = "[/B]"

	for _, c := range cases {
		res := colorizeResultString(c.query, c.from)

		if res != c.expect {

			t.Errorf(`Result mismatch
------------------------
query: %#v
string: %#v
expected: %v
got:      %v`,
				c.query,
				c.from,
				c.expect, res)
			return
		}

	}
}
