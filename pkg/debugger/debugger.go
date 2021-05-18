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

package debugger

import (
	"bufio"
	"fmt"
	"os"

	"github.com/lassandro/golc3/pkg/machine"
)

func (dbg *Debugger) Step(mc *machine.Machine) {
	if dbg.Break {
		dbg.HandleBreak(dbg, mc)
		return
	}

	for _, breakpoint := range dbg.Breakpoints {
		if mc.State.Program == breakpoint.Addr {
			dbg.HandleBreak(dbg, mc)
			break
		}
	}
}

func (dbg *Debugger) Read(addr uint16, mc *machine.Machine) {
	for _, watchpoint := range dbg.Watchpoints {
		if watchpoint.Type == WriteWatch {
			continue
		}

		if addr == watchpoint.Addr {
			dbg.HandleRead(addr, dbg, mc)
			break
		}
	}
}

func (dbg *Debugger) Write(addr uint16, mc *machine.Machine) {
	for _, watchpoint := range dbg.Watchpoints {
		if watchpoint.Type == ReadWatch {
			continue
		}

		if addr == watchpoint.Addr {
			dbg.HandleWrite(addr, dbg, mc)
			break
		}
	}
}

func (dbg *Debugger) PrintSource(addr uint16, count uint16) {
	if dbg.Source == nil {
		fmt.Println("No source file loaded")
		return
	}

	if dbg.SymTable == nil {
		fmt.Println("No symbol table loaded")
		return
	}

	if offset, exists := dbg.SymTable.Symbols[addr]; exists {
		if _, err := dbg.Source.Seek(offset, os.SEEK_SET); err != nil {
			panic(err)
		}

		scanner := bufio.NewScanner(dbg.Source)
		scanner.Split(bufio.ScanLines)

		for i := uint16(0); i < count; i++ {
			if !scanner.Scan() {
				break
			}

			line := scanner.Text()

			foundaddr := false
			for lineaddr, linebyte := range dbg.SymTable.Symbols {
				if linebyte == offset {
					fmt.Printf("\033[1m[%#04x]\033[0m ", lineaddr)
					foundaddr = true
					break
				}
			}

			if !foundaddr {
				fmt.Print("\033[1;30m~~~~~~~~\033[0m ")
			}

			fmt.Println(line)

			offset += int64(len(line) + 1)
		}

		if err := scanner.Err(); err != nil {
			fmt.Println(err)
		}
	} else {
		fmt.Printf("No instruction found at %#04x\n", addr)
	}
}

func (dbg *Debugger) PrintMem(mc *machine.MachineState, addr, count uint16) {
	for i := addr; i < addr+count; i++ {
		if i == addr {
			fmt.Printf("\033[1m[%#04x]\033[0m ", i)
		} else if (i-addr)%4 == 0 {
			fmt.Println()
			fmt.Printf("\033[1m[%#04x]\033[0m ", i)
		}

		result := mc.Memory[i]

		if result == 0 {
			fmt.Printf("\033[1;30m%#04x\033[0m ", result)
		} else {
			fmt.Printf("%#04x ", result)
		}
	}

	fmt.Println()
}
