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
	"fmt"
	"strings"
)

type LiteralType uint
type TokenType uint
type InstructionType uint
type DirectiveType uint

type Cursor struct {
	Line     int
	Column   int
	Byte     int64
	Size     int64
	LineByte int64
}

type Token struct {
	Type     TokenType
	Position Cursor
	Value    string
}

type SymTable struct {
	Source string
	Symbols map[uint16]int64
	Labels map[uint16]string
}

type TokenError interface {
	GetPosition() Cursor
}

type InvalidOperandError struct {
	Position Cursor
	Required []TokenType
	Received TokenType
}

func (err *InvalidOperandError) GetPosition() Cursor {
	return err.Position
}

func (err *InvalidOperandError) Error() string {
	var requiredString string
	var receivedString string

	requiredStrings := make([]string, 0, len(err.Required))

	for _, tokenType := range err.Required {
		switch tokenType {
		case TOKEN_IDENT:
			requiredStrings = append(requiredStrings, "Identifier")
		case TOKEN_DIRECTIVE:
			requiredStrings = append(requiredStrings, "Directive")
		case TOKEN_STRING:
			requiredStrings = append(requiredStrings, "String")
		case TOKEN_LITERAL:
			requiredStrings = append(requiredStrings, "Literal")
		default:
			requiredStrings = append(requiredStrings, "<invalid>")
		}
	}

	if count := len(requiredStrings); count == 1 {
		requiredString = requiredStrings[0]
	} else if count == 2 {
		requiredString = requiredStrings[0] + " or " + requiredStrings[1]
	} else if count > 2 {
		requiredString = strings.Join(
			requiredStrings[:len(requiredStrings)-1], ", ",
		) + ", or " + requiredStrings[len(requiredStrings)-1]
	}

	switch err.Received {
	case TOKEN_IDENT:
		receivedString = "Identifier"
	case TOKEN_DIRECTIVE:
		receivedString = "Directive"
	case TOKEN_STRING:
		receivedString = "String"
	case TOKEN_LITERAL:
		receivedString = "Literal"
	default:
		receivedString = "<invalid>"
	}

	return fmt.Sprintf(
		"%02d:%02d: Invalid operands\n\twant:%s\n\thave:%s",
		err.Position.Line,
		err.Position.Column,
		requiredString,
		receivedString,
	)
}

type InvalidNumArgumentsError struct {
	Position Cursor
	Required int
	Received int
}

func (err *InvalidNumArgumentsError) GetPosition() Cursor {
	return err.Position
}

func (err *InvalidNumArgumentsError) Error() string {
	return fmt.Sprintf(
		"%02d:%02d: Invalid number of arguments\n\twant:%d\n\thave:%v",
		err.Position.Line,
		err.Position.Column,
		err.Required,
		err.Received,
	)
}

type OversizedLabelError struct {
	Position Cursor
	Required int64
	Received int64
}

func (err *OversizedLabelError) GetPosition() Cursor {
	return err.Position
}

func (err *OversizedLabelError) Error() string {
	return fmt.Sprintf(
		"%02d:%02d: Label exceeds allowed distance\n\twant:%d\n\thave:%d",
		err.Position.Line,
		err.Position.Column,
		err.Required,
		err.Received,
	)
}

type InvalidLiteralError struct {
	Position Cursor
}

func (err *InvalidLiteralError) GetPosition() Cursor {
	return err.Position
}

func (err *InvalidLiteralError) Error() string {
	return fmt.Sprintf(
		"%02d:%02d: Invalid numeric literal",
		err.Position.Line,
		err.Position.Column,
	)
}

type InvalidStringError struct {
	Position Cursor
}

func (err *InvalidStringError) GetPosition() Cursor {
	return err.Position
}

func (err *InvalidStringError) Error() string {
	return fmt.Sprintf(
		"%02d:%02d: Invalid string literal",
		err.Position.Line,
		err.Position.Column,
	)
}

type OversizedLiteralError struct {
	Position Cursor
	Required interface{}
	Received interface{}
}

func (err *OversizedLiteralError) GetPosition() Cursor {
	return err.Position
}

func (err *OversizedLiteralError) Error() string {
	return fmt.Sprintf(
		"%02d:%02d: Literal exceeds allowed size\n\twant:%d\n\thave:%d",
		err.Position.Line,
		err.Position.Column,
		err.Required,
		err.Received,
	)
}

type InvalidRegisterError struct {
	Position Cursor
}

func (err *InvalidRegisterError) GetPosition() Cursor {
	return err.Position
}

func (err *InvalidRegisterError) Error() string {
	return fmt.Sprintf(
		"%02d:%02d: Invalid register identifier",
		err.Position.Line,
		err.Position.Column,
	)
}

type UnexpectedCharacterError struct {
	Position Cursor
	Received rune
}

func (err *UnexpectedCharacterError) GetPosition() Cursor {
	return err.Position
}

func (err *UnexpectedCharacterError) Error() string {
	return fmt.Sprintf(
		"%02d:%02d: Unexpected character %c",
		err.Position.Line,
		err.Position.Column,
		err.Received,
	)
}

type OversizedCharacterError struct {
	Position Cursor
}

func (err *OversizedCharacterError) GetPosition() Cursor {
	return err.Position
}

func (err *OversizedCharacterError) Error() string {
	return fmt.Sprintf(
		"%02d:%02d: Character exceeds ASCII limit",
		err.Position.Line,
		err.Position.Column,
	)
}

type RedeclaredLabelError struct {
	Position Cursor
	Received string
}

func (err *RedeclaredLabelError) GetPosition() Cursor {
	return err.Position
}

func (err *RedeclaredLabelError) Error() string {
	return fmt.Sprintf(
		"%02d:%02d: Redeclaration of label '%s'",
		err.Position.Line,
		err.Position.Column,
		err.Received,
	)
}

type UnknownLabelError struct {
	Position Cursor
	Received string
}

func (err *UnknownLabelError) GetPosition() Cursor {
	return err.Position
}

func (err *UnknownLabelError) Error() string {
	return fmt.Sprintf(
		"%02d:%02d: Unknown label '%s'",
		err.Position.Line,
		err.Position.Column,
		err.Received,
	)
}

type UnknownIdentifierError struct {
	Position Cursor
	Received string
}

func (err *UnknownIdentifierError) GetPosition() Cursor {
	return err.Position
}

func (err *UnknownIdentifierError) Error() string {
	return fmt.Sprintf(
		"%02d:%02d: Unknown identifier '%s'",
		err.Position.Line,
		err.Position.Column,
		err.Received,
	)
}

type OversizedBinaryError struct{}

func (err *OversizedBinaryError) Error() string {
	return "Binary exceeds allowed size"
}
