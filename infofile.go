// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	//	"bytes"
	"code.google.com/p/lzma"
	"encoding/xml"
	//	"fmt"
	"io"
	//	"io/ioutil"
	"os"
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
	/*
	   	if false {
	   		println("NEW")

	   		buf, err := ioutil.ReadAll(lz)
	   		if err != nil {
	   			return err
	   		}

	   		//		var wg sync.WaitGroup
	   		//		cnt := 1
	   		//		wg.Add(cnt)
	   		//		sz := len(buf)

	   		// 		for i:=0;i<=sz; i+1024*200; {
	   		// 			wg.Add(1)
	   		// 	go func() {
	   		// extractInfo(buf, i, , out)
	   		// 		}()
	   		// }
	   		// go func() {

	   		// }
	   		err = extractInfo(buf, 0, len(buf)/1, out)
	   		return nil
	   		//####################################3

	   		cur := buf[0:]
	   		for true {
	   			b := bytes.Index(cur, []byte("<info "))
	   			e := bytes.Index(cur, []byte("</info>"))
	   			if b < 0 {
	   				break
	   			}

	   			if e < 0 {
	   				return fmt.Errorf("Can't parse %s", file)
	   			}
	   			e += 7

	   			go func(_b int, _e int) error {
	   				record := struct {
	   					Fn          string `xml:"fn,attr"`
	   					Distepoch   string `xml:"distepoch"`
	   					Disttag     string `xml:"disttag,attr"`
	   					Sourcerpm   string `xml:"sourcerpm,attr"`
	   					URL         string `xml:"url,attr"`
	   					License     string `xml:"license,attr"`
	   					Description string `xml:",chardata"`
	   				}{}

	   				err = nil
	   				err = xml.Unmarshal(buf[_b:_e], &record)
	   				if err != nil {
	   					return err
	   				}

	   				return nil
	   			}(b, e)


	   //				err = xml.Unmarshal(buf[b:e], &record)
	   //				if err != nil {
	   //					return err
	   //				}


	   			//r := bytes.Reader(buf[b:e])

	   			//err := xml.DecodeElement(&record, &se)

	   			//	if err != nil {
	   			///		return InfoRecord{}, err
	   			//	}
	   			cur = cur[e:]
	   		}
	   		return nil
	   	}
	*/
	x := xml.NewDecoder(lz)
	var token xml.Token
	for {
		token, err = x.Token()

		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "info" {
				var res InfoRecord
				err := x.DecodeElement(&res, &se)

				if err != nil {
					return err
				}

				out <- res
			}
		}
	}

	return nil
}

/*
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
*/
