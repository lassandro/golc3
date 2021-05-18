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

package assembler

const (
	TOKEN_NONE TokenType = iota
	TOKEN_IDENT
	TOKEN_DIRECTIVE
	TOKEN_STRING
	TOKEN_LITERAL
)

const (
	LITERAL_IMM5       LiteralType = 5
	LITERAL_OFFSET6                = 6
	LITERAL_TRAPVEC8               = 8
	LITERAL_PCOFFSET9              = 9
	LITERAL_PCOFFSET11             = 11
	LITERAL_WORD                   = 16
)

const (
	// Assembly Instructions
	INSTRUCTION_INVALID InstructionType = iota
	INSTRUCTION_ADD
	INSTRUCTION_AND
	INSTRUCTION_BR
	INSTRUCTION_BRn
	INSTRUCTION_BRz
	INSTRUCTION_BRp
	INSTRUCTION_BRnz
	INSTRUCTION_BRzp
	INSTRUCTION_BRnp
	INSTRUCTION_BRnzp
	INSTRUCTION_JMP
	INSTRUCTION_JMPT
	INSTRUCTION_JSR
	INSTRUCTION_JSRR
	INSTRUCTION_LD
	INSTRUCTION_LDI
	INSTRUCTION_LDR
	INSTRUCTION_LEA
	INSTRUCTION_NOT
	INSTRUCTION_RET
	INSTRUCTION_RTI
	INSTRUCTION_RTT
	INSTRUCTION_ST
	INSTRUCTION_STI
	INSTRUCTION_STR
	INSTRUCTION_TRAP

	// Trap Routines
	INSTRUCTION_GETC
	INSTRUCTION_OUT
	INSTRUCTION_PUTS
	INSTRUCTION_IN
	INSTRUCTION_PUTSP
	INSTRUCTION_HALT
)

const (
	DIRECTIVE_INVALID DirectiveType = iota
	DIRECTIVE_ORIG
	DIRECTIVE_FILL
	DIRECTIVE_BLKW
	DIRECTIVE_STRINGZ
	DIRECTIVE_END
)
