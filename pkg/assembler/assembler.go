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

import (
	"bufio"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/lassandro/golc3/pkg/encoding"
)

func parseDirective(ident string) DirectiveType {
	if strings.EqualFold(ident, ".ORIG") {
		return DIRECTIVE_ORIG
	} else if strings.EqualFold(ident, ".FILL") {
		return DIRECTIVE_FILL
	} else if strings.EqualFold(ident, ".BLKW") {
		return DIRECTIVE_BLKW
	} else if strings.EqualFold(ident, ".STRINGZ") {
		return DIRECTIVE_STRINGZ
	} else if strings.EqualFold(ident, ".END") {
		return DIRECTIVE_END
	}

	return DIRECTIVE_INVALID
}

func parseInstruction(ident string) InstructionType {
	if strings.EqualFold(ident, "ADD") {
		return INSTRUCTION_ADD
	} else if strings.EqualFold(ident, "AND") {
		return INSTRUCTION_AND
	} else if strings.EqualFold(ident, "BR") {
		return INSTRUCTION_BR
	} else if strings.EqualFold(ident, "BRn") {
		return INSTRUCTION_BRn
	} else if strings.EqualFold(ident, "BRz") {
		return INSTRUCTION_BRz
	} else if strings.EqualFold(ident, "BRp") {
		return INSTRUCTION_BRp
	} else if strings.EqualFold(ident, "BRnz") {
		return INSTRUCTION_BRnz
	} else if strings.EqualFold(ident, "BRzp") {
		return INSTRUCTION_BRzp
	} else if strings.EqualFold(ident, "BRnp") {
		return INSTRUCTION_BRnp
	} else if strings.EqualFold(ident, "BRnzp") {
		return INSTRUCTION_BRnzp
	} else if strings.EqualFold(ident, "JMP") {
		return INSTRUCTION_JMP
	} else if strings.EqualFold(ident, "JMPT") {
		return INSTRUCTION_JMPT
	} else if strings.EqualFold(ident, "JSR") {
		return INSTRUCTION_JSR
	} else if strings.EqualFold(ident, "JSRR") {
		return INSTRUCTION_JSRR
	} else if strings.EqualFold(ident, "LD") {
		return INSTRUCTION_LD
	} else if strings.EqualFold(ident, "LDI") {
		return INSTRUCTION_LDI
	} else if strings.EqualFold(ident, "LDR") {
		return INSTRUCTION_LDR
	} else if strings.EqualFold(ident, "LEA") {
		return INSTRUCTION_LEA
	} else if strings.EqualFold(ident, "NOT") {
		return INSTRUCTION_NOT
	} else if strings.EqualFold(ident, "RET") {
		return INSTRUCTION_RET
	} else if strings.EqualFold(ident, "RTI") {
		return INSTRUCTION_RTI
	} else if strings.EqualFold(ident, "RTT") {
		return INSTRUCTION_RTT
	} else if strings.EqualFold(ident, "ST") {
		return INSTRUCTION_ST
	} else if strings.EqualFold(ident, "STI") {
		return INSTRUCTION_STI
	} else if strings.EqualFold(ident, "STR") {
		return INSTRUCTION_STR
	} else if strings.EqualFold(ident, "TRAP") {
		return INSTRUCTION_TRAP
	} else if strings.EqualFold(ident, "GETC") {
		return INSTRUCTION_GETC
	} else if strings.EqualFold(ident, "OUT") {
		return INSTRUCTION_OUT
	} else if strings.EqualFold(ident, "PUTS") {
		return INSTRUCTION_PUTS
	} else if strings.EqualFold(ident, "IN") {
		return INSTRUCTION_IN
	} else if strings.EqualFold(ident, "PUTSP") {
		return INSTRUCTION_PUTSP
	} else if strings.EqualFold(ident, "HALT") {
		return INSTRUCTION_HALT
	}

	return INSTRUCTION_INVALID
}

func parseLiteral(token *Token, bits LiteralType) (uint16, error) {
	if strings.ContainsAny(token.Value, "xX") {
		result, err := encoding.DecodeHex(token.Value)

		if err != nil {
			return 0, &InvalidLiteralError{token.Position}
		}

		if bits < 16 {
			limit := uint16(1) << bits

			if result >= limit {
				return 0, &OversizedLiteralError{token.Position, limit, result}
			}

			if (result & limit) != 0 {
				result = result | ((1 << uint16(bits)) - 1)
			}
		}

		return result, nil
	} else {
		result, err := encoding.DecodeInt(token.Value)

		if err != nil {
			return 0, &InvalidLiteralError{token.Position}
		}

		if bits < 16 {
			limit := (int16(1) << bits) - 1

			if result < -limit || result >= limit {
				return 0, &OversizedLiteralError{token.Position, limit, result}
			}

			if (result&(1<<bits) - 1) != 0 {
				result = result & ((int16(1) << bits) - 1)
			}
		}

		return uint16(result), nil
	}
}

func parseRegister(token *Token) (uint16, bool) {
	ident := token.Value

	if strings.EqualFold(ident, "R0") {
		return 0, true
	} else if strings.EqualFold(ident, "R1") {
		return 1, true
	} else if strings.EqualFold(ident, "R2") {
		return 2, true
	} else if strings.EqualFold(ident, "R3") {
		return 3, true
	} else if strings.EqualFold(ident, "R4") {
		return 4, true
	} else if strings.EqualFold(ident, "R5") {
		return 5, true
	} else if strings.EqualFold(ident, "R6") {
		return 6, true
	} else if strings.EqualFold(ident, "R7") {
		return 7, true
	}

	return 0, false
}

func AssembleLC3Source(input io.ReadSeeker, symtable *SymTable) (result []uint16, errs []error) {
	type LabelRef struct {
		Label    string
		Addr     uint16
		Size     LiteralType
		Position Cursor
	}

	type FillRef struct {
		Label    string
		Addr     uint16
		Position Cursor
	}

	var labels = make(map[string]uint16)
	var labelRefs []LabelRef
	var fillRefs []FillRef

	var program uint32 = 0

	var builder strings.Builder
	var scanner = bufio.NewScanner(input)

	var cursor = Cursor{Line: 1, Column: 0, Size: 0, Byte: 0}

	result = make([]uint16, 1<<16)
	errs = make([]error, 0)

	// Process:
	// - Parse line
	// - Assemble line
	for scanner.Scan() {
		var tokens = make([]Token, 0, 5)
		var tokenStart int = 0
		var tokenType TokenType = TOKEN_NONE

		var lineErrs = len(errs)

		line := scanner.Text()
		builder.Grow(len(line))

		cursor.Size = int64(len(line))

		// Parse Line:
		// - Gather tokens and their types
		// - Check for syntax errors
		for column, char := range line {
			cursor.Column = column + 1

			var flush bool = false
			var skip bool = false

			if tokenType == TOKEN_NONE {
				tokenStart = cursor.Column
			}

			switch {
			// Whitespace
			case unicode.IsSpace(char):
				if tokenType == TOKEN_NONE {
					continue
				} else if tokenType != TOKEN_STRING {
					flush = true
				}

			// Comments
			case char == ';':
				if tokenType == TOKEN_NONE {
					skip = true
				} else if tokenType != TOKEN_STRING {
					flush = true
					skip = true
				}

			// Assembler Directives
			case char == '.':
				if tokenType == TOKEN_NONE {
					tokenType = TOKEN_DIRECTIVE
				} else if tokenType != TOKEN_STRING {
					errs = append(errs, &UnexpectedCharacterError{cursor, char})
				}

			// Operand Separator
			case char == ',':
				if tokenType != TOKEN_STRING {
					flush = true
				}

			// Hex Literal (i.e. x2A, no leading zero)
			case char == 'x' || char == 'X':
				if tokenType == TOKEN_NONE {
					tokenType = TOKEN_LITERAL
				}

			// Base 10 Literal (i.e. #42)
			case char == '#':
				if tokenType == TOKEN_NONE {
					tokenType = TOKEN_LITERAL
				} else if tokenType != TOKEN_STRING {
					errs = append(errs, &UnexpectedCharacterError{cursor, char})
				}

			// String Literal
			case char == '"':
				if tokenType == TOKEN_NONE {
					tokenType = TOKEN_STRING
				} else if tokenType == TOKEN_STRING {
					flush = true
				} else {
					errs = append(errs, &UnexpectedCharacterError{cursor, char})
				}

			// Numeric Literal
			case unicode.IsDigit(char):
				if tokenType == TOKEN_NONE {
					tokenType = TOKEN_LITERAL
				}

			// Numeric Sign
			case char == '-':
				if tokenType != TOKEN_LITERAL {
					errs = append(errs, &UnexpectedCharacterError{cursor, char})
				}

			// Underscore'd Identifier
			case char == '_':
				if tokenType == TOKEN_NONE {
					tokenType = TOKEN_IDENT
				} else if tokenType != TOKEN_IDENT && tokenType != TOKEN_STRING {
					errs = append(errs, &UnexpectedCharacterError{cursor, char})
				}

			// Identifier
			case unicode.IsLetter(char):
				if char > unicode.MaxASCII {
					errs = append(errs, &OversizedCharacterError{cursor})
				}

				if tokenType == TOKEN_NONE {
					tokenType = TOKEN_IDENT
				}

			default:
				if char > unicode.MaxASCII {
					errs = append(errs, &OversizedCharacterError{cursor})
				}

				if tokenType != TOKEN_STRING {
					errs = append(
						errs, &UnexpectedCharacterError{cursor, char},
					)
				}
			}

			if cursor.Column == len(line) {
				if tokenType == TOKEN_STRING {
					if char != '"' || tokenStart == cursor.Column {
						errs = append(errs, &InvalidStringError{cursor})
					}
				} else {
					if char == ',' {
						errs = append(
							errs, &UnexpectedCharacterError{cursor, char},
						)
					}
				}

				flush = true
				builder.WriteRune(char)
			} else {
				if flush && tokenType == TOKEN_STRING && char == '"' {
					builder.WriteRune(char)
				}
			}

			if flush {
				if builder.Len() > 0 {
					var token Token
					token.Position = Cursor{
						Line:     cursor.Line,
						Column:   tokenStart,
						Byte:     cursor.Byte + int64(tokenStart-1),
						Size:     int64(builder.Len()),
						LineByte: cursor.Byte,
					}
					token.Type = tokenType
					token.Value = builder.String()
					tokens = append(tokens, token)
					builder.Reset()
				}

				flush = false
				tokenType = TOKEN_NONE
			} else if !skip {
				builder.WriteRune(char)
			}

			if skip {
				break
			}
		}

		if len(tokens) == 0 {
			cursor.Line++
			cursor.Byte += int64(len(line) + 1)
			cursor.LineByte += int64(len(line) + 1)
			continue
		}

		// Pass any potential assembler errors if we already had parser errors
		if len(errs) > lineErrs {
			cursor.Line++
			cursor.Byte += int64(len(line) + 1)
			cursor.LineByte += int64(len(line) + 1)
			continue
		}

		// Assemble line
		// - Write instruction bits to result
		// - Save label refs for unknown labels
		// - Type check instruction arguments
		var label *Token = nil
		var directive DirectiveType
		var instruction InstructionType
		var keyword *Token = nil
		var operands []Token

		var scratch uint16 = 0

		if instruction = parseInstruction(tokens[0].Value); instruction != INSTRUCTION_INVALID {
			keyword = &tokens[0]

			if len(tokens) > 1 {
				operands = tokens[1:]
			}
		} else if directive = parseDirective(tokens[0].Value); directive != DIRECTIVE_INVALID {
			keyword = &tokens[0]

			if len(tokens) > 1 {
				operands = tokens[1:]
			}
		} else {
			label = &tokens[0]
		}

		if label != nil {
			if _, exists := labels[label.Value]; !exists {
				labels[label.Value] = uint16(program)
			} else {
				errs = append(
					errs, &RedeclaredLabelError{label.Position, label.Value},
				)
			}

			// No need to assemble label-only statements
			if len(tokens) == 1 {
				cursor.Line++
				cursor.Byte += int64(len(line) + 1)
				cursor.LineByte += int64(len(line) + 1)
				continue
			}

			if instruction = parseInstruction(tokens[1].Value); instruction != INSTRUCTION_INVALID {
				keyword = &tokens[1]

				if len(tokens) > 2 {
					operands = tokens[2:]
				}
			} else if directive = parseDirective(tokens[1].Value); directive != DIRECTIVE_INVALID {
				keyword = &tokens[1]

				if len(tokens) > 2 {
					operands = tokens[2:]
				}
			}
		}

		if keyword == nil {
			errs = append(
				errs,
				&UnknownIdentifierError{tokens[0].Position, tokens[0].Value},
			)
		}

		if directive == DIRECTIVE_END {
			if count := len(operands); count != 0 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 0, count},
				)
			}

			break
		}

		switch directive {
		// .FILL #
		case DIRECTIVE_FILL:
			if count := len(operands); count != 1 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 1, count},
				)

				break
			}

			if operands[0].Type == TOKEN_LITERAL {
				literal, err := parseLiteral(
					&operands[0], LITERAL_WORD,
				)

				if err != nil {
					errs = append(errs, err)
				}

				result[program] = literal
			} else if operands[0].Type == TOKEN_IDENT {
				addr, exists := labels[operands[0].Value]

				if exists {
					result[program] = addr
				} else {
					fillRefs = append(
						fillRefs,
						FillRef{
							operands[0].Value,
							uint16(program),
							operands[0].Position,
						},
					)
				}
			} else {
				errs = append(
					errs,
					&InvalidOperandError{
						operands[0].Position,
						[]TokenType{TOKEN_LITERAL, TOKEN_IDENT},
						operands[0].Type,
					},
				)
			}

			program++

		// .BLKW #
		case DIRECTIVE_BLKW:
			if count := len(operands); count != 1 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 1, count},
				)

				break
			}

			if operands[0].Type != TOKEN_LITERAL {
				errs = append(
					errs,
					&InvalidOperandError{
						operands[0].Position,
						[]TokenType{TOKEN_LITERAL},
						operands[0].Type,
					},
				)

				break
			}

			literal, err := parseLiteral(
				&operands[0], LITERAL_WORD,
			)

			if err != nil {
				errs = append(errs, err)
			}

			program += uint32(literal)

		// .STRINGZ "..."
		case DIRECTIVE_STRINGZ:
			if count := len(operands); count != 1 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 1, count},
				)

				break
			}

			if operands[0].Type != TOKEN_STRING {
				errs = append(
					errs,
					&InvalidOperandError{
						operands[0].Position,
						[]TokenType{TOKEN_STRING},
						operands[0].Type,
					},
				)

				break
			}

			s, err := strconv.Unquote(operands[0].Value)

			if err != nil {
				errs = append(errs, &InvalidStringError{operands[0].Position})
			}

			for _, c := range s {
				result[program] = uint16(c)
				program++
			}

			result[program] = 0
			program++

		// .ORIG #
		case DIRECTIVE_ORIG:
			if count := len(operands); count != 1 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 1, count},
				)

				break
			}

			if operands[0].Type != TOKEN_LITERAL {
				errs = append(
					errs,
					&InvalidOperandError{
						operands[0].Position,
						[]TokenType{TOKEN_LITERAL},
						operands[0].Type,
					},
				)

				break
			}

			literal, err := parseLiteral(&operands[0], LITERAL_WORD)

			if err != nil {
				errs = append(errs, err)
			}

			program = uint32(literal)
		}

		switch instruction {
		// ADD  |0001    |DR   |SR1  |0|00 |SR2   | Register  addition
		// ADD  |0001    |DR   |SR1  |1|imm5      | Immediate addition
		// AND  |0101    |DR   |SR1  |0|00 |SR2   | Register  bitwise
		// AND  |0101    |DR   |SR1  |1|imm5      | Immediate bitwise
		// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
		case INSTRUCTION_ADD, INSTRUCTION_AND:
			if count := len(operands); count != 3 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 3, count},
				)

				break
			}

			if instruction == INSTRUCTION_ADD {
				scratch |= 0b0001
			} else if instruction == INSTRUCTION_AND {
				scratch |= 0b0101
			}

			for i := 0; i < 2; i++ {
				if operands[i].Type != TOKEN_IDENT {
					errs = append(
						errs,
						&InvalidOperandError{
							operands[i].Position,
							[]TokenType{TOKEN_IDENT},
							operands[i].Type,
						},
					)

					continue
				}

				reg, ok := parseRegister(&operands[i])

				if !ok {
					errs = append(
						errs, &InvalidRegisterError{operands[i].Position},
					)
				}

				scratch <<= 3
				scratch |= (reg & 0x7)
			}

			if operands[2].Type == TOKEN_IDENT {
				reg, ok := parseRegister(&operands[2])

				if !ok {
					errs = append(
						errs, &InvalidRegisterError{operands[2].Position},
					)
				}

				scratch <<= 6
				scratch |= (reg & 0x7)
			} else if operands[2].Type == TOKEN_LITERAL {
				literal, err := parseLiteral(&operands[2], LITERAL_IMM5)

				if err != nil {
					errs = append(errs, err)
				}

				scratch <<= 1
				scratch |= 0x1
				scratch <<= 5
				scratch |= (literal & 0x1F)
			} else {
				errs = append(
					errs,
					&InvalidOperandError{
						operands[2].Position,
						[]TokenType{TOKEN_LITERAL, TOKEN_IDENT},
						operands[2].Type,
					},
				)
			}

		// BR   |0000    |N|Z|P|PCoffset9         | Conditional branch
		// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
		case INSTRUCTION_BR,
			INSTRUCTION_BRn,
			INSTRUCTION_BRz,
			INSTRUCTION_BRp,
			INSTRUCTION_BRnz,
			INSTRUCTION_BRzp,
			INSTRUCTION_BRnp,
			INSTRUCTION_BRnzp:
			if count := len(operands); count != 1 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 1, count},
				)

				break
			}

			const N_FLIP = 0x4
			const Z_FLIP = 0x2
			const P_FLIP = 0x1

			/* scratch |= 0b0000 */

			switch instruction {
			case INSTRUCTION_BRn:
				scratch |= N_FLIP
			case INSTRUCTION_BRz:
				scratch |= Z_FLIP
			case INSTRUCTION_BRp:
				scratch |= P_FLIP
			case INSTRUCTION_BRnz:
				scratch |= (N_FLIP | Z_FLIP)
			case INSTRUCTION_BRzp:
				scratch |= (Z_FLIP | P_FLIP)
			case INSTRUCTION_BRnp:
				scratch |= (N_FLIP | P_FLIP)
			case INSTRUCTION_BRnzp:
				scratch |= (N_FLIP | Z_FLIP | P_FLIP)
			}

			if operands[0].Type != TOKEN_IDENT {
				errs = append(
					errs,
					&InvalidOperandError{
						operands[0].Position,
						[]TokenType{TOKEN_IDENT},
						operands[0].Type,
					},
				)

				break
			}

			labelRefs = append(
				labelRefs,
				LabelRef{
					operands[0].Value,
					uint16(program),
					LITERAL_PCOFFSET9,
					operands[0].Position,
				},
			)

			scratch <<= 9

		// JMP  |1100    |000  |BaseR|000000      | Jump
		// JMPT |1100    |000  |BaseR|000001      | Jump (Clear Privilege)
		// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
		case INSTRUCTION_JMP,
			INSTRUCTION_JMPT:
			if count := len(operands); count != 1 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 1, count},
				)

				break
			}

			if operands[0].Type != TOKEN_IDENT {
				errs = append(
					errs,
					&InvalidOperandError{
						operands[0].Position,
						[]TokenType{TOKEN_IDENT},
						operands[0].Type,
					},
				)

				break
			}

			scratch |= 0b1100
			scratch <<= 6

			reg, ok := parseRegister(&operands[0])

			if !ok {
				errs = append(errs, &InvalidRegisterError{operands[0].Position})
			}

			scratch |= (reg & 0x7)
			scratch <<= 6

			if instruction == INSTRUCTION_JMPT {
				scratch |= 0x1
			}

		// RET  |1100    |000  |111  |000000      | Return
		// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
		case INSTRUCTION_RET:
			if count := len(operands); count != 0 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 0, count},
				)
			}

			scratch = 0b1100000111000000

		// RTT  |1100    |000  |111  |000001      | Return (Clear Privilege)
		// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
		case INSTRUCTION_RTT:
			if count := len(operands); count != 0 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 0, count},
				)
			}

			scratch = 0b1100000111000001

		// JSR  |0100    |1|PCoffset11            | Jump to subroutine
		// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
		case INSTRUCTION_JSR:
			if count := len(operands); count != 1 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 1, count},
				)

				break
			}

			if operands[0].Type != TOKEN_IDENT {
				errs = append(
					errs,
					&InvalidOperandError{
						operands[0].Position,
						[]TokenType{TOKEN_IDENT},
						operands[0].Type,
					},
				)

				break
			}

			scratch |= 0b0100

			scratch <<= 1
			scratch |= 0x1

			labelRefs = append(
				labelRefs,
				LabelRef{
					operands[0].Value,
					uint16(program),
					LITERAL_PCOFFSET11,
					operands[0].Position,
				},
			)

			scratch <<= 11

		// JSRR |0100    |0|00 |BaseR|000000      | Jump to subroutine register
		// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
		case INSTRUCTION_JSRR:
			if count := len(operands); count != 1 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 1, count},
				)

				break
			}

			if operands[0].Type != TOKEN_IDENT {
				errs = append(
					errs,
					&InvalidOperandError{
						operands[0].Position,
						[]TokenType{TOKEN_LITERAL, TOKEN_IDENT},
						operands[0].Type,
					},
				)

				break
			}

			scratch |= 0b0100
			scratch <<= 6

			reg, ok := parseRegister(&operands[0])

			if !ok {
				errs = append(errs, &InvalidRegisterError{operands[0].Position})
			}

			scratch |= (reg & 0x7)
			scratch <<= 6

		// LD   |0010    |DR   |PCoffset9         | Load
		// LDI  |1010    |DR   |PCoffset9         | Load indirect
		// ST   |0011    |SR   |PCoffset9         | Store
		// STI  |1011    |SR   |PCoffset9         | Store indirect
		// LEA  |1110    |DR   |PCoffset9         | Load effective address
		// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
		case INSTRUCTION_LD,
			INSTRUCTION_LDI,
			INSTRUCTION_LEA,
			INSTRUCTION_ST,
			INSTRUCTION_STI:
			if count := len(operands); count != 2 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 2, count},
				)

				break
			}

			switch instruction {
			case INSTRUCTION_LD:
				scratch |= 0b0010
			case INSTRUCTION_LDI:
				scratch |= 0b1010
			case INSTRUCTION_LEA:
				scratch |= 0b1110
			case INSTRUCTION_ST:
				scratch |= 0b0011
			case INSTRUCTION_STI:
				scratch |= 0b1011
			}

			if operands[0].Type != TOKEN_IDENT {
				errs = append(
					errs,
					&InvalidOperandError{
						operands[0].Position,
						[]TokenType{TOKEN_IDENT},
						operands[0].Type,
					},
				)
			} else {
				if reg, ok := parseRegister(&operands[0]); ok {
					scratch <<= 3
					scratch |= (reg & 0x7)
				} else {
					errs = append(
						errs, &InvalidRegisterError{operands[0].Position},
					)
				}
			}

			if operands[1].Type != TOKEN_IDENT {
				errs = append(
					errs,
					&InvalidOperandError{
						operands[1].Position,
						[]TokenType{TOKEN_IDENT},
						operands[1].Type,
					},
				)

				break
			}

			labelRefs = append(
				labelRefs,
				LabelRef{
					operands[1].Value,
					uint16(program),
					LITERAL_PCOFFSET9,
					operands[1].Position,
				},
			)

			scratch <<= 9

		// LDR  |0110    |DR   |BaseR|offset6     | Load base+offset
		// STR  |0111    |SR   |BaseR|offset6     | Store base+offset
		// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
		case INSTRUCTION_LDR, INSTRUCTION_STR:
			if count := len(operands); count != 3 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 3, count},
				)

				break
			}

			if instruction == INSTRUCTION_LDR {
				scratch |= 0b0110
			} else {
				scratch |= 0b0111
			}

			for i := 0; i < 2; i++ {
				if operands[i].Type != TOKEN_IDENT {
					errs = append(
						errs,
						&InvalidOperandError{
							operands[i].Position,
							[]TokenType{TOKEN_IDENT},
							operands[i].Type,
						},
					)

					continue
				}

				reg, ok := parseRegister(&operands[i])

				if !ok {
					errs = append(
						errs, &InvalidRegisterError{operands[i].Position},
					)
				}

				scratch <<= 3
				scratch |= (reg & 0x7)
			}

			if operands[2].Type != TOKEN_LITERAL {
				errs = append(
					errs,
					&InvalidOperandError{
						operands[2].Position,
						[]TokenType{TOKEN_LITERAL},
						operands[2].Type,
					},
				)

				break
			}

			literal, err := parseLiteral(&operands[2], LITERAL_OFFSET6)

			if err != nil {
				errs = append(errs, err)
			}

			scratch <<= 6
			scratch |= (literal & 0x3F)

		// NOT  |1001    |DR   |SR   |1|11111     | Bitwise complement
		// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
		case INSTRUCTION_NOT:
			if count := len(operands); count != 2 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 2, count},
				)

				break
			}

			scratch |= 0b1001

			for i := 0; i < 2; i++ {
				if operands[i].Type != TOKEN_IDENT {
					errs = append(
						errs,
						&InvalidOperandError{
							operands[i].Position,
							[]TokenType{TOKEN_IDENT},
							operands[i].Type,
						},
					)

					continue
				}

				reg, ok := parseRegister(&operands[i])

				if !ok {
					errs = append(
						errs, &InvalidRegisterError{operands[i].Position},
					)
				}

				scratch <<= 3
				scratch |= (reg & 0x7)
			}

			scratch <<= 6
			scratch |= 0x3F

		// RTI  |1000    |000000000000            | Return from interrupt
		// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
		case INSTRUCTION_RTI:
			if count := len(operands); count != 0 {
				errs = append(
					errs, &InvalidNumArgumentsError{keyword.Position, 0, count},
				)

				break
			}

			scratch = 0b1000000000000000

		// TRAP |1111    |0000   |trapvect8       | Store base+offset
		// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
		case INSTRUCTION_TRAP,
			INSTRUCTION_GETC,  // TRAP 0x20
			INSTRUCTION_OUT,   // TRAP 0x21
			INSTRUCTION_PUTS,  // TRAP 0x22
			INSTRUCTION_IN,    // TRAP 0x23
			INSTRUCTION_PUTSP, // TRAP 0x24
			INSTRUCTION_HALT:  // TRAP 0x25
			if instruction == INSTRUCTION_TRAP {
				if count := len(operands); count != 1 {
					errs = append(
						errs,
						&InvalidNumArgumentsError{keyword.Position, 1, count},
					)

					break
				}

				if operands[0].Type != TOKEN_LITERAL {
					errs = append(
						errs,
						&InvalidOperandError{
							operands[0].Position,
							[]TokenType{TOKEN_LITERAL},
							operands[0].Type,
						},
					)

					break
				}
			} else {
				if count := len(operands); count != 0 {
					errs = append(
						errs,
						&InvalidNumArgumentsError{keyword.Position, 0, count},
					)
				}
			}

			scratch |= 0b1111

			var trap uint16
			switch instruction {
			case INSTRUCTION_GETC:
				trap = 0x20
			case INSTRUCTION_OUT:
				trap = 0x21
			case INSTRUCTION_PUTS:
				trap = 0x22
			case INSTRUCTION_IN:
				trap = 0x23
			case INSTRUCTION_PUTSP:
				trap = 0x24
			case INSTRUCTION_HALT:
				trap = 0x25
			default:
				literal, err := parseLiteral(&operands[0], LITERAL_TRAPVEC8)

				if err != nil {
					errs = append(errs, err)
				}

				trap = uint16(literal)
			}

			if trap > 0xFF {
				errs = append(
					errs,
					&OversizedLiteralError{operands[0].Position, 0xFF, trap},
				)
			}

			scratch <<= 12
			scratch |= (trap & 0xFF)
		}

		if symtable != nil {
			symtable.Symbols[uint16(program)] = cursor.LineByte
		}

		if instruction != INSTRUCTION_INVALID {
			result[program] = scratch
			program++
		}

		if program >= math.MaxUint16 {
			errs = append(errs, &OversizedBinaryError{})
			return
		}

		cursor.Line++
		cursor.Byte += int64(len(line) + 1)
		cursor.LineByte += int64(len(line) + 1)
	}

	// Label
	// - Validate and resolve label references
	// - Add labels to symbol table
	for _, ref := range labelRefs {
		addr, exists := labels[ref.Label]

		if !exists {
			errs = append(errs, &UnknownLabelError{ref.Position, ref.Label})
			continue
		}

		limit := int64(1) << (ref.Size - 1)
		offset := int64(addr) - int64(ref.Addr) - 1

		if offset < -limit || offset >= limit {
			errs = append(
				errs, &OversizedLabelError{ref.Position, limit, offset},
			)

			continue
		}

		scratch := result[ref.Addr]
		scratch |= (uint16(offset&0xFFFF) & ((1 << ref.Size) - 1))

		result[ref.Addr] = scratch
	}

	if symtable != nil {
		for label, addr := range labels {
			symtable.Labels[addr] = label
		}
	}

	// Fill
	// - Validate and resolve fill directives whose arguments were unresolved
	//	 label references
	for _, ref := range fillRefs {
		addr, exists := labels[ref.Label]

		if !exists {
			errs = append(errs, &UnknownLabelError{ref.Position, ref.Label})
			continue
		}

		result[ref.Addr] = addr
	}

	return
}
