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

; Ignore Privilege Exception
.ORIG 0x100
.FILL INT_NOOP

; Keyboard Interrupt
.ORIG 0x180
.FILL HANDLE_KEY

; Operating System Entry
.ORIG 0x200
LD R7 MEMSPACE_USER
JMPT R7

HANDLE_KEY
    LDI R0 DEVICE_KBSR ; Read keyboard, has side effect of filling KBDR in VM
    LDI R1 DEVICE_KBDR ; Read keyboard data value
    STI R1 DEVICE_DDR  ; Print key back out to device
    RTI

MEMSPACE_USER .FILL 0x3000
DEVICE_KBSR   .FILL 0xFE00
DEVICE_KBDR   .FILL 0xFE02
DEVICE_DDR    .FILL 0xFE06

; User Memory Space
.ORIG 0x3000
LOOP
    JSR LOOP

INT_NOOP RTI

.END
