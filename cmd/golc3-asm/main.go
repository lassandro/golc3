// Copyright (C) 2021  Antonio Lassandro

// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.

// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for
// more details.

// You should have received a copy of the GNU General Public License along
// with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/lassandro/golc3/pkg/assembler"
)

var helpvar bool
var debugvar bool
var outvar string

const usage = "golc3-asm [-debug] [-o outfile] filename"

func init() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)
}

func init() {
	flag.BoolVar(&helpvar, "help", false, "Displays command usage")
	flag.BoolVar(
		&debugvar, "debug", false,
		"Specifies whether to generate debugging information as a symbol "+
			"table. The table will use the output filename with extension "+
			"'.lc3db'",
	)
	flag.StringVar(
		&outvar, "out", "",
		"Specifies a precise name for the output file, "+
			"overriding the default means of determining it",
	)
	flag.Parse()
}

func golc3_asm() int {
	if helpvar {
		fmt.Println(usage)
		flag.PrintDefaults()
		return 0
	}

	args := flag.Args()

	var infile string
	var input io.ReadSeeker

	if stat, _ := os.Stdin.Stat(); stat.Mode()&os.ModeCharDevice == 0 {
		input = os.Stdin
		log.SetPrefix("\033[1m<stdin>:\033[0m")

		if outvar == "" {
			outvar = "out.bin"
		}
	} else {
		if len(args) != 1 {
			log.Println(usage)
			return 1
		}

		file, err := os.Open(args[0])

		if err != nil {
			log.Println(err)
			return 1
		}

		defer file.Close()

		filename := filepath.Base(file.Name())

		if stat, err := file.Stat(); err != nil {
			log.Println(err)
			return 1
		} else {
			if stat.IsDir() {
				log.Printf("%s is not a valid LC3 assembly file", filename)
				return 1
			}
		}

		input = file
		infile = file.Name()
		log.SetPrefix(fmt.Sprintf("\033[1m%s:\033[0m", filename))

		if outvar == "" {
			outvar = strings.ReplaceAll(
				filename, filepath.Ext(filename), ".bin",
			)
		}
	}

	var symtable assembler.SymTable
	var symtarget *assembler.SymTable = nil

	if debugvar {
		if input != os.Stdin {
			var err error
			if symtable.Source, err = filepath.Abs(infile); err != nil {
				log.Println(err)
				symtable.Source = ""
			}
		}
		symtable.Symbols = make(map[uint16]int64)
		symtable.Labels = make(map[uint16]string)
		symtarget = &symtable
	}

	result, errs := assembler.AssembleLC3Source(input, symtarget)

	if len(errs) > 0 {

		if input == os.Stdin {
			for _, err := range errs {
				log.Println(err)
			}
		} else {
			for _, err := range errs {
				if tokenErr, ok := err.(assembler.TokenError); ok {
					cursor := tokenErr.GetPosition()

					if _, err := input.Seek(
						cursor.LineByte, os.SEEK_SET,
					); err != nil {
						panic(err)
					}

					line, _ := bufio.NewReader(input).ReadString('\n')

					underlinefmt := fmt.Sprintf(
						"%% %ds%s",
						int(cursor.Byte-cursor.LineByte)+1,
						strings.Repeat("~", int(cursor.Size)-1),
					)

					log.Printf(
						"%s\n%s\n\033[31m%s\033[0m",
						err,
						line[:len(line)-1],
						fmt.Sprintf(underlinefmt, "^"),
					)
				} else {
					log.Println(err)
				}
			}
		}

		return 1
	}

	{
		buffer := new(bytes.Buffer)

		if err := binary.Write(buffer, binary.BigEndian, result); err != nil {
			log.Println("Error writing output file")
			log.Println(err)
			return 1
		}

		if err := os.WriteFile(outvar, buffer.Bytes(), 0666); err != nil {
			log.Println("Error writing output file")
			log.Println(err)
			return 1
		}
	}

	if debugvar {
		filename := filepath.Dir(outvar) + "/" + strings.ReplaceAll(
			filepath.Base(outvar), filepath.Ext(outvar), ".lc3db",
		)

		if file, err := os.OpenFile(
			filename, os.O_WRONLY|os.O_CREATE, 0666,
		); err == nil {
			if err := gob.NewEncoder(file).Encode(symtable); err != nil {
				log.Println("Error writing symbol table")
				log.Println(err)
				return 1
			}

			file.Close()
		} else {
			log.Println("Error creating symbol table")
			log.Println(err)
			return 1
		}
	}

	return 0
}

func main() {
	os.Exit(golc3_asm())
}
