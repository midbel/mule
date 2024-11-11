package json

import (
	"fmt"
)

const (
	powLowest = iota
	powComma
	powTernary
	powOr
	powAnd
	powCmp
	powEq
	powAdd
	powMul
	powPrefix
	powCall
	powGrp
	powMap
	powFilter
	powTransform
)

var bindings = map[rune]int{
	BegGrp:    powCall,
	BegArr:    powFilter,
	BegObj:    powFilter,
	Ternary:   powTernary,
	And:       powAnd,
	Or:        powOr,
	Add:       powAdd,
	Sub:       powAdd,
	Mul:       powMul,
	Div:       powMul,
	Mod:       powMul,
	Wildcard:  powMul,
	Parent:    powMul,
	Eq:        powEq,
	Ne:        powEq,
	In:        powCmp,
	Lt:        powCmp,
	Le:        powCmp,
	Gt:        powCmp,
	Ge:        powCmp,
	Concat:    powAdd,
	Map:       powMap,
	Transform: powTransform,
}

const (
	EOF = -(1 + iota)
	BegArr
	EndArr
	BegObj
	EndObj
	Comma
	Colon
	Boolean
	Null
	String
	Number
	Ident
	Func
	Comment
	// query token
	Doc
	BegGrp
	EndGrp
	In
	And
	Or
	Add
	Sub
	Mul
	Div
	Mod
	Eq
	Ne
	Lt
	Le
	Gt
	Ge
	Concat
	Ternary
	Map
	Parent
	Wildcard
	Descent
	Range
	Transform
	// common
	Invalid
)

type Token struct {
	Literal string
	Type    rune
}

func (t Token) String() string {
	var prefix string
	switch t.Type {
	case Transform:
		return "<transform>"
	case Doc:
		return "<document>"
	case Ternary:
		return "<ternary>"
	case Colon:
		return "<colon>"
	case BegGrp:
		return "<beg-grp>"
	case EndGrp:
		return "<end-grp>"
	case And:
		return "<and>"
	case Or:
		return "<or>"
	case In:
		return "<in>"
	case Add:
		return "<add>"
	case Sub:
		return "<subtract>"
	case Mul:
		return "<multiply>"
	case Div:
		return "<divide>"
	case Mod:
		return "<modulo>"
	case Eq:
		return "<equal>"
	case Ne:
		return "<not-equal>"
	case Lt:
		return "<lesser-than>"
	case Le:
		return "<lesser-eq>"
	case Gt:
		return "<greater-than>"
	case Ge:
		return "<greater-eq>"
	case Concat:
		return "<concat>"
	case Map:
		return "<map>"
	case Parent:
		return "<parent>"
	case Wildcard:
		return "<wildcard>"
	case Descent:
		return "<descend>"
	case Range:
		return "<range>"
	case EOF:
		return "<eof>"
	case BegArr:
		return "<beg-arr>"
	case EndArr:
		return "<end-arr>"
	case BegObj:
		return "<beg-obj>"
	case EndObj:
		return "<end-obj>"
	case Comma:
		return "<comma>"
	case Boolean:
		prefix = "boolean"
	case Null:
		return "<null>"
	case String:
		prefix = "string"
	case Number:
		prefix = "number"
	case Ident:
		prefix = "identifier"
	case Func:
		prefix = "function"
	case Comment:
		prefix = "comment"
	case Invalid:
		prefix = "invalid"
	}
	return fmt.Sprintf("%s(%s)", prefix, t.Literal)
}

func isComment(c, k rune) bool {
	return c == '/' && c == k
}

func isHex(c rune) bool {
	return isNumber(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isNumber(c rune) bool {
	return c >= '0' && c <= '9'
}

func isLower(c rune) bool {
	return c >= 'a' && c <= 'z'
}

func isUpper(c rune) bool {
	return c >= 'A' && c <= 'Z'
}

func isLetter(c rune) bool {
	return isLower(c) || isUpper(c)
}

func isAlpha(c rune) bool {
	return isLetter(c) || isNumber(c) || c == '_'
}

func isApos(c rune) bool {
	return c == '\''
}

func isQuote(c rune) bool {
	return c == '"'
}

func isBackQuote(c rune) bool {
	return c == '`'
}

func isDelim(c rune) bool {
	return c == '{' || c == '}' || c == '[' || c == ']' || c == ',' || c == ':'
}

func isNL(c rune) bool {
	return c == '\n' || c == '\r'
}

func isOperator(c rune) bool {
	return c == '!' || c == '=' || c == '<' || c == '>' ||
		c == '&' || c == '*' || c == '/' || c == '%' || c == '-' ||
		c == '+' || c == '.' || c == '?' || c == ':'
}

func isTransform(c rune) bool {
	return c == '|'
}

func isDollar(c rune) bool {
	return c == '$'
}
