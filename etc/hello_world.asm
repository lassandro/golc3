; Copyright (C) 2021  Antonio Lassandro

; This program is free software: you can redistribute it and/or modify it
; under the terms of the GNU General Public License as published by the Free
; Software Foundation, either version 3 of the License, or (at your option)
; any later version.

; This program is distributed in the hope that it will be useful, but WITHOUT
; ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
; FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for
; more details.

; You should have received a copy of the GNU General Public License along
; with this program.  If not, see <http://www.gnu.org/licenses/>.

; OUT/PUTS Trap Vectors
.ORIG 0x21
.FILL TRAP_OUT
.FILL TRAP_PUTS

; Ignore Privilege Exception
.ORIG 0x100
.FILL INT_NOOP

; Ignore Keyboard Interrupt
.ORIG 0x180
.FILL INT_NOOP

.ORIG 0x200
LD R7 GREET_USER
JMPT R7

GREET_USER .FILL 0x3000
DEVICE_DSR .FILL 0xFE04
DEVICE_DDR .FILL 0xFE06

; Write R0 to the display
TRAP_OUT
    LDI R2, DEVICE_DSR
    BRzp TRAP_OUT  ; Wait for ready
    STI R0, DEVICE_DDR
    RET

TRAP_PUTS
    ADD R1, R0, #0 ; Move string addr to r1

PUTS_LOOP
    LDR R0, R1, #0
    BRz PUTS_DONE
    OUT
    ADD R1, R1, #1
    JSR PUTS_LOOP

PUTS_DONE
    RET

.ORIG 0x3000
LEA R0 GREETING
PUTS

LOOP
    JSR LOOP

GREETING .STRINGZ "Hello world!\n"
INT_NOOP RTI
.END
