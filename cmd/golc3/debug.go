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
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/lassandro/golc3/pkg/debugger"
	"github.com/lassandro/golc3/pkg/encoding"
	"github.com/lassandro/golc3/pkg/machine"
)

var lastcmd []string

func debugBreak(dbg *debugger.Debugger, args []string) {
	const usage = "break [add|list|remove]"

	if len(args) == 0 {
		args = append(args, "l")
	}

	cmd := args[0]
	args = args[1:]

	switch cmd {
	case "a", "add":
		const usage = "break add [0x####]"

		if len(args) != 1 {
			log.Println(usage)
			return
		}

		addr, err := encoding.DecodeHex(args[0])

		if err != nil {
			log.Println(err)
			return
		}

		exists := false

		for _, breakpoint := range dbg.Breakpoints {
			if breakpoint.Addr == addr {
				exists = true
				break
			}
		}

		if !exists {
			dbg.Breakpoints = append(
				dbg.Breakpoints,
				debugger.Breakpoint{addr},
			)

			fmt.Printf("Breakpoint added [%#04x]\n", addr)
		}

	case "l", "ls", "list":
		const usage = "break list"

		if len(args) != 0 {
			log.Println(usage)
			return
		}

		var fmtstring string
		{
			digits := math.Floor(math.Log10(float64(len(dbg.Breakpoints) + 1)))
			fmtstring = fmt.Sprintf("#%%0%dd: %%#x\n", int64(digits)+1)
		}

		for i, breakpoint := range dbg.Breakpoints {
			log.Printf(fmtstring, i, breakpoint.Addr)
		}

	case "r", "rm", "remove":
		const usage = "break remove [#]"

		if len(args) != 1 {
			log.Println(usage)
			return
		}

		i, err := strconv.ParseInt(args[0], 10, 64)

		if err != nil {
			log.Println(err)
			return
		}

		if i < 0 || i >= int64(len(dbg.Breakpoints)) {
			log.Println("Invalid breakpoint number")
			return
		}

		dbg.Breakpoints[i] = dbg.Breakpoints[len(dbg.Breakpoints)-1]
		dbg.Breakpoints = dbg.Breakpoints[:len(dbg.Breakpoints)-1]
		fmt.Printf("Breakpoint removed [%d]\n", i)

	case "clear":
		dbg.Breakpoints = make([]debugger.Breakpoint, 0)
		fmt.Println("Breakpoints reset")

	default:
		log.Printf("break: '%s' is not a valid command\n", args[0])
	}
}

func debugWatch(dbg *debugger.Debugger, args []string) {
	const usage = "watch [add|list|rm]"

	if len(args) == 0 {
		log.Println(usage)
		return
	}

	cmd := args[0]
	args = args[1:]

	switch cmd {
	case "a", "add":
		const usage = "watch add [0x####] [read|write|readwrite]"

		if len(args) != 2 {
			log.Println(usage)
			return
		}

		addr, err := encoding.DecodeHex(args[0])

		if err != nil {
			log.Println(err)
			return
		}

		var wtype debugger.WatchpointType

		switch args[1] {
		case "r", "read":
			wtype = debugger.ReadWatch
		case "w", "write":
			wtype = debugger.WriteWatch
		case "rw", "rwrite", "readwrite":
			wtype = debugger.ReadWriteWatch
		default:
			log.Println(usage)
			return
		}

		exists := false

		for _, watchpoint := range dbg.Watchpoints {
			if watchpoint.Addr == addr && watchpoint.Type == wtype {
				exists = true
				break
			}
		}

		if !exists {
			dbg.Watchpoints = append(
				dbg.Watchpoints,
				debugger.Watchpoint{addr, wtype},
			)

			var typename string
			switch wtype {
			case debugger.ReadWatch:
				typename = "R"
			case debugger.WriteWatch:
				typename = "W"
			case debugger.ReadWriteWatch:
				typename = "RW"
			}

			fmt.Printf("Watchpoint added [%#04x] (%s)\n", addr, typename)
		}

	case "l", "ls", "list":
		const usage = "watch list"

		if len(args) != 0 {
			log.Println(usage)
			return
		}

		var fmtstring string
		{
			digits := math.Floor(math.Log10(float64(len(dbg.Watchpoints) + 1)))
			fmtstring = fmt.Sprintf("#%%0%dd: %%#x %%s\n", int64(digits)+1)
		}

		for i, watchpoint := range dbg.Watchpoints {
			switch watchpoint.Type {
			case debugger.WriteWatch:
				log.Printf(fmtstring, i, watchpoint.Addr, "write")
			case debugger.ReadWatch:
				log.Printf(fmtstring, i, watchpoint.Addr, "read")
			case debugger.ReadWriteWatch:
				log.Printf(fmtstring, i, watchpoint.Addr, "rwrite")
			}
		}

	case "r", "rm", "remove":
		const usage = "watch rm [#]"

		if len(args) != 1 {
			log.Println(usage)
			return
		}

		i, err := strconv.ParseInt(args[0], 10, 64)

		if err != nil {
			log.Println(err)
			return
		}

		if i < 0 || i >= int64(len(dbg.Watchpoints)) {
			log.Println("Invalid breakpoint number")
			return
		}

		dbg.Watchpoints[i] = dbg.Watchpoints[len(dbg.Watchpoints)-1]
		dbg.Watchpoints = dbg.Watchpoints[:len(dbg.Watchpoints)-1]
		fmt.Printf("Watchpoint removed [%d]\n", i)

	case "clear":
		dbg.Watchpoints = make([]debugger.Watchpoint, 0)
		fmt.Println("Watchpoints reset")

	default:
		log.Printf("watch: '%s' is not a valid command\n", cmd)
	}
}

func debugReg(dbg *debugger.Debugger, mc *machine.MachineState, args []string) {
	const usage = "register [R#|PC|PS] [0x####]"

	if len(args) > 0 {
		if len(args) != 2 {
			log.Println(usage)
			return
		}

		value, err := encoding.DecodeHex(args[1])

		if err != nil {
			log.Println(err)
			return
		}

		args[0] = strings.ToUpper(args[0])

		switch args[0] {
		case "R0":
			mc.Registers[0] = value
		case "R1":
			mc.Registers[1] = value
		case "R2":
			mc.Registers[2] = value
		case "R3":
			mc.Registers[3] = value
		case "R4":
			mc.Registers[4] = value
		case "R5":
			mc.Registers[5] = value
		case "R6":
			mc.Registers[6] = value
		case "R7":
			mc.Registers[7] = value
		case "PC":
			mc.Program = value
		case "PS":
			mc.Procstat = value
		default:
			log.Println("Invalid regsiter")
			return
		}

		fmt.Printf("\033[1m%s:\033[0m %#04x\n", args[0], value)
	} else {
		for i, register := range mc.Registers {
			fmt.Printf("\033[1mR%d:\033[0m %#04x\t", i, register)
			if i == (len(mc.Registers)-1)/2 {
				fmt.Println()
			}
		}

		fmt.Println()
		fmt.Printf(
			"\033[1mPC:\033[0m %#04x\t\033[1mPS:\033[0m %#04x\n",
			mc.Program,
			mc.Procstat,
		)
	}
}

func debugSource(dbg *debugger.Debugger, mc *machine.MachineState, args []string) {
	const usage = "source [0x####|label] [#]"

	if len(args) > 2 {
		log.Println(usage)
		return
	}

	if dbg.SymTable == nil {
		fmt.Println("No symbol table loaded")
		return
	}

	var addr uint16 = mc.Program
	var size uint16 = 3
	var err error = nil

	if len(args) > 0 {
		isLabel := false
		for labelAddr, label := range dbg.SymTable.Labels {
			if label == args[0] {
				isLabel = true
				addr = labelAddr
				break
			}
		}

		if !isLabel {
			addr, err = encoding.DecodeHex(args[0])

			if err != nil {
				var value int64
				value, err = strconv.ParseInt(args[0], 10, 16)

				if err != nil {
					log.Println(err)
					return
				}

				addr = mc.Program
				size = uint16(value)
			}
		}
	}

	if len(args) > 1 {
		var value int64
		value, err = strconv.ParseInt(args[1], 10, 16)

		if err != nil {
			log.Println(err)
			return
		}

		size = uint16(value)
	}

	dbg.PrintSource(addr, size)
}

func debugLabels(dbg *debugger.Debugger, args []string) {
	const usage = "labels"

	if len(args) > 0 {
		fmt.Println(usage)
		return
	}

	if dbg.SymTable == nil {
		fmt.Println("No symbol table loaded")
		return
	}

	keys := make([]uint16, 0, len(dbg.SymTable.Labels))
	for addr, _ := range dbg.SymTable.Labels {
		keys = append(keys, addr)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	for _, addr := range keys {
		fmt.Printf(
			"\033[1m[%#04x]\033[0m %s\n", addr, dbg.SymTable.Labels[addr],
		)
	}
}

func debugJump(dbg *debugger.Debugger, mc *machine.MachineState, args []string) {
	const usage = "jump [0x####|label]"

	if len(args) != 1 {
		fmt.Println(usage)
		return
	}

	if addr, err := encoding.DecodeHex(args[0]); err == nil {
		mc.Program = addr

		fmt.Printf("\033[1mPC:\033[0m %#04x\n", addr)
	} else if dbg.SymTable != nil {
		for addr, label := range dbg.SymTable.Labels {
			if label == args[0] {
				mc.Program = addr
				fmt.Printf(
					"\033[1mPC:\033[0m %#04x \033[1;30m(%s)\033[0m\n",
					addr,
					label,
				)
				return
			}
		}

		fmt.Printf("Unable to find '%s'\n", args[0])
	} else {
		fmt.Println("No symbol table loaded")
	}
}

func debugMemory(dbg *debugger.Debugger, mc *machine.MachineState, args []string) {
	const usage = "memory [0x####|#] [#]"

	if len(args) > 2 {
		log.Println(usage)
		return
	}

	var size uint16 = 1
	var addr uint16 = mc.Program
	var err error

	if len(args) > 0 {
		addr, err = encoding.DecodeHex(args[0])

		if err != nil {
			var value int64
			value, err = strconv.ParseInt(args[0], 10, 16)

			if err != nil {
				log.Println(err)
				return
			}

			addr = mc.Program
			size = uint16(value)
		}
	}

	if len(args) > 1 {
		var value int64
		value, err = strconv.ParseInt(args[1], 10, 16)

		if err != nil {
			log.Println(err)
			return
		}

		size = uint16(value)
	}

	dbg.PrintMem(mc, addr, size)
}

func debugSet(dbg *debugger.Debugger, mc *machine.MachineState, args []string) {
	const usage = "set [0x####] [0x####]"

	if len(args) != 2 {
		log.Println(usage)
		return
	}

	var addr uint16
	var value uint16
	var err error

	addr, err = encoding.DecodeHex(args[0])

	if err != nil {
		log.Println(err)
		return
	}

	value, err = encoding.DecodeHex(args[1])

	if err != nil {
		log.Println(err)
		return
	}

	mc.Memory[addr] = value
	dbg.PrintMem(mc, addr, 1)
}

func debugREPL(dbg *debugger.Debugger, mc *machine.Machine) {
	exitRawTerm()
	defer enterRawTerm()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\033[1;30m(dbg)\033[0m ")

		if !scanner.Scan() {
			fmt.Println()
			shouldexit = true
			return
		}

		args := strings.Split(strings.TrimSpace(scanner.Text()), " ")

		if len(args[0]) == 0 {
			if len(lastcmd) == 0 {
				continue
			}
			args = lastcmd
		} else {
			lastcmd = make([]string, len(args))
			copy(lastcmd, args)
		}

		cmd := args[0]
		args = args[1:]

		switch cmd {
		case "b", "bp", "break", "breakpoint":
			debugBreak(dbg, args)

		case "w", "wp", "watch", "watchpoint":
			debugWatch(dbg, args)

		case "r", "reg", "register", "registers":
			debugReg(dbg, &mc.State, args)

		case "s", "src", "source":
			debugSource(dbg, &mc.State, args)

		case "l", "label", "labels":
			debugLabels(dbg, args)

		case "j", "jmp", "jump":
			debugJump(dbg, &mc.State, args)

		case "m", "mem", "memory":
			debugMemory(dbg, &mc.State, args)

		case "set":
			debugSet(dbg, &mc.State, args)

		case "c", "continue":
			dbg.Break = false
			return

		case "n", "next":
			dbg.Break = true
			return

		case "q", "quit", "exit":
			shouldexit = true
			return

		case "clear":
			fmt.Print("\033[H\033[2J")

		case "reset":
			mc.LoadBin(dbg.Source)

		default:
			fmt.Printf("error: '%s' is not a valid command\n", cmd)
		}
	}
}

func handleBreak(dbg *debugger.Debugger, mc *machine.Machine) {
	if !dbg.Break {
		fmt.Println()
		fmt.Println("Program stopped")
		dbg.PrintSource(mc.State.Program, 8)
	}
	debugREPL(dbg, mc)
}

func handleRead(addr uint16, dbg *debugger.Debugger, mc *machine.Machine) {
	fmt.Println()
	fmt.Println("Program stopped")
	dbg.PrintMem(&mc.State, addr, 1)
	debugREPL(dbg, mc)
}

func handleWrite(addr uint16, dbg *debugger.Debugger, mc *machine.Machine) {
	fmt.Println()
	fmt.Println("Program stopped")
	dbg.PrintMem(&mc.State, addr, 1)
	debugREPL(dbg, mc)
}
