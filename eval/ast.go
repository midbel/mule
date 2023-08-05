package eval

type Expression interface{}

type Primitive[T bool | float64 | string] struct {
	Literal T
}

func createString(str string) Primitive[string] {
	return Primitive[string]{
		Literal: str,
	}
}

func createNumber(v float64) Primitive[float64] {
	return Primitive[float64]{
		Literal: v,
	}
}

func createBool(b bool) Primitive[bool] {
	return Primitive[bool]{
		Literal: b,
	}
}

type Variable struct {
	Ident string
}

type Null struct{}

type Block struct {
	List []Expression
}

type Array struct {
	List []Expression
}

type Object struct {
	List map[Expression]Expression
}

type Binary struct {
	Op    rune
	Left  Expression
	Right Expression
}

type Unary struct {
	Op    rune
	Right Expression
}

type Call struct {
	Ident string
	Args  []Expression
}

type Index struct {
	Expr  Expression
	Index Expression
}

type Let struct {
	Ident string
	Expr  Expression
}

type Argument struct {
	Ident   string
	Default Expression
}

type Function struct {
	Name string
	Args []Expression
	Body Expression
}

type Return struct {
	Expression
}

type If struct {
	Cdt Expression
	Csq Expression
	Alt Expression
}

type Switch struct {
	Cdt   Expression
	Cases []Expression
}

type Case struct {
	Value Expression
	Body  Expression
}

type For struct {
	Init Expression
	Cdt  Expression
	Incr Expression
	Body Expression
}

type While struct {
	Do   bool
	Cdt  Expression
	Body Expression
}

type Break struct {
	Label string
}

type Continue struct {
	Label string
}

type Label struct {
	Name string
}
