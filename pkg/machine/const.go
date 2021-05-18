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

const (
	FLAG_POS  uint16 = 1 << 0
	FLAG_ZERO uint16 = 1 << 1
	FLAG_NEG  uint16 = 1 << 2
)

const (
	TRAP_GETC  uint16 = 0x20
	TRAP_OUT   uint16 = 0x21
	TRAP_PUTS  uint16 = 0x22
	TRAP_IN    uint16 = 0x23
	TRAP_PUTSP uint16 = 0x24
	TRAP_HALT  uint16 = 0x25
)

const (
	MEMSPACE_TRAP_TABLE uint16 = 0x0000
	MEMSPACE_INT_TABLE         = 0x0100
	MEMSPACE_SUPERVISOR        = 0x0200
	MEMSPACE_USER              = 0x3000
	MEMSPACE_DEVICES           = 0xFE00
)

const (
	DEV_KBSR uint16 = 0xFE00
	DEV_KBDR        = 0xFE02
	DEV_DSR         = 0xFE04
	DEV_DDR         = 0xFE06
)

const (
	OP_ADD  uint16 = 0b0001
	OP_AND  uint16 = 0b0101
	OP_BR   uint16 = 0b0000
	OP_JMP  uint16 = 0b1100
	OP_JSR  uint16 = 0b0100
	OP_LD   uint16 = 0b0010
	OP_LDI  uint16 = 0b1010
	OP_LDR  uint16 = 0b0110
	OP_LEA  uint16 = 0b1110
	OP_NOT  uint16 = 0b1001
	OP_RTI  uint16 = 0b1000
	OP_ST   uint16 = 0b0011
	OP_STI  uint16 = 0b1011
	OP_STR  uint16 = 0b0111
	OP_TRAP uint16 = 0b1111

	// Reserved
	OP_RES uint16 = 0b1101
)
