package play

import "fmt"

const (
	EOF = -(iota + 1)
	EOL
	Keyword
	Ident
	Text
	Number
	Boolean
	Invalid
	New
	TypeOf
	InstanceOf
	Decorate
	Del
	Optional
	Nullish
	Incr
	Decr
	Add
	Sub
	Mul
	Div
	Mod
	Pow
	Assign
	Not
	Eq
	Seq
	Ne
	Sne
	Lt
	Le
	Gt
	Ge
	And
	Or
	Arrow
	Spread
	Dot
	Comma
	Question
	Colon
	Lparen
	Rparen
	Lsquare
	Rsquare
	Lcurly
	Rcurly
)

var keywords = []string{
	"let",
	"const",
	"using",
	"break",
	"continue",
	"for",
	"in",
	"of",
	"if",
	"else",
	"switch",
	"case",
	"default",
	"while",
	"do",
	"function",
	"return",
	"import",
	"export",
	"from",
	"as",
	"true",
	"false",
	"try",
	"catch",
	"finally",
	"throw",
	"null",
	"undefined",
	"instanceof",
	"typeof",
	"with",
}

type Position struct {
	Line   int
	Column int
}

type Token struct {
	Literal string
	Type    rune
	Position
}

func (t Token) String() string {
	var prefix string
	switch t.Type {
	case EOF:
		return "<eof>"
	case EOL:
		return "<eol>"
	case Spread:
		return "spread"
	case Dot:
		return "<dot>"
	case Arrow:
		return "<arrow>"
	case Comma:
		return "<comma>"
	case Lparen:
		return "<lparen>"
	case Rparen:
		return "<rparen>"
	case Lsquare:
		return "<lsquare>"
	case Rsquare:
		return "<rsquare>"
	case Lcurly:
		return "<lcurly>"
	case Rcurly:
		return "<rcurly>"
	case Incr:
		return "<incr>"
	case Decr:
		return "<decr>"
	case Add:
		return "<add>"
	case Sub:
		return "<sub>"
	case Mul:
		return "<mul>"
	case Div:
		return "<div>"
	case Mod:
		return "<mod>"
	case Pow:
		return "<pow>"
	case Assign:
		return "<assign>"
	case Not:
		return "<not>"
	case Eq:
		return "<eq>"
	case Seq:
		return "<seq>"
	case Ne:
		return "<ne>"
	case Sne:
		return "<sne>"
	case Lt:
		return "<lt>"
	case Le:
		return "<le>"
	case Gt:
		return "<gt>"
	case Ge:
		return "<ge>"
	case And:
		return "<and>"
	case Or:
		return "<or>"
	case New:
		return "<new>"
	case TypeOf:
		return "<typeof>"
	case InstanceOf:
		return "<instanceof>"
	case Del:
		return "<delete>"
	case Question:
		return "<question>"
	case Nullish:
		return "<nullish>"
	case Optional:
		return "<optional>"
	case Colon:
		return "<colon>"
	case Decorate:
		return "<decorator>"
	case Keyword:
		prefix = "keyword"
	case Boolean:
		prefix = "boolean"
	case Ident:
		prefix = "identifier"
	case Text:
		prefix = "string"
	case Number:
		prefix = "number"
	case Invalid:
		prefix = "invalid"
	default:
		prefix = "unknown"
	}
	return fmt.Sprintf("%s(%s)", prefix, t.Literal)
}
