package play

type Node interface{}

type Body struct {
	Nodes []Node
}

type Null struct {
	Position
}

type Undefined struct {
	Position
}

type List struct {
	Position
	Nodes []Node
}

type Map struct {
	Position
	Nodes map[Node]Node
}

type Literal[T string | float64 | bool] struct {
	Value T
	Position
}

type Group struct {
	Nodes []Node
	Position
}

type Identifier struct {
	Name string
	Position
}

type Index struct {
	Ident Node
	Expr  Node
	Position
}

type Access struct {
	Ident Node
	Node
	Position
}

type Unary struct {
	Op rune
	Node
	Position
}

type Binary struct {
	Op    rune
	Left  Node
	Right Node
	Position
}

type Assignment struct {
	Ident Node
	Node
}

type Let struct {
	Node
	Position
}

type Const struct {
	Node
	Position
}

type Increment struct {
	Node
	Post bool
	Position
}

type Decrement struct {
	Node
	Post bool
	Position
}

type If struct {
	Cdt Node
	Csq Node
	Alt Node
	Position
}

type Switch struct {
	Cdt     Node
	Cases   []Node
	Default Node
	Position
}

type Case struct {
	Value Node
	Body  Node
	Position
}

type Do struct {
	Cdt  Node
	Body Node
	Position
}

type While struct {
	Cdt  Node
	Body Node
	Position
}

type OfCtrl struct {
	Ident Node
	Iter  Node
}

type InCtrl struct {
	Ident Node
	Iter  Node
}

type ForCtrl struct {
	Init  Node
	Cdt   Node
	After Node
}

type For struct {
	Ctrl Node
	Body Node
	Position
}

type Break struct {
	Label string
	Position
}

type Continue struct {
	Label string
	Position
}

type Try struct {
	Node
	Catch   Node
	Finally Node
	Position
}

type Catch struct {
	Err  Node
	Body Node
	Position
}

type Throw struct {
	Node
	Position
}

type Return struct {
	Node
	Position
}

type Call struct {
	Ident Node
	Args  []Node
	Position
}

type Func struct {
	Ident string
	Args  []Node
	Body  Node
	Arrow bool
	Position
}