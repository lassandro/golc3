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
	"os"

	"github.com/lassandro/golc3/pkg/assembler"
	"github.com/lassandro/golc3/pkg/machine"
)

type WatchpointType uint

type Watchpoint struct {
	Addr uint16
	Type WatchpointType
}

type Breakpoint struct {
	Addr uint16
}

type Debugger struct {
	Break bool

	Breakpoints []Breakpoint
	Watchpoints []Watchpoint

	Source   *os.File
	Binary   *os.File
	SymTable *assembler.SymTable

	HandleBreak func(*Debugger, *machine.Machine)
	HandleRead  func(uint16, *Debugger, *machine.Machine)
	HandleWrite func(uint16, *Debugger, *machine.Machine)
}
