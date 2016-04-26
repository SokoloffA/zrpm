// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"code.google.com/p/lzma"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
)

const (
	chankSize = 1024 * 512
)

type InfoRecord struct {
	Filename    string `xml:"fn,attr"`
	Name        string
	Sourcerpm   string `xml:"sourcerpm,attr"`
	URL         string `xml:"url,attr"`
	License     string `xml:"license,attr"`
	Description string `xml:",chardata"`
	Distepoch   string `xml:"distepoch"`
	Disttag     string `xml:"disttag,attr"`
}

func ReadInfoFile(file string, out chan<- InfoRecord) error {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil
	}

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	lz := lzma.NewReader(f)
	defer lz.Close()

	buf, err := ioutil.ReadAll(lz)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	sz := len(buf)
	for i := 0; i < sz; i += chankSize {
		to := i + chankSize
		if to > sz {
			to = sz
		}

		wg.Add(1)
		go func(f int, t int) {
			defer wg.Done()
			err = extractInfo(buf, f, t, out)
		}(i, to)
	}

	wg.Wait()
	return nil
}

func extractInfo(buffer []byte, begin int, end int, out chan<- InfoRecord) error {
	buf := buffer[begin:]
	n := begin
	for true {
		b := bytes.Index(buf, []byte("<info "))
		e := bytes.Index(buf, []byte("</info>"))

		if b < 0 {
			return nil
		}

		if e < 0 {
			return fmt.Errorf("Can't parse %s", "file")
		}
		e += 7

		var res InfoRecord

		err := xml.Unmarshal(buf[b:e], &res)
		if err != nil {
			return err
		}

		out <- res

		buf = buf[e:]
		n += e

		if n > end {
			return nil
		}
	}

	return nil
}
