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

package assembler_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/lassandro/golc3/pkg/assembler"
)

type testCase struct {
	Name     string
	Input    string
	Output   map[uint16]uint16
	SymTable *assembler.SymTable
}

type failCase struct {
	Name  string
	Input string
	Error error
}

func testAssemblerSuccess(t *testing.T, test *testCase) {
	var result []uint16
	var errs []error
	var symtable assembler.SymTable
	var symtarget *assembler.SymTable = nil

	if test.SymTable != nil {
		symtable.Symbols = make(map[uint16]int64)
		symtable.Labels = make(map[uint16]string)
		symtarget = &symtable
	}

	result, errs = assembler.AssembleLC3Source(
		strings.NewReader(test.Input), symtarget,
	)

	if len(errs) > 0 {
		t.Fatal(errs[0])
	}

	if size := len(result); size != math.MaxUint16+1 {
		t.Fatalf(
			"Invalid buffer length\n"+
				"want:%d\n"+
				"have:%d",
			math.MaxUint16,
			size,
		)
	}

	for addr := 0; addr < len(result); addr++ {
		have := result[addr]
		want, exists := test.Output[uint16(addr)]
		if exists && have != want {
			t.Fatalf(
				"Instruction encoding mismatch\n"+
					"want:%#04x (test.Output[%#04x])\n"+
					"have:%#04x",
				want,
				addr,
				have,
			)
		} else if !exists && have != 0 {
			t.Fatalf(
				"Unexpected instruction\n"+
					"want:0x0000\n"+
					"have:%#04x (result [%#04x])",
				have,
				addr,
			)
		}
	}

	if test.SymTable != nil {
		for addr, want := range test.SymTable.Symbols {
			have, exists := symtable.Symbols[addr]

			if !exists {
				t.Fatalf(
					"Missing symtable encoding\n"+
						"want:%d (test.SymTable.Symbols[%#04x])\n"+
						"have:nil",
					want,
					addr,
				)
			} else if have != want {
				t.Fatalf(
					"Symtable encoding mismatch\n"+
						"want:%d (test.SymTable.Symbols[%#04x])\n"+
						"have:%d",
					want,
					addr,
					have,
				)
			}
		}

		for addr, have := range symtable.Symbols {
			_, exists := test.SymTable.Symbols[addr]

			if !exists {
				t.Fatalf(
					"Unexpected symtable encoding\n"+
						"want: nil\n"+
						"have: %d (symtable.Labels[%#04x])",
					have,
					addr,
				)
			}
		}

		for addr, want := range test.SymTable.Labels {
			have, exists := symtable.Labels[addr]

			if !exists {
				t.Fatalf(
					"Missing symtable encoding\n"+
						"want:%s (test.SymTable.Labels[%#04x])\n"+
						"have:nil",
					want,
					addr,
				)
			} else if have != want {
				t.Fatalf(
					"Symtable encoding mismatch\n"+
						"want:%s (test.SymTable.Labels[%#04x])\n"+
						"have:%s",
					want,
					addr,
					have,
				)
			}
		}

		for addr, have := range symtable.Labels {
			_, exists := test.SymTable.Labels[addr]

			if !exists {
				t.Fatalf(
					"Unexpected symtable encoding\n"+
						"want: nil\n"+
						"have: %s (symtable.Labels[%#04x])",
					have,
					addr,
				)
			}
		}
	}
}

func testAssemblerFail(t *testing.T, test *failCase) {
	file := strings.NewReader(test.Input)

	_, errs := assembler.AssembleLC3Source(file, nil)

	if test.Error == nil {
		panic("Fail case missing error value")
	}

	if len(errs) == 0 {
		t.Fatalf(
			"%s produced error of incorrect type"+
				"\nwant:%T (test.Error)\nhave:<nil>",
			t.Name(),
			test.Error,
		)
	}

	if len(errs) > 1 {
		errTypes := make([]reflect.Type, 0, len(errs))
		for _, err := range errs {
			errTypes = append(errTypes, reflect.TypeOf(err))
		}

		t.Fatalf(
			"%s produced multiple errors:\n\twant:%T (test.Error)\n\thave:%v",
			t.Name(),
			test.Error,
			errTypes,
		)
	}

	if reflect.TypeOf(errs[0]) != reflect.TypeOf(test.Error) {
		t.Fatalf(
			"%s produced error of incorrect type"+
				"\nwant:%T (test.Error)\nhave:%T",
			t.Name(),
			test.Error,
			errs[0],
		)
	}
}

func testSuccess(t *testing.T, tests []testCase) {
	t.Run("Success", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				testAssemblerSuccess(t, &test)
			})
		}
	})
}

func testFail(t *testing.T, tests []failCase) {
	t.Run("Fail", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				testAssemblerFail(t, &test)
			})
		}
	})
}

// ADD  |0001    |DR   |SR1  |0|00 |SR2   | Register  addition
// ADD  |0001    |DR   |SR1  |1|imm5      | Immediate addition
// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
func TestAdd(t *testing.T) {
	testSuccess(t, []testCase{
		// ADD DR SR1 SR2
		{
			Name:  "ADD",
			Input: `ADD R0, R1, R2`,
			Output: map[uint16]uint16{
				0x0000: 0b0001_000_001_0_00_010,
			},
		},

		// ADD DR SR1 imm5
		{
			Name:  "ADD imm5",
			Input: `ADD R0, R1, #16`,
			Output: map[uint16]uint16{
				0x0000: 0b0001_000_001_1_10000,
			},
		},
		{
			Name:  "ADD imm5",
			Input: `ADD R0, R1, 0x10`,
			Output: map[uint16]uint16{
				0x0000: 0b0001_000_001_1_10000,
			},
		},
	})

	testFail(t, []failCase{
		// SR2/imm5
		{
			Name:  "ADD Bad SR2",
			Input: `ADD R0, R1, R9`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "ADD Label SR2",
			Input: `ADD R0, R1, LABEL`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "ADD String imm5",
			Input: `ADD R0, R1, "foo"`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "ADD Oversized imm5",
			Input: `ADD R0, R1, #1234`,
			Error: &assembler.OversizedLiteralError{},
		},
		{
			Name:  "ADD Oversized imm5",
			Input: `ADD R0, R1, 0xFF`,
			Error: &assembler.OversizedLiteralError{},
		},

		// SR1
		{
			Name:  "ADD Bad SR1",
			Input: `ADD R0, R9, R2`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "ADD Label SR1",
			Input: `ADD R0, LABEL, R2`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "ADD String SR1",
			Input: `ADD R0, "foo", R2`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "ADD Literal SR1",
			Input: `ADD R0, #1, R2`,
			Error: &assembler.InvalidOperandError{},
		},

		// DR
		{
			Name:  "ADD Bad DR",
			Input: `ADD R9, R1, R2`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "ADD Label DR",
			Input: `ADD LABEL, R1, R2`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "ADD String DR",
			Input: `ADD "foo", R1, R2`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "ADD Literal DR",
			Input: `ADD #1, R1, R2`,
			Error: &assembler.InvalidOperandError{},
		},

		// Misc
		{
			Name:  "ADD Bad Argc",
			Input: `ADD R0, R1, R2, R3`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "ADD Bad Argc",
			Input: `ADD R0, R1`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "ADD Bad Argc",
			Input: `ADD R0`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
	})
}

// AND  |0101    |DR   |SR1  |0|00 |SR2   | Register  bitwise
// AND  |0101    |DR   |SR1  |1|imm5      | Immediate bitwise
func TestAnd(t *testing.T) {
	testSuccess(t, []testCase{
		// AND DR SR1 SR2
		{
			Name:  "AND",
			Input: `AND R0, R1, R2`,
			Output: map[uint16]uint16{
				0x0000: 0b0101_000_001_0_00_010,
			},
		},

		// AND DR SR1 imm5
		{
			Name:  "AND imm5",
			Input: `AND R0, R1, #16`,
			Output: map[uint16]uint16{
				0x0000: 0b0101_000_001_1_10000,
			},
		},
		{
			Name:  "AND imm5",
			Input: `AND R0, R1, 16`,
			Output: map[uint16]uint16{
				0x0000: 0b0101_000_001_1_10000,
			},
		},
		{
			Name:  "AND imm5",
			Input: `AND R0, R1, 0x10`,
			Output: map[uint16]uint16{
				0x0000: 0b0101_000_001_1_10000,
			},
		},
	})

	testFail(t, []failCase{
		// SR2/imm5
		{
			Name:  "AND Label SR2",
			Input: `AND R0, R1, LABEL`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "AND String imm5",
			Input: `AND R0, R1, "foo"`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "AND Oversized imm5",
			Input: `AND R0, R1, #255`,
			Error: &assembler.OversizedLiteralError{},
		},
		{
			Name:  "AND Oversized imm5",
			Input: `AND R0, R1, 0xFF`,
			Error: &assembler.OversizedLiteralError{},
		},

		// SR1
		{
			Name:  "AND Bad SR1",
			Input: `AND R0, R9, R2`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "AND Label SR1",
			Input: `AND R0, LABEL, R2`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "AND String SR1",
			Input: `AND R0, "foo", R2`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "AND Literal SR1",
			Input: `AND R0, #1, R2`,
			Error: &assembler.InvalidOperandError{},
		},

		// DR
		{
			Name:  "AND Bad DR",
			Input: `AND R9, R1, R2`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "AND Label DR",
			Input: `AND LABEL, R1, R2`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "AND String DR",
			Input: `AND "foo", R1, R2`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "AND Literal DR",
			Input: `AND #1, R1, R2`,
			Error: &assembler.InvalidOperandError{},
		},

		// Misc
		{
			Name:  "AND Bad Argc",
			Input: `AND R0, R1, R2, R3`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "AND Bad Argc",
			Input: `AND R0, R1`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "AND Bad Argc",
			Input: `AND R0`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "AND Bad Argc",
			Input: `AND`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
	})
}

// BR   |0000    |N|Z|P|PCoffset9         | Conditional branch
// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
func TestBranch(t *testing.T) {
	testSuccess(t, []testCase{
		// BR PCoffset9
		{
			Name:  "BR",
			Input: `LABEL BR LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b0000_000_111111111,
			},
		},

		// BR(n|z|p) PCoffset9
		{
			Name:  "BRn",
			Input: `LABEL BRn LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b0000_100_111111111,
			},
		},
		{
			Name:  "BRz",
			Input: `LABEL BRz LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b0000_010_111111111,
			},
		},
		{
			Name:  "BRp",
			Input: `LABEL BRp LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b0000_001_111111111,
			},
		},

		// BR(nz|zp|np) PCoffset9
		{
			Name:  "BRnz",
			Input: `LABEL BRnz LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b0000_110_111111111,
			},
		},
		{
			Name:  "BRzp",
			Input: `LABEL BRzp LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b0000_011_111111111,
			},
		},
		{
			Name:  "BRnp",
			Input: `LABEL BRnp LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b0000_101_111111111,
			},
		},

		// BRnzp PCoffset9
		{
			Name:  "BRnzp",
			Input: `LABEL BRnzp LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b0000_111_111111111,
			},
		},
	})

	testFail(t, []failCase{
		// BR(nzp) PCoffset9
		{
			Name:  "BR(nzp) Bad PCoffset9",
			Input: `LABEL BR FOO`,
			Error: &assembler.UnknownLabelError{},
		},
		{
			Name:  "BR(nzp) String PCoffset9",
			Input: `LABEL BR "LABEL"`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "BR(nzp) Literal PCoffset9",
			Input: `LABEL BR 0x3000`,
			Error: &assembler.InvalidOperandError{},
		},

		// Misc
		{
			Name:  "BR(nzp) Bad Argc",
			Input: `LABEL BR LABEL FOO`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "BR(nzp) Bad Argc",
			Input: `LABEL BR`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "BR(nzp) Bad Order",
			Input: `LABEL BRpnz LABEL`,
			Error: &assembler.UnknownIdentifierError{},
		},
		{
			Name:  "BR(nzp) Bad Order",
			Input: `LABEL BRznp LABEL`,
			Error: &assembler.UnknownIdentifierError{},
		},
		{
			Name:  "BR(nzp) Bad Order",
			Input: `LABEL BRnpz LABEL`,
			Error: &assembler.UnknownIdentifierError{},
		},
	})
}

// JMP  |1100    |000  |BaseR|000000      | Jump
// JMPT |1100    |000  |BaseR|000001      | Jump (Clear Privilege)
// JSR  |0100    |1|PCoffset11            | Jump to subroutine
// JSRR |0100    |0|00 |BaseR|000000      | Jump to subroutine register
// RET  |1100    |000  |111  |000000      | Return
// RTT  |1100    |000  |111  |000001      | Return (Clear Privilege)
// RTI  |1000    |000000000000            | Return from interrupt
// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
func TestJump(t *testing.T) {
	testSuccess(t, []testCase{
		// JMP BaseR
		{
			Name:  "JMP",
			Input: `JMP R2`,
			Output: map[uint16]uint16{
				0x0000: 0b1100_000_010_000000,
			},
		},

		// JMPT BaseR
		{
			Name:  "JMPT",
			Input: `JMPT R2`,
			Output: map[uint16]uint16{
				0x0000: 0b1100_000_010_000001,
			},
		},

		// JSR PCOffset11
		{
			Name:  "JSR",
			Input: `LABEL JSR LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b0100_1_11111111111,
			},
		},

		// JSRR BaseR
		{
			Name:  "JSRR",
			Input: `JSRR R2`,
			Output: map[uint16]uint16{
				0x0000: 0b0100_000_010_000000,
			},
		},

		// RET
		{
			Name:  "RET",
			Input: `RET`,
			Output: map[uint16]uint16{
				0x0000: 0b1100_000_111_000000,
			},
		},

		// RTT
		{
			Name:  "RTT",
			Input: `RTT`,
			Output: map[uint16]uint16{
				0x0000: 0b1100_000_111_000001,
			},
		},

		// RTI
		{
			Name:  "RTI",
			Input: `RTI`,
			Output: map[uint16]uint16{
				0x0000: 0b1000_000000000000,
			},
		},
	})

	testFail(t, []failCase{
		// JMP BaseR
		{
			Name:  "JMP Bad BaseR",
			Input: `JMP R9`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "JMP Literal BaseR",
			Input: `JMP #1`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "JMP String BaseR",
			Input: `JMP "foo"`,
			Error: &assembler.InvalidOperandError{},
		},

		// JMP Misc
		{
			Name:  "JMP Bad Argc",
			Input: `JMP R0, R1`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "JMP Bad Argc",
			Input: `JMP`,
			Error: &assembler.InvalidNumArgumentsError{},
		},

		// JMPT BaseR
		{
			Name:  "JMPT Bad BaseR",
			Input: `JMPT R9`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "JMPT Literal BaseR",
			Input: `JMPT #1`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "JMPT String BaseR",
			Input: `JMPT "foo"`,
			Error: &assembler.InvalidOperandError{},
		},

		// JMPT Misc
		{
			Name:  "JMPT Bad Argc",
			Input: `JMPT R0, R1`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "JMPT Bad Argc",
			Input: `JMPT`,
			Error: &assembler.InvalidNumArgumentsError{},
		},

		// JSR PCOffset11
		{
			Name:  "JSR String PCOffset11",
			Input: `LABEL JSR "LABEL"`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "JSR Literal PCOffset11",
			Input: `LABEL JSR #1`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "JSR Unknown PCOffset11",
			Input: `LABEL JSR FOO`,
			Error: &assembler.UnknownLabelError{},
		},

		// JSR Misc
		{
			Name:  "JSR Bad Argc",
			Input: `LABEL JSR LABEL, LABEL`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "JSR Bad Argc",
			Input: `LABEL JSR`,
			Error: &assembler.InvalidNumArgumentsError{},
		},

		// JSRR BaseR
		{
			Name:  "JSRR Bad BaseR",
			Input: `JSRR R9`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "JSRR Literal BaseR",
			Input: `JSRR #1`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "JSRR String BaseR",
			Input: `JSRR "R1"`,
			Error: &assembler.InvalidOperandError{},
		},

		// JSRR Misc
		{
			Name:  "JSRR Bad Argc",
			Input: `JSRR R0, R1`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "JSRR Bad Argc",
			Input: `JSRR`,
			Error: &assembler.InvalidNumArgumentsError{},
		},

		// RET/RTT/RTI Misc
		{
			Name:  "RET Bad Argc",
			Input: `RET R0`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "RTT Bad Argc",
			Input: `RTT R0`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "RTI Bad Argc",
			Input: `RTI R0`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
	})
}

// LD   |0010    |DR   |PCoffset9         | Load
// LDI  |1010    |DR   |PCoffset9         | Load indirect
// LDR  |0110    |DR   |BaseR|offset6     | Load base+offset
// LEA  |1110    |DR   |PCoffset9         | Load effective address
// ST   |0011    |SR   |PCoffset9         | Store
// STI  |1011    |SR   |PCoffset9         | Store indirect
// STR  |0111    |SR   |BaseR|offset6     | Store base+offset
// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
func TestLoadStore(t *testing.T) {
	testSuccess(t, []testCase{
		// LD DR PCoffset9
		{
			Name:  "LD",
			Input: `LABEL LD R2 LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b0010_010_111111111,
			},
		},

		// LDI DR PCoffset9
		{
			Name:  "LDI",
			Input: `LABEL LDI R2 LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b1010_010_111111111,
			},
		},

		// LDR DR BaseR offset6
		{
			Name:  "LDR",
			Input: `LDR R2, R3, #32`,
			Output: map[uint16]uint16{
				0x0000: 0b0110_010_011_100000,
			},
		},
		{
			Name:  "LDR",
			Input: `LDR R2, R3, 32`,
			Output: map[uint16]uint16{
				0x0000: 0b0110_010_011_100000,
			},
		},
		{
			Name:  "LDR",
			Input: `LDR R2, R3, 0x20`,
			Output: map[uint16]uint16{
				0x0000: 0b0110_010_011_100000,
			},
		},

		// LEA DR PCoffset9
		{
			Name:  "LEA",
			Input: `LABEL LEA R2, LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b1110_010_111111111,
			},
		},

		// ST SR PCoffset9
		{
			Name:  "ST",
			Input: `LABEL ST R2, LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b0011_010_111111111,
			},
		},

		// STI SR PCoffset9
		{
			Name:  "STI",
			Input: `LABEL STI R2, LABEL`,
			Output: map[uint16]uint16{
				0x0000: 0b1011_010_111111111,
			},
		},

		// STR DR BaseR offset6
		{
			Name:  "STR",
			Input: `STR R2, R3, #32`,
			Output: map[uint16]uint16{
				0x0000: 0b0111_010_011_100000,
			},
		},
		{
			Name:  "STR",
			Input: `STR R2, R3, 32`,
			Output: map[uint16]uint16{
				0x0000: 0b0111_010_011_100000,
			},
		},
		{
			Name:  "STR",
			Input: `STR R2, R3, 0x20`,
			Output: map[uint16]uint16{
				0x0000: 0b0111_010_011_100000,
			},
		},
	})

	testFail(t, []failCase{
		// LD PCoffset9
		{
			Name:  "LD Bad PCoffset9",
			Input: `LABEL LD R0 FOO`,
			Error: &assembler.UnknownLabelError{},
		},
		{
			Name:  "LD String PCoffset9",
			Input: `LABEL LD R0 "LABEL"`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "LD Literal DR",
			Input: `LABEL LD R0 0x3000`,
			Error: &assembler.InvalidOperandError{},
		},

		// LD DR
		{
			Name:  "LD Bad DR",
			Input: `LABEL LD R9 LABEL`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "LD String DR",
			Input: `LABEL LD "R0" LABEL`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "LD Literal DR",
			Input: `LABEL LD #0 LABEL`,
			Error: &assembler.InvalidOperandError{},
		},

		// LDI PCoffset9
		{
			Name:  "LDI Bad PCoffset9",
			Input: `LABEL LDI R0 FOO`,
			Error: &assembler.UnknownLabelError{},
		},
		{
			Name:  "LDI String PCoffset9",
			Input: `LABEL LDI R0 "LABEL"`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "LDI Literal DR",
			Input: `LABEL LDI R0 0x3000`,
			Error: &assembler.InvalidOperandError{},
		},

		// LDI DR
		{
			Name:  "LDI Bad DR",
			Input: `LABEL LDI R9 LABEL`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "LDI String DR",
			Input: `LABEL LDI "R0" LABEL`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "LDI Literal DR",
			Input: `LABEL LDI #0 LABEL`,
			Error: &assembler.InvalidOperandError{},
		},

		// LDR offset6
		{
			Name:  "LDR String offset6",
			Input: `LDR R0 R1 "FOO"`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "LDR Label DR",
			Input: `LABEL LDR R0 R1 LABEL`,
			Error: &assembler.InvalidOperandError{},
		},

		// LDR BaseR
		{
			Name:  "LDR Bad BaseR",
			Input: `LDR R0 R9 #32`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "LDR String BaseR",
			Input: `LDR R0 "R1" #32`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "LDR Literal DR",
			Input: `LDR R0 #1 #32`,
			Error: &assembler.InvalidOperandError{},
		},

		// LDR DR
		{
			Name:  "LDR Bad BaseR",
			Input: `LDR R9 R0 #32`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "LDR String BaseR",
			Input: `LDR "R0" R1 #32`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "LDR Literal DR",
			Input: `LDR #0 R1 #32`,
			Error: &assembler.InvalidOperandError{},
		},

		// LEA PCoffset9
		{
			Name:  "LEA Bad PCoffset9",
			Input: `LABEL LEA R0 FOO`,
			Error: &assembler.UnknownLabelError{},
		},
		{
			Name:  "LEA String PCoffset9",
			Input: `LABEL LEA R0 "LABEL"`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "LEA Literal DR",
			Input: `LABEL LEA R0 0x3000`,
			Error: &assembler.InvalidOperandError{},
		},

		// LEA DR
		{
			Name:  "LEA Bad DR",
			Input: `LABEL LEA R9 LABEL`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "LEA String DR",
			Input: `LABEL LEA "R0" LABEL`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "LEA Literal DR",
			Input: `LABEL LEA #0 LABEL`,
			Error: &assembler.InvalidOperandError{},
		},

		// LD PCoffset9
		{
			Name:  "LD Bad PCoffset9",
			Input: `LABEL LD R0 FOO`,
			Error: &assembler.UnknownLabelError{},
		},
		{
			Name:  "LD String PCoffset9",
			Input: `LABEL LD R0 "LABEL"`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "LD Literal DR",
			Input: `LABEL LD R0 0x3000`,
			Error: &assembler.InvalidOperandError{},
		},

		// ST DR
		{
			Name:  "ST Bad DR",
			Input: `LABEL ST R9 LABEL`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "ST String DR",
			Input: `LABEL ST "R0" LABEL`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "ST Literal DR",
			Input: `LABEL ST #0 LABEL`,
			Error: &assembler.InvalidOperandError{},
		},

		// STI PCoffset9
		{
			Name:  "STI Bad PCoffset9",
			Input: `LABEL STI R0 FOO`,
			Error: &assembler.UnknownLabelError{},
		},
		{
			Name:  "STI String PCoffset9",
			Input: `LABEL STI R0 "LABEL"`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "STI Literal DR",
			Input: `LABEL STI R0 0x3000`,
			Error: &assembler.InvalidOperandError{},
		},

		// STI DR
		{
			Name:  "STI Bad DR",
			Input: `LABEL STI R9 LABEL`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "STI String DR",
			Input: `LABEL STI "R0" LABEL`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "STI Literal DR",
			Input: `LABEL STI #0 LABEL`,
			Error: &assembler.InvalidOperandError{},
		},

		// STR offset6
		{
			Name:  "LDR String offset6",
			Input: `LDR R0 R1 "FOO"`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "LDR Label DR",
			Input: `LABEL LDR R0 R1 LABEL`,
			Error: &assembler.InvalidOperandError{},
		},

		// STR BaseR
		{
			Name:  "STR Bad BaseR",
			Input: `STR R0 R9 #32`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "STR String BaseR",
			Input: `STR R0 "R1" #32`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "STR Literal DR",
			Input: `STR R0 #1 #32`,
			Error: &assembler.InvalidOperandError{},
		},

		// STR DR
		{
			Name:  "STR Bad BaseR",
			Input: `STR R9 R0 #32`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "STR String BaseR",
			Input: `STR "R0" R1 #32`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "STR Literal DR",
			Input: `STR #0 R1 #32`,
			Error: &assembler.InvalidOperandError{},
		},
	})
}

// NOT  |1001    |DR   |SR   |1|11111     | Bitwise complement
// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
func TestNot(t *testing.T) {
	testSuccess(t, []testCase{
		// NOT DR SR
		{
			Name:  "NOT",
			Input: `NOT R3 R4`,
			Output: map[uint16]uint16{
				0x0000: 0b1001_011_100_1_11111,
			},
		},
	})

	testFail(t, []failCase{
		// SR
		{
			Name:  "NOT Bad SR",
			Input: `NOT R3, R9`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "NOT String SR",
			Input: `NOT R3, "foo"`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "NOT Literal SR",
			Input: `NOT R3, #1`,
			Error: &assembler.InvalidOperandError{},
		},

		// DR
		{
			Name:  "NOT Bad DR",
			Input: `NOT R9, R4`,
			Error: &assembler.InvalidRegisterError{},
		},
		{
			Name:  "NOT String DR",
			Input: `NOT "foo", R4`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "NOT Literal DR",
			Input: `NOT #1, R4`,
			Error: &assembler.InvalidOperandError{},
		},

		// Misc
		{
			Name:  "NOT Bad Argc",
			Input: `NOT R0, R1, R2`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "NOT Bad Argc",
			Input: `NOT R0`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "NOT Bad Argc",
			Input: `NOT`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
	})
}

// TRAP |1111    |0000   |trapvect8       | Store base+offset
// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
func TestTrap(t *testing.T) {
	testSuccess(t, []testCase{
		// TRAP trapvect8
		{
			Name:  "TRAP",
			Input: `TRAP 0x20`,
			Output: map[uint16]uint16{
				0x0000: 0b1111_0000_00100000,
			},
		},

		// GETC (TRAP 0x20)
		{
			Name:  "GETC",
			Input: `GETC`,
			Output: map[uint16]uint16{
				0x0000: 0b1111_0000_00100000,
			},
		},

		// OUT (TRAP 0x21)
		{
			Name:  "OUT",
			Input: `OUT`,
			Output: map[uint16]uint16{
				0x0000: 0b1111_0000_00100001,
			},
		},

		// PUTS (TRAP 0x22)
		{
			Name:  "PUTS",
			Input: `PUTS`,
			Output: map[uint16]uint16{
				0x0000: 0b1111_0000_00100010,
			},
		},

		// IN (TRAP 0x23)
		{
			Name:  "IN",
			Input: `IN`,
			Output: map[uint16]uint16{
				0x0000: 0b1111_0000_00100011,
			},
		},

		// PUTSP (TRAP 0x24)
		{
			Name:  "PUTSP",
			Input: `PUTSP`,
			Output: map[uint16]uint16{
				0x0000: 0b1111_0000_00100100,
			},
		},

		// HALT (TRAP 0x25)
		{
			Name:  "HALT",
			Input: `HALT`,
			Output: map[uint16]uint16{
				0x0000: 0b1111_0000_00100101,
			},
		},
	})

	testFail(t, []failCase{
		// TRAP trapvect8
		{
			Name:  "TRAP String trapvect8",
			Input: `TRAP "foo"`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  "TRAP Bad trapvect8",
			Input: `TRAP 0x1FF`,
			Error: &assembler.OversizedLiteralError{},
		},

		// Misc
		{
			Name:  "TRAP Bad Argc",
			Input: `TRAP 0x0020 0x0020`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "GETC Bad Argc",
			Input: `GETC 0x0020`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "OUT Bad Argc",
			Input: `OUT 0x0020`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "PUTS Bad Argc",
			Input: `PUTS 0x0020`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "IN Bad Argc",
			Input: `IN 0x0020`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "PUTSP Bad Argc",
			Input: `PUTSP 0x0020`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
		{
			Name:  "HALT Bad Argc",
			Input: `HALT 0x0020`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
	})
}

func TestOrig(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name: ".ORIG Zero",
			Input: `
			.ORIG 0x0000
			RET
			`,
			Output: map[uint16]uint16{
				0x0000: 0b1100_000_111_000000,
			},
		},
		{
			Name: ".ORIG Zero",
			Input: `
			.ORIG #0
			RET
			`,
			Output: map[uint16]uint16{
				0: 0b1100_000_111_000000,
			},
		},
		{
			Name: ".ORIG Literal",
			Input: `
			.ORIG 0x3000
			RET
			`,
			Output: map[uint16]uint16{
				0x3000: 0b1100_000_111_000000,
			},
		},
		{
			Name: ".ORIG Literal",
			Input: `
			.ORIG #63
			RET
			`,
			Output: map[uint16]uint16{
				63: 0b1100_000_111_000000,
			},
		},
		{
			Name: ".ORIG Multiple",
			Input: `
			.ORIG 0x0000
			OUT
			.ORIG 0x3000
			RET
			.ORIG 0x1000
			PUTS
			`,
			Output: map[uint16]uint16{
				0x0000: 0b1111_0000_00100001,
				0x3000: 0b1100_000_111_000000,
				0x1000: 0b1111_0000_00100010,
			},
		},
	})

	testFail(t, []failCase{
		{
			Name: ".ORIG Label",
			Input: `
			LABEL
			.ORIG LABEL
			`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name: ".ORIG String Literal",
			Input: `
			.ORIG "foo"
			`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name: ".ORIG Invalid",
			Input: `
			.ORIG #999999999
			`,
			Error: &assembler.InvalidLiteralError{},
		},
	})
}

func TestFill(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name:  ".FILL Literal",
			Input: `.FILL 0xFFFF`,
			Output: map[uint16]uint16{
				0x0000: 0b1111111111111111,
			},
		},
		{
			Name:  ".FILL Literal",
			Input: `.FILL #13`,
			Output: map[uint16]uint16{
				0x0000: 0b0000000000001101,
			},
		},
		{
			Name: ".FILL Forward Label",
			Input: `
			.FILL LABEL
			LABEL RET
			HALT
			`,
			Output: map[uint16]uint16{
				0x0000: 0x0001,
				0x0001: 0b1100_000_111_000000,
				0x0002: 0b1111_0000_00100101,
			},
		},
		{
			Name: ".FILL Backward Label",
			Input: `
			LABEL RET
			.FILL LABEL
			HALT
			`,
			Output: map[uint16]uint16{
				0x0000: 0b1100_000_111_000000,
				0x0001: 0x0000,
				0x0002: 0b1111_0000_00100101,
			},
		},
	})

	testFail(t, []failCase{
		{
			Name:  ".FILL String Literal",
			Input: `.FILL "foo"`,
			Error: &assembler.InvalidOperandError{},
		},
	})
}

func TestBlkw(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name: ".BLKW Literal",
			Input: `
			.BLKW 0x03
			RET
			`,
			Output: map[uint16]uint16{
				0x0003: 0b1100_000_111_000000,
			},
		},
		{
			Name: ".BLKW Literal",
			Input: `
			.BLKW #64
			RET
			`,
			Output: map[uint16]uint16{
				64: 0b1100_000_111_000000,
			},
		},
	})

	testFail(t, []failCase{
		{
			Name:  ".BLKW Label",
			Input: `LABEL .BLKW LABEL`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  ".BLKW String",
			Input: `.BLKW "foo"`,
			Error: &assembler.InvalidOperandError{},
		},
	})
}

func TestStringz(t *testing.T) {
	t.Run(".STRINGZ", func(t *testing.T) {
		file := strings.NewReader(`
		.STRINGZ "Hello World"
		.STRINGZ "Hello World"
		`)

		result, errs := assembler.AssembleLC3Source(file, nil)

		if len(errs) > 0 {
			t.Fatal(errs[0])
		}

		{
			want := math.MaxUint16 + 1
			have := len(result)
			if have != want {
				t.Fatalf(
					"Invalid output length\nwant:%d\nhave:%d",
					want,
					have,
				)
			}
		}

		expected := "Hello World"
		for i, want := range expected {
			if have := int32(result[i]); have != want {
				t.Fatalf(
					"Invalid string encoding [%d]\nwant:%c\nhave:%c",
					i,
					want,
					have,
				)
			}
		}

		if result[len(expected)] != 0 {
			t.Fatalf("Missing null terminator in string encoding")
		}

		for i, want := range expected {
			i += len(expected) + 1
			if have := int32(result[i]); have != want {
				t.Fatalf(
					"Invalid string encoding [%d]\nwant:%c\nhave:%c",
					i,
					want,
					have,
				)
			}
		}

		for i := (len(expected) + 1) * 2; i < len(result)-1; i++ {
			if have := result[i]; have != 0 {
				t.Fatalf("Unexpected byte [%d]\nhave:%c", i, have)
			}
		}
	})

	testFail(t, []failCase{
		{
			Name:  ".STRINGZ Label",
			Input: `.STRINGZ LABEL`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  ".STRINGZ Literal",
			Input: `.STRINGZ #16`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  ".STRINGZ Literal",
			Input: `.STRINGZ 0xFF`,
			Error: &assembler.InvalidOperandError{},
		},
		{
			Name:  ".STRINGZ Missing Delimiter",
			Input: `.STRINGZ "foo`,
			Error: &assembler.InvalidStringError{},
		},
	})
}

func TestEnd(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name: ".END",
			Input: `
			.END
			`,
			Output: make(map[uint16]uint16),
		},
		{
			Name: ".END After Instructions",
			Input: `
			RET
			.END
			`,
			Output: map[uint16]uint16{
				0x0000: 0b1100_000_111_000000,
			},
		},
		{
			Name: ".END Before Instructions",
			Input: `
			.END
			RET
			`,
			Output: make(map[uint16]uint16),
		},
	})

	testFail(t, []failCase{
		{
			Name:  ".END Bad Argc",
			Input: `.END foo`,
			Error: &assembler.InvalidNumArgumentsError{},
		},
	})
}

func TestComment(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name:   "Comment",
			Input:  `; Lorem Ipsum`,
			Output: make(map[uint16]uint16),
		},
		{
			Name: "Multiple Comments",
			Input: `
			; Lorem Ipsum
			; Lorem Ipsum
			; Lorem Ipsum ; Lorem Ipsum
			`,
			Output: make(map[uint16]uint16),
		},
		{
			Name: "Comments With Statements",
			Input: `
			; Lorem Ipsum
			; Lorem Ipsum
			; Lorem Ipsum ; Lorem Ipsum
			RET
			`,
			Output: map[uint16]uint16{
				0x0000: 0b1100_000_111_000000,
			},
		},
		{
			Name: "Inline Comments",
			Input: `
			; Lorem Ipsum
			OUT ; Lorem Ipsum
			RET; Lorem Ipsum
			; HALT
			`,
			Output: map[uint16]uint16{
				0x0000: 0b1111_0000_00100001,
				0x0001: 0b1100_000_111_000000,
			},
		},
	})
}

func TestLabel(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name: "Backwards Label",
			Input: `
			LABEL
				HALT
				HALT
				JSR LABEL
			`,
			Output: map[uint16]uint16{
				0x0000: 0b1111_0000_00100101, // HALT
				0x0001: 0b1111_0000_00100101, // HALT
				0x0002: 0b0100_1_11111111101, // JSR -(2)
			},
		},
		{
			Name: "Forwards Label",
			Input: `
			JSR LABEL
			HALT
			HALT
			LABEL
			`,
			Output: map[uint16]uint16{
				0x0000: 0b0100_1_00000000010, // JSR +(2)
				0x0001: 0b1111_0000_00100101, // HALT
				0x0002: 0b1111_0000_00100101, // HALT
			},
		},
		{
			Name: "Forwards Label Long",
			Input: `
			BR LABEL
			.BLKW #255
			LABEL
			`,
			Output: map[uint16]uint16{
				0x0000: 0b0000_000_011111111,
			},
		},
		{
			Name: "Backwards Label Long",
			Input: `
			LABEL
				.BLKW #255
				BR LABEL
			`,
			Output: map[uint16]uint16{
				255: 0b0000_000_100000000,
			},
		},
	})

	testFail(t, []failCase{
		{
			Name:  "Invalid Label",
			Input: `JSR LABEL`,
			Error: &assembler.UnknownLabelError{},
		},
		{
			Name: "Oversized Label",
			Input: `
			LABEL
				.BLKW #1024
				JSR LABEL
			`,
			Error: &assembler.OversizedLabelError{},
		},
		{
			Name: "Oversized Label",
			Input: `
			JSR LABEL
			.BLKW #1024
			LABEL
			`,
			Error: &assembler.OversizedLabelError{},
		},
	})
}

func TestProgramSize(t *testing.T) {
	testFail(t, []failCase{
		{
			Name:  "Oversized Binary",
			Input: `.BLKW 0xFFFF`,
			Error: &assembler.OversizedBinaryError{},
		},
		{
			Name: "Oversized Binary",
			Input: `
			.ORIG 0xFFFF
			.BLKW 0x000F
			RET
			`,
			Error: &assembler.OversizedBinaryError{},
		},
	})
}

func TestSymtable(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name: "Symtable",
			/*
				+ 13	.ORIG 0x3000
				+  7	LABEL1
				+ 10	TRAP 0x00
				+  7	LABEL2
				+ 10	.BLKW #10
				+  7	LABEL3
				+  3	RTI
				----
				= 57
			*/
			Input: (".ORIG 0x3000\n" +
				"LABEL1\n" +
				"TRAP 0x00\n" +
				"LABEL2\n" +
				".BLKW #10\n" +
				"LABEL3\n" +
				"RTI"),
			Output: map[uint16]uint16{
				0x3000: 0b1111_0000_00000000,
				0x300B: 0b1000_000000000000,
			},
			SymTable: &assembler.SymTable{
				Symbols: map[uint16]int64{
					0x3000: 20, // TRAP
					0x300B: 54, // RTI
				},
				Labels: map[uint16]string{
					0x3000: "LABEL1",
					0x3001: "LABEL2",
					0x300B: "LABEL3",
				},
			},
		},
	})
}
