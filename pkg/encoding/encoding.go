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

package encoding

import (
	"errors"
	"strconv"
	"strings"
)

// Decodes a hexidecimal string in the formats: 0xFFFF, xFFFF, 0xFF, xFF
func DecodeHex(s string) (uint16, error) {
	if i := strings.IndexAny(s, "xX"); i == 0 {
		s = "0" + s
	} else if i == -1 || i != 1 {
		return 0, errors.New("Invalid hex string")
	}

	result, err := strconv.ParseUint(s, 0, 16)

	if err != nil {
		return 0, err
	}

	return uint16(result), nil
}

// Decodes a base-10 string in the formats: #123, 123
func DecodeInt(s string) (int16, error) {
	if i := strings.Index(s, "#"); i == 0 {
		s = s[1:]
	}

	result, err := strconv.ParseInt(s, 10, 16)

	if err != nil {
		return 0, err
	}

	return int16(result), nil
}

func SwapEndian(value uint16) uint16 {
	return (value >> 8) | (value << 8)
}

func SignExtend(value uint16, bitcount uint16) uint16 {
	if (value>>(bitcount-1))&0x1 == 1 {
		value |= (0xFFFF << bitcount)
	}

	return value
}

func ZeroExtend(value uint16, bitcount uint16) uint16 {
	if (value>>(bitcount-1))&0x1 == 1 {
		value |= (0x0000 << bitcount)
	}

	return value
}
