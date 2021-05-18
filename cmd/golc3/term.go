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

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

var termRestore unix.Termios

func enterRawTerm() {
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)

	if err != nil {
		panic(err)
	}

	termRestore = *termios
	termstate := *termios

	termstate.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.INLCR
	termstate.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.IEXTEN
	termstate.Cflag &^= unix.CSIZE | unix.PARENB
	termstate.Cflag |= unix.CS8

	termstate.Cc[unix.VMIN] = 0
	termstate.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(
		int(os.Stdin.Fd()), unix.TIOCSETA, &termstate,
	); err != nil {
		panic(err)
	}
}

func exitRawTerm() {
	if err := unix.IoctlSetTermios(
		int(os.Stdin.Fd()), unix.TIOCSETA, &termRestore,
	); err != nil {
		panic(err)
	}
}
