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
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/lassandro/golc3/pkg/assembler"
	"github.com/lassandro/golc3/pkg/debugger"
	"github.com/lassandro/golc3/pkg/machine"
)

var helpvar bool
var debugvar bool
var shouldexit bool

const usage = "golc3 filename"

func init() {
	exe, _ := os.Executable()
	log.SetFlags(0)
	log.SetPrefix(fmt.Sprintf("%s: ", filepath.Base(exe)))
	log.SetOutput(os.Stderr)
}

func init() {
	flag.BoolVar(&helpvar, "help", false, "Displays command usage")
	flag.BoolVar(&debugvar, "debug", false, "Runs the machine in a debug CLI")
	flag.Parse()
}

func golc3() int {
	if helpvar {
		fmt.Println(usage)
		return 0
	}

	args := flag.Args()

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

	var mc machine.Machine
	var dh machine.DeviceHandler
	dh.Keyboard = bufio.NewReader(os.Stdin)
	dh.Display = bufio.NewWriter(os.Stdout)
	mc.Devices = &dh

	if debugvar {
		var dbg debugger.Debugger
		dbg.HandleBreak = handleBreak
		dbg.HandleRead = handleRead
		dbg.HandleWrite = handleWrite
		dbg.Binary = file
		mc.Debugger = &dbg

		filename := filepath.Dir(args[0]) + "/" + strings.ReplaceAll(
			filepath.Base(args[0]), filepath.Ext(args[0]), ".lc3db",
		)

		if file, err := os.Open(filename); err == nil {
			var symtable assembler.SymTable

			if err := gob.NewDecoder(file).Decode(&symtable); err == nil {
				dbg.SymTable = &symtable
			} else {
				log.Println("Error loading symbol file")
				log.Println(err)
			}

			file.Close()
		} else {
			log.Println("Error loading symbol file")
			log.Println(err)
		}

		if dbg.SymTable != nil && dbg.SymTable.Source != "" {
			if file, err := os.Open(dbg.SymTable.Source); err == nil {
				dbg.Source = file
				defer file.Close()
			} else {
				log.Println("Error loading source file")
				log.Println(err)
			}
		}

		c := make(chan os.Signal, 1)
		defer close(c)

		signal.Notify(c, os.Interrupt)
		go func() {
			for _ = range c {
				fmt.Println()
				dbg.Break = true
			}
		}()
	}

	if err := mc.LoadBin(file); err != nil {
		log.Println(err)
		return 1
	}

	enterRawTerm()
	defer exitRawTerm()

	if debugvar {
		debugREPL(mc.Debugger.(*debugger.Debugger), &mc)
	}

	for !shouldexit {
		mc.Step()
	}

	return 0
}

func main() {
	os.Exit(golc3())
}
