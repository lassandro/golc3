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
	"encoding/binary"
	"errors"
	"io"

	"github.com/lassandro/golc3/pkg/encoding"
)

func (mc *MachineState) Reset() {
	for i, _ := range mc.Registers {
		mc.Registers[i] = 0x0000
	}

	for i, _ := range mc.Memory {
		mc.Memory[i] = 0x0000
	}

	// Program begins in the supervisor memory space with supervisor privilege
	mc.Program = MEMSPACE_SUPERVISOR
	mc.Procstat = 0x8000

	// R6 is SSP, USP is saved in state
	mc.Registers[6] = MEMSPACE_USER
	mc.Stack = MEMSPACE_DEVICES
}

func (mc *Machine) LoadBin(reader io.Reader) error {
	mc.State.Reset()

	scratch := make([]byte, 2)
	index := 0

	for index < (1<<16)-1 {
		n, err := reader.Read(scratch)

		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		} else if n != 2 {
			return errors.New("Error reading binary")
		}

		mc.State.Memory[index] = binary.BigEndian.Uint16(scratch)
		index++
	}

	return nil
}

func (mc *Machine) push(value uint16) {
	mc.State.Registers[6] -= 2
	mc.write(mc.State.Registers[6], value)
}

func (mc *Machine) pop() uint16 {
	result := mc.read(mc.State.Registers[6])
	mc.State.Registers[6] += 2
	return result
}

func (mc *Machine) read(addr uint16) uint16 {
	if addr == DEV_KBSR {
		var key byte
		var err error

		if mc.Devices != nil && mc.Devices.Keyboard != nil {
			key, err = mc.Devices.Keyboard.ReadByte()
			if err != nil && err != io.EOF {
				panic(err)
			}

		} else {
			err = io.EOF
		}

		if err != io.EOF {
			mc.State.Memory[DEV_KBSR] = 1 << 15
			mc.State.Memory[DEV_KBDR] = uint16(key)
		} else {
			mc.State.Memory[DEV_KBSR] = 0
		}
	} else if addr == DEV_DSR {
		if mc.Devices != nil && mc.Devices.Display != nil {
			if mc.Devices.Display.Available() > 0 {
				mc.State.Memory[DEV_DSR] = 1 << 15
			} else {
				mc.State.Memory[DEV_DSR] = 0
			}
		} else {
			mc.State.Memory[DEV_DSR] = 0
		}
	}

	if mc.Debugger != nil {
		mc.Debugger.Read(addr, mc)
	}

	if addr != DEV_DDR {
		return mc.State.Memory[addr]
	} else {
		return 0
	}
}

func (mc *Machine) write(addr uint16, value uint16) {
	if addr == DEV_DDR {
		err := mc.Devices.Display.WriteByte(byte(value & 0xFF))

		if err != nil {
			panic(err)
		}

		err = mc.Devices.Display.Flush()

		if err != nil {
			panic(err)
		}
	}

	if addr != DEV_KBDR {
		mc.State.Memory[addr] = value
	}

	if mc.Debugger != nil {
		mc.Debugger.Write(addr, mc)
	}
}

func (mc *Machine) setPrivilege(privileged bool) {
	if privileged != mc.getPrivilege() {
		// Swap USP/SSP
		currentStack := mc.State.Registers[6]
		mc.State.Registers[6] = mc.State.Stack
		mc.State.Stack = currentStack
	}

	if privileged {
		// Enable privilege bit, but preserve priority and condition bits
		mc.State.Procstat |= uint16(0x1 << 15)
	} else {
		// Reset privilege bit, but preserve priority and condition bits
		mc.State.Procstat &= ^uint16(0x1 << 15)
	}
}

func (mc *Machine) getPrivilege() bool {
	return mc.State.Procstat>>15 == 1
}

func (mc *Machine) setPriority(value uint8) {
	if value > 0x7 {
		panic("Invalid priority value")
	}

	mc.State.Procstat &= ^uint16(0x7 << 8)
	mc.State.Procstat |= uint16(value&0x7) << 8
}

func (mc *Machine) getPriority() uint8 {
	return uint8((mc.State.Procstat >> 8) & 0x7)
}

func (mc *Machine) raiseException(vector uint8, priority uint8) {
	mc.push(mc.State.Procstat)
	mc.push(mc.State.Program)
	mc.setPriority(priority)
	mc.setPrivilege(true)
	mc.State.Program = mc.read(MEMSPACE_INT_TABLE | uint16(vector))
}

func (mc *Machine) setFlags(value uint16) {
	// Reset condition flags, but preserve privilege and priority bits
	mc.State.Procstat &= ^uint16(0x7)

	if value == 0 {
		mc.State.Procstat |= FLAG_ZERO
	} else if value>>15 == 1 {
		mc.State.Procstat |= FLAG_NEG
	} else {
		mc.State.Procstat |= FLAG_POS
	}
}

func (mc *Machine) Step() {
	instruction := mc.read(mc.State.Program)
	opcode := instruction >> 12

	mc.State.Program++

	switch opcode {
	// ADD  |0001    |DR   |SR1  |0|00 |SR2   | Register  addition
	// ADD  |0001    |DR   |SR1  |1|imm5      | Immediate addition
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_ADD:
		dest := (instruction >> 9) & 0x7
		src1 := (instruction >> 6) & 0x7

		// Immediate value addition
		if (instruction>>5)&0x1 == 1 {
			imm5 := encoding.SignExtend(instruction&0x1F, 5)

			mc.State.Registers[dest] = mc.State.Registers[src1] + imm5
		} else {
			src2 := (instruction & 0x7)

			mc.State.Registers[dest] = mc.State.Registers[src1] +
				mc.State.Registers[src2]
		}

		mc.setFlags(mc.State.Registers[dest])

	// AND  |0101    |DR   |SR1  |0|00 |SR2   | Register  bitwise
	// AND  |0101    |DR   |SR1  |1|imm5      | Immediate bitwise
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_AND:
		dest := (instruction >> 9) & 0x7
		src1 := (instruction >> 6) & 0x7

		// Immediate value addition
		if (instruction>>5)&0x1 == 1 {
			imm5 := encoding.SignExtend(instruction&0x1F, 5)

			mc.State.Registers[dest] = mc.State.Registers[src1] & imm5
		} else {
			src2 := (instruction & 0x3)

			mc.State.Registers[dest] = mc.State.Registers[src1] &
				mc.State.Registers[src2]
		}

		mc.setFlags(mc.State.Registers[dest])

	// BR   |0000    |N|Z|P|PCoffset9         | Conditional branch
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_BR:
		flags := (instruction >> 9) & 0x7

		if flags == 0 || flags&(mc.State.Procstat&0x7) > 0 {
			mc.State.Program += encoding.SignExtend(instruction&0x1FF, 9)
		}

	// JMP  |1100    |000  |BaseR|000000      | Jump
	// JMPT |1100    |000  |BaseR|000001      | Jump (Clear Privilege)
	// RET  |1100    |000  |111  |000000      | Return
	// RTT  |1100    |000  |111  |000001      | Return (Clear Privilege)
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_JMP:
		src := (instruction >> 6) & 0x7

		mc.State.Program = mc.State.Registers[src]

		if instruction&0x1 == 1 {
			if mc.getPrivilege() {
				mc.setPrivilege(false)
			} else {
				// 0x00 Privilege Violation Vector -> 0x0100 Interrupt Addr
				mc.raiseException(0x00, mc.getPriority())
			}
		}

	// JSR  |0100    |1|PCoffset11            | Jump to subroutine
	// JSRR |0100    |0|00 |BaseR|000000      | Jump to subroutine register
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_JSR:
		mc.State.Registers[7] = mc.State.Program

		if (instruction>>11)&0x1 == 1 {
			mc.State.Program += encoding.SignExtend(instruction&0x7FF, 11)
		} else {
			src := (instruction >> 6) & 0x7

			mc.State.Program = mc.State.Registers[src]
		}

	// LD   |0010    |DR   |PCoffset9         | Load
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_LD:
		dest := (instruction >> 9) & 0x7
		addr := mc.State.Program + encoding.SignExtend(instruction&0x1FF, 9)

		mc.State.Registers[dest] = mc.read(addr)

		mc.setFlags(mc.State.Registers[dest])

	// LDI  |1010    |DR   |PCoffset9         | Load indirect
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_LDI:
		dest := (instruction >> 9) & 0x7
		addr := mc.State.Program + encoding.SignExtend(instruction&0x1FF, 9)

		mc.State.Registers[dest] = mc.read(mc.read(addr))

		mc.setFlags(mc.State.Registers[dest])

	// LDR  |0110    |DR   |BaseR|offset6     | Load base+offset
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_LDR:
		dest := (instruction >> 9) & 0x7
		src := (instruction >> 6) & 0x7
		addr := mc.State.Registers[src] +
			encoding.SignExtend(instruction&0x3F, 6)

		mc.State.Registers[dest] = mc.read(addr)

		mc.setFlags(mc.State.Registers[dest])

	// LEA  |1110    |DR   |PCoffset9         | Load effective address
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_LEA:
		dest := (instruction >> 9) & 0x7
		addr := mc.State.Program + encoding.SignExtend(instruction&0x1FF, 9)

		mc.State.Registers[dest] = addr

		mc.setFlags(mc.State.Registers[dest])

	// NOT  |1001    |DR   |SR   |1|11111     | Bitwise complement
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_NOT:
		dest := (instruction >> 9) & 0x7
		src := (instruction >> 6) & 0x7

		mc.State.Registers[dest] = ^mc.State.Registers[src]

		mc.setFlags(mc.State.Registers[dest])

	// RTI  |1000    |000000000000            | Return from interrupt
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_RTI:
		if mc.getPrivilege() {
			mc.setPrivilege(false)
			mc.State.Program = mc.pop()
			mc.State.Procstat = mc.pop()
		} else {
			// 0x00 Privilege Violation Vector -> 0x0100 Interrupt Addr
			mc.raiseException(0x00, mc.getPriority())
		}

	// ST   |0011    |SR   |PCoffset9         | Store
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_ST:
		src := (instruction >> 9) & 0x7
		addr := mc.State.Program + encoding.SignExtend(instruction&0x1FF, 9)

		mc.write(addr, mc.State.Registers[src])

	// STI  |1011    |SR   |PCoffset9         | Store indirect
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_STI:
		src := (instruction >> 9) & 0x7
		addr := mc.State.Program + encoding.SignExtend(instruction&0x1FF, 9)

		mc.write(mc.read(addr), mc.State.Registers[src])

	// STR  |0111    |SR   |BaseR|offset6     | Store base+offset
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_STR:
		src := (instruction >> 9) & 0x7
		dest := (instruction >> 6) & 0x7
		addr := mc.State.Registers[dest] +
			encoding.SignExtend(instruction&0x3F, 6)

		mc.write(addr, mc.State.Registers[src])

	// TRAP |1111    |0000   |trapvect8       | Store base+offset
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	case OP_TRAP:
		call := instruction & 0xFF

		mc.setPrivilege(true)
		mc.State.Registers[7] = mc.State.Program
		mc.State.Program = mc.read(encoding.ZeroExtend(call, 8))

	// RES  |1101    |                        | Reserved (illegal)
	// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
	default:
		// 0x01 Illegal Opcode Vector -> 0x0101 Interrupt Addr
		mc.raiseException(0x01, mc.getPriority())
	}

	if mc.Devices != nil && mc.Devices.Keyboard != nil {
		_, err := mc.Devices.Keyboard.Peek(1)
		if err == nil && mc.getPriority() < 0x4 {
			// 0x80 Keyboard Interrupt Vector -> 0x0180 Interrupt Addr
			mc.raiseException(0x80, 4)
		}
	}

	if mc.Debugger != nil {
		mc.Debugger.Step(mc)
	}
}
