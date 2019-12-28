//
// Copyright (c) 2019 Björn Kalkbrenner <terminar@cyberphoria.org>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
//
package main

import (
	"os"
	"bufio"
	"path"
	"time"
	"strconv"
	"errors"
)

var archiveStoragePath = ""
var archiveStorageFlat = false


type archiveStorage struct {
	dataFile *os.File
	data *bufio.Writer

	metaFile *os.File
	meta *bufio.Writer
}

func careDirectory(dirName string) error {
	src, err := os.Stat(dirName)

	if os.IsNotExist(err) {
			errDir := os.MkdirAll(dirName, 0755)
			if errDir != nil {
				return errDir
			}
			return nil
	}

	if src.Mode().IsRegular() {
		return errors.New("already exist as a file!")
	}

	return nil
}

func getArchiveFolder() (string, error) {
	if archiveStoragePath == "" {
		return "", errors.New("archive storage path not set")
	}

	if archiveStorageFlat {
		return archiveStoragePath, nil
	} else {
		return path.Join(archiveStoragePath,time.Now().Format("2006-01"),time.Now().Format("02")), nil
	}
}

func (a *archiveStorage) Open(fname string) error {
	var err error
	var t = time.Now()
	var fFolder string

	if fFolder, err = getArchiveFolder(); err != nil {
		return err
	}
	fName := fname + "." + strconv.FormatInt(t.Unix(), 10)

	if err := careDirectory(fFolder); err != nil {
		return err
	}
	fPath := path.Join(fFolder,fName)

	if a.dataFile, err = os.Create(fPath); err != nil {
		return err
	}
	a.data = bufio.NewWriter(a.dataFile)

	if a.metaFile, err = os.Create(fPath + ".meta"); err != nil {
		return err
	}
	a.meta = bufio.NewWriter(a.metaFile)

	a.Meta("DATAFILE=" + fName)
	a.Meta("TIME=" + t.String())

	return nil
}

func (a *archiveStorage) Close() error {
	if a.data != nil {
		if err := a.data.Flush(); err != nil {
			return err
		}
	}

	if a.dataFile != nil {
		if err := a.dataFile.Close(); err != nil {
			return err
		}
	}

	if a.meta != nil {
		if err := a.meta.Flush(); err != nil {
			return err
		}
	}

	if a.metaFile != nil {
		if err := a.metaFile.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (a *archiveStorage) Data(text string) {
	if a.data != nil {
		a.data.WriteString(text + "\n")
	}
}

func (a *archiveStorage) Meta(text string) {
	if a.meta != nil {
		a.meta.WriteString(text + "\n")
	}
}

func (a *archiveStorage) Flush() error {
	if a.data != nil {
		if err := a.data.Flush(); err != nil {
			return err
		}
	}
	if a.meta != nil {
		if err := a.meta.Flush(); err != nil {
			return err
		}
	}
	return nil
}
