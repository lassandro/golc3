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

package machine_test

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/lassandro/golc3/pkg/machine"
)

type testMachineState struct {
	Registers [8]uint16
	Program   uint16
	Privilege bool
	Priority  uint16
	Condition uint16
	Memory    map[uint16]uint16
	Stack     uint16
}

type testCase struct {
	Name     string
	Steps    uint
	Keyboard string
	Display  string
	Input    testMachineState
	Output   testMachineState
}

func testMachineSuccess(t *testing.T, test *testCase) {
	if test.Input.Priority > 0x7 {
		panic("Priority must be 0x7 or lower")
	}

	if test.Input.Condition > 0x7 {
		panic("Condition must be 0x7 or lower")
	}

	if test.Input.Memory == nil && test.Output.Memory == nil {
		panic("No memory maps provided")
	}

	var mc machine.Machine
	var devices machine.DeviceHandler
	var displayBuf bytes.Buffer

	if len(test.Keyboard) > 0 {
		devices.Keyboard = bufio.NewReader(
			bytes.NewReader([]byte(test.Keyboard)),
		)
	}

	if len(test.Display) > 0 {
		devices.Display = bufio.NewWriter(&displayBuf)
	}

	if devices.Keyboard != nil || devices.Display != nil {
		mc.Devices = &devices
	}

	mc.State.Reset()
	mc.State.Registers = test.Input.Registers
	mc.State.Program = test.Input.Program
	mc.State.Stack = test.Input.Stack

	if test.Input.Privilege {
		mc.State.Procstat |= (1 << 15)
	} else {
		mc.State.Procstat = 0
	}
	mc.State.Procstat |= test.Input.Priority << 8
	mc.State.Procstat |= test.Input.Condition

	for addr, value := range test.Input.Memory {
		mc.State.Memory[addr] = value
	}

	if test.Steps == 0 {
		test.Steps = 1
	}

	for i := uint(0); i < test.Steps; i++ {
		mc.Step()
	}

	for i := 0; i < 8; i++ {
		want := test.Output.Registers[i]
		have := mc.State.Registers[i]
		if have != want {
			t.Errorf(
				"Register mismatch"+
					"\nwant:%#04x (test.Output.Registers[%d])\nhave:%#04x",
				want,
				i,
				have,
			)
		}
	}

	if mc.State.Program != test.Output.Program {
		t.Errorf(
			"Program register mismatch"+
				"\nwant:%#04x (test.Output.Program)\nhave:%#04x",
			test.Output.Program,
			mc.State.Program,
		)
	}

	if test.Output.Privilege && (mc.State.Procstat>>15) != 1 {
		t.Error(
			"Privilege level mismatch" +
				"\nwant:Supervisor Mode (test.Output.Privilege)" +
				"\nhave:User Mode",
		)
	} else if !test.Output.Privilege && (mc.State.Procstat>>15) != 0 {
		t.Error(
			"Privilege level mismatch" +
				"\nwant:User Mode (test.Output.Privilege)" +
				"\nhave:Supervisor Mode",
		)
	}

	if have := ((mc.State.Procstat >> 8) & 0x7); have != test.Output.Priority {
		t.Errorf(
			"Priority level mismatch"+
				"\nwant:%#01x (test.Output.Priority)\nhave:%#01x",
			test.Output.Priority,
			have,
		)
	}

	if have := (mc.State.Procstat & 0x7); have != test.Output.Condition {
		t.Errorf(
			"Condition flag mismatch"+
				"\nwant:%#03b (test.Output.Condition)\nhave:%#03b",
			test.Output.Condition,
			have,
		)
	}

	if have := mc.State.Stack; have != test.Output.Stack {
		t.Errorf(
			"Saved stack mismtach"+
				"\nwant:%#04x (test.Output.Stack)\nhave:%#04x",
			test.Output.Stack,
			have,
		)
	}

	for i, value := range mc.State.Memory {
		input, expectingInput := test.Input.Memory[uint16(i)]
		output, expectingOutput := test.Output.Memory[uint16(i)]

		if expectingOutput {
			// Value was supposed to change
			if value != output {
				t.Fatalf(
					"Memory value mismatch"+
						"\nwant:%#02x (test.Output.Memory[%#04x])\nhave:%#02x",
					output,
					i,
					value,
				)
			}
		} else if expectingInput {
			// Value was supposed to remain
			if value != input {
				t.Fatalf(
					"Memory value mismatch"+
						"\nwant:%#02x (test.Input.Memory[%#04x])\nhave:%#02x",
					input,
					i,
					value,
				)
			}
		} else if value != 0 {
			// Value was expected to remain unitialized
			t.Fatalf(
				"Memory unexpectedly changed"+
					"\nwant:0x00 (test.Output.Memory[%#04x])\nhave:%#02x",
				i,
				value,
			)
		}
	}

	if len(test.Display) > 0 {
		if have := displayBuf.String(); have != test.Display {
			t.Errorf(
				"Display output mismatch"+
					"\nwant:%s (test.Display)\nhave:%s",
				test.Display,
				have,
			)
		}
	}
}

func testSuccess(t *testing.T, tests []testCase) {
	t.Run("Success", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				testMachineSuccess(t, &test)
			})
		}
	})
}

// ADD  |0001    |DR   |SR1  |0|00 |SR2   | Register  addition
// ADD  |0001    |DR   |SR1  |1|imm5      | Immediate addition
// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
func TestAdd(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name: "ADD SR2 Negative",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x0001, // SR1
					2: 0x8001, // SR2
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0001_000_001_000_010,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b100,
				Registers: [8]uint16{
					0: 0x8002, // DR
					1: 0x0001, // SR1
					2: 0x8001, // SR2
				},
			},
		},
		{
			Name: "ADD SR2 Zero",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x0000, // SR1
					2: 0x0000, // SR2
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0001_000_001_000_010,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b010,
				Registers: [8]uint16{
					0: 0x0000, // DR
					1: 0x0000, // SR1
					2: 0x0000, // SR2
				},
			},
		},
		{
			Name: "ADD Overflow SR2 Zero",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0xFFFF, // SR1
					2: 0x0001, // SR2
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0001_000_001_000_010,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b010,
				Registers: [8]uint16{
					0: 0x0000, // DR
					1: 0xFFFF, // SR1
					2: 0x0001, // SR2
				},
			},
		},
		{
			Name: "ADD SR2 Positive",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x00FF, // SR1
					2: 0x0001, // SR2
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0001_000_001_000_010,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b001,
				Registers: [8]uint16{
					0: 0x0100, // DR
					1: 0x00FF, // SR1
					2: 0x0001, // SR2
				},
			},
		},
		{
			Name: "ADD imm5 Negative",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x8001, // SR1
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0001_000_001_1_00001,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b100,
				Registers: [8]uint16{
					0: 0x8002, // DR
					1: 0x8001, // SR1
				},
			},
		},
		{
			Name: "ADD imm5 Zero",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x0000, // SR1
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0001_000_001_1_00000,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b010,
				Registers: [8]uint16{
					0: 0x0000, // DR
					1: 0x0000, // SR1
				},
			},
		},
		{
			Name: "ADD Overflow imm5 Zero",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0xFFF1, // SR1
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0001_000_001_1_01111,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b010,
				Registers: [8]uint16{
					0: 0x0000, // DR
					1: 0xFFF1, // SR1
				},
			},
		},
		{
			Name: "ADD imm5 Positive",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x0001, // SR1
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0001_000_001_1_00010,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b001,
				Registers: [8]uint16{
					0: 0x0003, // DR
					1: 0x0001, // SR1
				},
			},
		},
	})
}

// AND  |0101    |DR   |SR1  |0|00 |SR2   | Register  bitwise
// AND  |0101    |DR   |SR1  |1|imm5      | Immediate bitwise
// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
func TestAnd(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name: "AND SR2 Negative",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x8001, // SR1
					2: 0x8001, // SR2
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0101_000_001_000_010,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b100,
				Registers: [8]uint16{
					0: 0x8001, // DR
					1: 0x8001, // SR1
					2: 0x8001, // SR2
				},
			},
		},
		{
			Name: "AND SR2 Zero",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x0000, // SR1
					2: 0x1111, // SR2
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0101_000_001_000_010,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b010,
				Registers: [8]uint16{
					0: 0x0000, // DR
					1: 0x0000, // SR1
					2: 0x1111, // SR2
				},
			},
		},
		{
			Name: "AND SR2 Positive",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x0001, // SR1
					2: 0x0001, // SR2
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0101_000_001_000_010,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b001,
				Registers: [8]uint16{
					0: 0x0001, // DR
					1: 0x0001, // SR1
					2: 0x0001, // SR2
				},
			},
		},
		{
			Name: "AND imm5 Negative",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x8001, // SR1
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0101_000_001_1_10001,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b100,
				Registers: [8]uint16{
					0: 0x8001, // DR
					1: 0x8001, // SR1
				},
			},
		},
		{
			Name: "AND imm5 Zero",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x0000, // SR1
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0101_000_001_1_11111,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b010,
				Registers: [8]uint16{
					0: 0x0000, // DR
					1: 0x0000, // SR1
				},
			},
		},
		{
			Name: "AND imm5 Positive",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x0001, // SR1
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0101_000_001_1_00001,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b001,
				Registers: [8]uint16{
					0: 0x0001, // DR
					1: 0x0001, // SR1
				},
			},
		},
	})
}

func TestBranch(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name: "BR Forwards",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b000,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_000_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3081,
				Condition: 0b000,
			},
		},
		{
			Name: "BR Backwards",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b000,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_000_110000000,
				},
			},
			Output: testMachineState{
				Program:   0x2F81, // (0x3000 + 0x2) - 0x180
				Condition: 0b000,
			},
		},
		{
			Name: "BRn True",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b100,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_100_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3081,
				Condition: 0b100,
			},
		},
		{
			Name: "BRn False",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b000,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_100_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b000,
			},
		},
		{
			Name: "BRz True",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b010,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_010_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3081,
				Condition: 0b010,
			},
		},
		{
			Name: "BRz False",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b000,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_010_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b000,
			},
		},
		{
			Name: "BRp True",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b001,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_001_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3081,
				Condition: 0b001,
			},
		},
		{
			Name: "BRp False",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b000,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_001_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b000,
			},
		},
		{
			Name: "BRnz True",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b110,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_110_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3081,
				Condition: 0b110,
			},
		},
		{
			Name: "BRnz False",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b000,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_110_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b000,
			},
		},
		{
			Name: "BRzp True",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b011,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_011_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3081,
				Condition: 0b011,
			},
		},
		{
			Name: "BRzp False",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b000,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_011_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b000,
			},
		},
		{
			Name: "BRnp True",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b101,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_101_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3081,
				Condition: 0b101,
			},
		},
		{
			Name: "BRnp False",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b000,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_101_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b000,
			},
		},
		{
			Name: "BRnpz True",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b111,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_111_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3081,
				Condition: 0b111,
			},
		},
		{
			Name: "BRnpz False",
			Input: testMachineState{
				Program:   0x3000,
				Condition: 0b000,
				Memory: map[uint16]uint16{
					0x3000: 0b0000_111_010000000,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b000,
			},
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
		{
			Name: "JMP",
			Input: testMachineState{
				Privilege: true,
				Program:   0x3000,
				Registers: [8]uint16{
					0: 0x6000, // BaseR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b1100_000_000_000000,
				},
			},
			Output: testMachineState{
				Privilege: true,
				Program:   0x6000,
				Registers: [8]uint16{
					0: 0x6000, // BaseR
				},
			},
		},
		{
			Name: "JMPT",
			Input: testMachineState{
				Privilege: true,
				Program:   0x3000,
				Stack:     0xFE00, // USP
				Registers: [8]uint16{
					0: 0x6000, // BaseR
					6: 0x2FFD, // SSP
				},
				Memory: map[uint16]uint16{
					0x3000: 0b1100_000_000_000001,
				},
			},
			Output: testMachineState{
				Privilege: false,
				Program:   0x6000,
				Stack:     0x2FFD, // SSP
				Registers: [8]uint16{
					0: 0x6000, // BaseR
					6: 0xFE00, // USP
				},
			},
		},
		{
			Name: "JSR Forwards",
			Input: testMachineState{
				Privilege: true,
				Program:   0x3000,
				Memory: map[uint16]uint16{
					0x3000: 0b0100_1_00000010000,
				},
			},
			Output: testMachineState{
				Privilege: true,
				Program:   0x3011,
				Registers: [8]uint16{
					7: 0x3001, // Return Addr
				},
			},
		},
		{
			Name: "JSR Backwards",
			Input: testMachineState{
				Privilege: true,
				Program:   0x3000,
				Memory: map[uint16]uint16{
					0x3000: 0b0100_1_11111111100,
				},
			},
			Output: testMachineState{
				Privilege: true,
				Program:   0x2FFD,
				Registers: [8]uint16{
					7: 0x3001, // Return Addr
				},
			},
		},
		{
			Name: "JSRR",
			Input: testMachineState{
				Privilege: true,
				Program:   0x3000,
				Registers: [8]uint16{
					0: 0x6000, // BaseR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0100_000_000_000000,
				},
			},
			Output: testMachineState{
				Privilege: true,
				Program:   0x6000,
				Registers: [8]uint16{
					0: 0x6000, // BaseR
					7: 0x3001, // Return Addr
				},
			},
		},
		{
			Name: "RET",
			Input: testMachineState{
				Privilege: true,
				Program:   0x3000,
				Registers: [8]uint16{
					7: 0x6000,
				},
				Memory: map[uint16]uint16{
					0x3000: 0b1100_000_111_000000,
				},
			},
			Output: testMachineState{
				Privilege: true,
				Program:   0x6000,
				Registers: [8]uint16{
					7: 0x6000,
				},
			},
		},
		{
			Name: "RTT",
			Input: testMachineState{
				Privilege: true,
				Program:   0x3000,
				Registers: [8]uint16{
					7: 0x6000,
				},
				Memory: map[uint16]uint16{
					0x3000: 0b1100_000_111_000001,
				},
			},
			Output: testMachineState{
				Privilege: false,
				Program:   0x6000,
				Registers: [8]uint16{
					7: 0x6000,
				},
			},
		},
		{
			Name: "RTI",
			Input: testMachineState{
				Privilege: true,
				Program:   0x3000,
				Priority:  1,
				Stack:     0xFDFC, // USP
				Registers: [8]uint16{
					6: 0x2FF9, // SSP
				},
				Memory: map[uint16]uint16{
					0xFDFE: 0x0400, // USP[1], Procstat
					0xFDFC: 0x6000, // USP[0], Program
					0x3000: 0b1000_000000000000,
				},
			},
			Output: testMachineState{
				Privilege: false,
				Program:   0x6000,
				Priority:  4,
				Stack:     0x2FF9, // SSP
				Registers: [8]uint16{
					6: 0xFE00, // USP
				},
			},
		},
		{
			Name: "RTI Privilege Violation",
			Input: testMachineState{
				Privilege: false,
				Priority:  4,
				Program:   0x3000,
				Stack:     0x2FFD, // SSP
				Registers: [8]uint16{
					6: 0xFE00, // USP
					7: 0xDEAD,
				},
				Memory: map[uint16]uint16{
					0x0100: 0x6000,
					0x3000: 0b1000_000000000000,
				},
			},
			Output: testMachineState{
				Privilege: true,
				Program:   0x6000,
				Priority:  4,
				Stack:     0xFDFC, // USP
				Registers: [8]uint16{
					6: 0x2FFD, // SSP
					7: 0xDEAD,
				},
				Memory: map[uint16]uint16{
					0xFDFE: 0x0400, // Procstat
					0xFDFC: 0x3001, // Program
				},
			},
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
		{
			Name: "LD Backwards",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
				},
				Memory: map[uint16]uint16{
					0x2FFC: 0x800F,
					0x3000: 0b0010_000_111111011, // PCoffset9 = -0x5
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b100,
				Registers: [8]uint16{
					0: 0x800F, // DR
				},
			},
		},
		{
			Name: "LD Negative",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0010_000_000010000, // PCoffset9 = 0x10
					0x3011: 0x800F,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b100,
				Registers: [8]uint16{
					0: 0x800F, // DR
				},
			},
		},
		{
			Name: "LD Zero",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0010_000_000010000, // PCoffset9 = 0x10
					0x3011: 0x0000,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b010,
				Registers: [8]uint16{
					0: 0x0000, // DR
				},
			},
		},
		{
			Name: "LD Positive",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0010_000_000010000, // PCoffset9 = 0x10
					0x3011: 0x000F,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b001,
				Registers: [8]uint16{
					0: 0x000F, // DR
				},
			},
		},
		{
			Name: "LDI Backwards",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
				},
				Memory: map[uint16]uint16{
					0x2FFC: 0x6000,
					0x3000: 0b1010_000_111111011, // PCoffset9 = -0x5
					0x6000: 0x800F,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b100,
				Registers: [8]uint16{
					0: 0x800F, // DR
				},
			},
		},
		{
			Name: "LDI Negative",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b1010_000_000010000, // PCoffset9 = 0x10
					0x3011: 0x6000,
					0x6000: 0x800F,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b100,
				Registers: [8]uint16{
					0: 0x800F, // DR
				},
			},
		},
		{
			Name: "LDI Zero",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b1010_000_000010000, // PCoffset9 = 0x10
					0x3011: 0x6000,
					0x6000: 0x0000,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b010,
				Registers: [8]uint16{
					0: 0x0000, // DR
				},
			},
		},
		{
			Name: "LDI Positive",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b1010_000_000010000, // PCoffset9 = 0x10
					0x3011: 0x6000,
					0x6000: 0x000F,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b001,
				Registers: [8]uint16{
					0: 0x000F, // DR
				},
			},
		},
		{
			Name: "LDR Backwards",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x6005, // BaseR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0110_000_001_111011, // offset6 = -0x5
					0x6000: 0x800F,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b100,
				Registers: [8]uint16{
					0: 0x800F, // DR
					1: 0x6005, // BaseR
				},
			},
		},
		{
			Name: "LDR Negative",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x6000, // BaseR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0110_000_001_010000, // offset6 = 0x10
					0x6010: 0x800F,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b100,
				Registers: [8]uint16{
					0: 0x800F, // DR
					1: 0x6000, // BaseR
				},
			},
		},
		{
			Name: "LDR Zero",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x6000, // BaseR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0110_000_001_010000, // offset6 = 0x10
					0x6010: 0x0000,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b010,
				Registers: [8]uint16{
					0: 0x0000, // DR
					1: 0x6000, // BaseR
				},
			},
		},
		{
			Name: "LDR Positive",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x6000, // BaseR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0110_000_001_010000, // offset6 = 0x10
					0x6010: 0x000F,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b001,
				Registers: [8]uint16{
					0: 0x000F, // DR
					1: 0x6000, // BaseR
				},
			},
		},
		{
			Name: "LEA Backwards",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
				},
				Memory: map[uint16]uint16{
					0x2FFC: 0xDEAD,
					0x3000: 0b1110_000_111111011, // PCoffset9 = -0x5
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b001,
				Registers: [8]uint16{
					0: 0x2FFC, // DR
				},
			},
		},
		{
			Name: "LEA Negative",
			Input: testMachineState{
				Program: 0x7F7F,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
				},
				Memory: map[uint16]uint16{
					0x7F7F: 0b1110_000_010000000, // PCoffset9 = 0x80
					0x8000: 0xDEAD,
				},
			},
			Output: testMachineState{
				Program:   0x7F80,
				Condition: 0b100,
				Registers: [8]uint16{
					0: 0x8000, // DR
				},
			},
		},
		{
			Name: "LEA Zero",
			Input: testMachineState{
				Program: 0x007F,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
				},
				Memory: map[uint16]uint16{
					0x0000: 0xDEAD,
					0x007F: 0b1110_000_110000000, // PCoffset9 = -0x80
				},
			},
			Output: testMachineState{
				Program:   0x0080,
				Condition: 0b010,
				Registers: [8]uint16{
					0: 0x0000, // DR
				},
			},
		},
		{
			Name: "LEA Positive",
			Input: testMachineState{
				Program: 0x0000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
				},
				Memory: map[uint16]uint16{
					0x0000: 0b1110_000_001111111, // PCoffset9 = 0x7F
					0x0080: 0xDEAD,
				},
			},
			Output: testMachineState{
				Program:   0x0001,
				Condition: 0b001,
				Registers: [8]uint16{
					0: 0x0080, // DR
				},
			},
		},
		{
			Name: "ST Backwards",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // SR
				},
				Memory: map[uint16]uint16{
					0x2FFC: 0xDEAD,
					0x3000: 0b0011_000_111111011, // PCoffset9 = -0x5
				},
			},
			Output: testMachineState{
				Program: 0x3001,
				Registers: [8]uint16{
					0: 0xCAFE, // SR
				},
				Memory: map[uint16]uint16{
					0x2FFC: 0xCAFE,
				},
			},
		},
		{
			Name: "ST Negative",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // SR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0011_000_000010000, // PCoffset9 = 0x10
					0x3011: 0xDEAD,
				},
			},
			Output: testMachineState{
				Program: 0x3001,
				Registers: [8]uint16{
					0: 0xCAFE, // SR
				},
				Memory: map[uint16]uint16{
					0x3011: 0xCAFE,
				},
			},
		},
		{
			Name: "ST Zero",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0x0000, // SR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0011_000_000010000, // PCoffset9 = 0x10
					0x3011: 0xDEAD,
				},
			},
			Output: testMachineState{
				Program: 0x3001,
				Registers: [8]uint16{
					0: 0x0000, // SR
				},
				Memory: map[uint16]uint16{
					0x3011: 0x0000,
				},
			},
		},
		{
			Name: "ST Positive",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0x000F, // SR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0011_000_000010000, // PCoffset9 = 0x10
					0x3011: 0xDEAD,
				},
			},
			Output: testMachineState{
				Program: 0x3001,
				Registers: [8]uint16{
					0: 0x000F, // SR
				},
				Memory: map[uint16]uint16{
					0x3011: 0x000F,
				},
			},
		},
		{
			Name: "STI Backwards",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // SR
				},
				Memory: map[uint16]uint16{
					0x2FFC: 0x6000,
					0x3000: 0b1011_000_111111011, // PCoffset9 = -0x5
					0x6000: 0xDEAD,
				},
			},
			Output: testMachineState{
				Program: 0x3001,
				Registers: [8]uint16{
					0: 0xCAFE, // SR
				},
				Memory: map[uint16]uint16{
					0x6000: 0xCAFE,
				},
			},
		},
		{
			Name: "STI Negative",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // SR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b1011_000_000010000, // PCoffset9 = 0x10
					0x3011: 0x6000,
					0x6000: 0xDEAD,
				},
			},
			Output: testMachineState{
				Program: 0x3001,
				Registers: [8]uint16{
					0: 0xCAFE, // SR
				},
				Memory: map[uint16]uint16{
					0x6000: 0xCAFE,
				},
			},
		},
		{
			Name: "STI Zero",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0x0000, // SR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b1011_000_000010000, // PCoffset9 = 0x10
					0x3011: 0x6000,
					0x6000: 0xDEAD,
				},
			},
			Output: testMachineState{
				Program: 0x3001,
				Registers: [8]uint16{
					0: 0x0000, // SR
				},
				Memory: map[uint16]uint16{
					0x6000: 0x0000,
				},
			},
		},
		{
			Name: "STI Positive",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0x000F, // SR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b1011_000_000010000, // PCoffset9 = 0x10
					0x3011: 0x6000,
					0x6000: 0xDEAD,
				},
			},
			Output: testMachineState{
				Program: 0x3001,
				Registers: [8]uint16{
					0: 0x000F, // SR
				},
				Memory: map[uint16]uint16{
					0x6000: 0x000F,
				},
			},
		},
		{
			Name: "STR Backwards",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // SR
					1: 0x6005, // BaseR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0111_000_001_111011, // offset6 = -0x5
					0x6000: 0xDEAD,
				},
			},
			Output: testMachineState{
				Program: 0x3001,
				Registers: [8]uint16{
					0: 0xCAFE, // SR
					1: 0x6005, // BaseR
				},
				Memory: map[uint16]uint16{
					0x6000: 0xCAFE,
				},
			},
		},
		{
			Name: "STR Negative",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // SR
					1: 0x6000, // BaseR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0111_000_001_010000, // offset6 = 0x10
					0x6010: 0xDEAD,
				},
			},
			Output: testMachineState{
				Program: 0x3001,
				Registers: [8]uint16{
					0: 0xCAFE, // SR
					1: 0x6000, // BaseR
				},
				Memory: map[uint16]uint16{
					0x6010: 0xCAFE,
				},
			},
		},
		{
			Name: "STR Zero",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0x0000, // SR
					1: 0x6000, // BaseR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0111_000_001_010000, // offset6 = 0x10
					0x6010: 0xDEAD,
				},
			},
			Output: testMachineState{
				Program: 0x3001,
				Registers: [8]uint16{
					0: 0x0000, // SR
					1: 0x6000, // BaseR
				},
				Memory: map[uint16]uint16{
					0x6010: 0x0000,
				},
			},
		},
		{
			Name: "STR Positive",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0x000F, // SR
					1: 0x6000, // BaseR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b0111_000_001_010000, // offset6 = 0x10
					0x6010: 0xDEAD,
				},
			},
			Output: testMachineState{
				Program: 0x3001,
				Registers: [8]uint16{
					0: 0x000F, // SR
					1: 0x6000, // BaseR
				},
				Memory: map[uint16]uint16{
					0x6010: 0x000F,
				},
			},
		},
	})
}

// NOT  |1001    |DR   |SR   |1|11111     | Bitwise complement
// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
func TestNot(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name: "NOT SR Negative",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0x0FFF, // SR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b1001_000_001_1_11111,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b100,
				Registers: [8]uint16{
					0: 0xF000, // DR
					1: 0x0FFF, // SR
				},
			},
		},
		{
			Name: "NOT SR Zero",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0xFFFF, // SR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b1001_000_001_1_11111,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b010,
				Registers: [8]uint16{
					0: 0x0000, // DR
					1: 0xFFFF, // SR
				},
			},
		},
		{
			Name: "NOT SR Positive",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xCAFE, // DR
					1: 0xF000, // SR
				},
				Memory: map[uint16]uint16{
					0x3000: 0b1001_000_001_1_11111,
				},
			},
			Output: testMachineState{
				Program:   0x3001,
				Condition: 0b001,
				Registers: [8]uint16{
					0: 0x0FFF, // DR
					1: 0xF000, // SR
				},
			},
		},
	})
}

// TRAP |1111    |0000   |trapvect8       | Store base+offset
// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
func TestTrap(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name: "TRAP",
			Input: testMachineState{
				Program: 0x3000,
				Stack:   0x2FFD, // SSP
				Registers: [8]uint16{
					6: 0xFE00, // USP
					7: 0xCAFE,
				},
				Memory: map[uint16]uint16{
					0x0010: 0x6000, // TRAP Vector value
					0x3000: 0b1111_0000_00010000,
				},
			},
			Output: testMachineState{
				Privilege: true,
				Program:   0x6000,
				Stack:     0xFE00, // USP
				Registers: [8]uint16{
					6: 0x2FFD, // SSP
					7: 0x3001,
				},
			},
		},
	})
}

// RES  |1101    |                        | Reserved (illegal)
// ---- [ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ ]
func TestReserved(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name: "RES Illegal Opcode",
			Input: testMachineState{
				Privilege: false,
				Priority:  4,
				Program:   0x3000,
				Stack:     0x2FFD, // SSP
				Registers: [8]uint16{
					6: 0xFE00, // USP
				},
				Memory: map[uint16]uint16{
					0x0101: 0x6000,
					0x3000: 0b1101_000000000000,
				},
			},
			Output: testMachineState{
				Privilege: true,
				Program:   0x6000,
				Priority:  4,
				Stack:     0xFDFC, // USP
				Registers: [8]uint16{
					6: 0x2FFD, // SSP
				},
				Memory: map[uint16]uint16{
					0xFDFE: 0x0400, // Procstat
					0xFDFC: 0x3001, // Program
				},
			},
		},
	})
}

func TestInterrupt(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name:     "Interrupt Low Priority Process",
			Keyboard: "foobar",
			Input: testMachineState{
				Privilege: false,
				Priority:  1,
				Program:   0x3000,
				Stack:     0x2FFD, // SSP
				Registers: [8]uint16{
					6: 0xFE00, // USP
				},
				Memory: map[uint16]uint16{
					0x0180: 0x6000,              // Interrupt Handler Address
					0x3000: 0b0000_000_00000000, // BR 0x0
				},
			},
			Output: testMachineState{
				Privilege: true,
				Priority:  4,
				Program:   0x6000,
				Stack:     0xFDFC, // USP
				Registers: [8]uint16{
					6: 0x2FFD, // SSP
				},
				Memory: map[uint16]uint16{
					0xFDFE: 0x0100, // Procstat
					0xFDFC: 0x3001, // Program (after BR)
				},
			},
		},
		{
			Name:     "Interrupt High Priority Process",
			Keyboard: "foobar",
			Input: testMachineState{
				Privilege: false,
				Priority:  5,
				Program:   0x3000,
				Registers: [8]uint16{
					6: 0xFE00, // SSP
				},
				Memory: map[uint16]uint16{
					0x0180: 0x6000,              // Interrupt Handler Address
					0x3000: 0b0000_000_00000000, // BR 0x0
				},
			},
			Output: testMachineState{
				Privilege: false,
				Priority:  5,
				Program:   0x3001,
				Registers: [8]uint16{
					6: 0xFE00, // SSP
				},
			},
		},
	})
}

func TestKeyboard(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name:     "Read Keyboard",
			Steps:    2,
			Keyboard: "foobar",
			Input: testMachineState{
				Priority: 7, // Ignore interrupt
				Program:  0x3000,
				Registers: [8]uint16{
					0: 0xDEAD, // LDR[0] DR
					1: 0xFE00, // LDR[0] BaseR (Keyboard Status Register)
					2: 0xDEAD, // LDR[1] DR
					3: 0xFE02, // LDR[1] BaseR (Keyboard Data Register)
				},
				Memory: map[uint16]uint16{
					// LDR R0 R1 0x0
					0x3000: 0b0110_000_001_000000,
					// LDR R2 R3 0x0
					0x3001: 0b0110_010_011_000000,
					// Uninitialized KBSR
					0xFE00: 0x0000,
					// Uninitialized KBDR
					0xFE02: 0x0000,
				},
			},
			Output: testMachineState{
				Priority:  7,
				Program:   0x3002,
				Condition: 0b001, // Positive LDR[1] DR (#102)
				Registers: [8]uint16{
					0: 0x8000, // LDR[0] DR (KBSR: 1 << 15)
					1: 0xFE00, // LDR[0] BaseR (Keyboard Status Register)
					2: 0x0066, // LDR[1] DR (KBDR: 'f', #102)
					3: 0xFE02, // LDR[1] BaseR (Keyboard Data Register)
				},
				Memory: map[uint16]uint16{
					// KBSR: 1 << 15
					0xFE00: 0x8000,
					// KBDR: 'f', #102
					0xFE02: 0x0066,
				},
			},
		},
	})
}

func TestDisplay(t *testing.T) {
	testSuccess(t, []testCase{
		{
			Name:    "Write Display",
			Steps:   8,
			Display: "aaa",
			Input: testMachineState{
				Program: 0x3000,
				Registers: [8]uint16{
					0: 0xDEAD, // LDR DR
					1: 0xFE04, // LDR BaseR (Display Status Register)
					2: 0x0061, // STR SR ('a', #97)
					3: 0xFE06, // STR BaseR (Display Data Register)
					4: 0x3000, // JMP BaseR
				},
				Memory: map[uint16]uint16{
					// LDR R0 R1 0x0
					0x3000: 0b0110_000_001_000000,
					// STR R2 R3 0x0
					0x3001: 0b0111_010_011_000000,
					// JMP R4
					0x3002: 0b1100_000_100_000000,
				},
			},
			Output: testMachineState{
				Program:   0x3002,
				Condition: 0b100, // Negative LDR DR (1<<15)
				Registers: [8]uint16{
					0: 0x8000, // LDR DR (DSR: 1 << 15)
					1: 0xFE04, // LDR BaseR (Display Status Register)
					2: 0x0061, // STR SR ('a', #97)
					3: 0xFE06, // STR BaseR (Display Data Register)
					4: 0x3000, // JMP BaseR
				},
				Memory: map[uint16]uint16{
					// DSR: 1 << 15
					0xFE04: 0x8000,
					// DDR: Contains last written character
					0xFE06: 0x0061,
				},
			},
		},
	})
}
