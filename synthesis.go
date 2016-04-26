// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	gzip "github.com/klauspost/pgzip"

	"fmt"
	"io"
	"os"

	"strconv"
	"strings"
)

func splitLine(line string, out ...*string) error {
	items := strings.Split(line, "@")
	if len(items) < len(out)+2 {
		return fmt.Errorf("Can't parse synthesis line %v: expected %v fields got %v", line, len(out)+2, len(items))
	}

	for i := 0; i < len(out); i++ {
		if out[i] != nil {
			*out[i] = items[i+2]
		}
	}

	return nil
}

func ReadSynthesisFile(repo Repository, file string, out chan<- Package) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	r := bufio.NewReader(gz)

	cur := Package{}
	for err == nil {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("can't read synthesis file: %v", err)
		}

		line = strings.Trim(line, "\n\r")

		// @provides@libvo-amrwbenc.so.0@libvo-amrwbenc0[== 0.1.2-1:2014.1]
		// @requires@libc.so.6@libc.so.6(GLIBC_2.0)@libc.so.6(GLIBC_2.1.3)
		// @summary@VisualOn AMR-WB encoder library
		// @filesize@68793
		// @info@libvo-amrwbenc0-0.1.2-1-rosa2014.1.i586@0@143156@System/Libraries@rosa@2014.1

		if strings.HasPrefix(line, "@info") {
			var size string

			err = splitLine(line,
				&cur.FileName,
				nil,
				&size,
				&cur.Group,
				&cur.Disttag,
				&cur.Distepoch)
			if err != nil {
				return err
			}

			cur.Repository = repo.Name

			cur.Size, err = strconv.Atoi(size)
			if err != nil {
				return fmt.Errorf("Can't read synthesis file: incorrect filesize '%v': %v", size, err)
			}

			items := strings.Split(cur.FileName, "-")
			if len(items) < 4 {
				return fmt.Errorf("Can't parse package filename %s", cur.FileName)
			}

			s := items[len(items)-1]
			n := strings.LastIndex(s, ".")
			cur.Arch = s[n+1:]

			cur.Version = items[len(items)-3] + "-" + items[len(items)-2]
			cur.Name = strings.Join(items[:len(items)-3], "-")

			out <- cur
			cur = Package{}
			continue
		}

		if strings.HasPrefix(line, "@summary@") {
			cur.Summary = line[9:]
			continue
		}

		if strings.HasPrefix(line, "@filesize@") {
			items := strings.Split(line, "@")
			if len(items) < 3 {
				return fmt.Errorf("can't read synthesis file: package filesize not found %v", line)
			}

			cur.RPMSize, err = strconv.Atoi(items[2])
			if err != nil {
				return fmt.Errorf("Can't read synthesis file: incorrect RPM size %v: %v", items[2], err)
			}
			continue
		}
	}

	return nil
}
