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

package machine

import (
	"bufio"
)

type DeviceHandler struct {
	Keyboard *bufio.Reader
	Display  *bufio.Writer
}

type MachineState struct {
	Registers [8]uint16
	Program uint16
	Procstat uint16
	Stack uint16
	Memory [1 << 16]uint16
}

type MachineDebugger interface {
	Step(mc *Machine)
	Read(addr uint16, mc *Machine)
	Write(addr uint16, mc *Machine)
}

type Machine struct {
	Devices  *DeviceHandler
	State    MachineState
	Debugger MachineDebugger
}
