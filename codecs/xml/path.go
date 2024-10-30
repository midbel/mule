package xml

type Expr interface {
	Eval(Node) ([]Node, error)
}

func Compile(query string) (Expr, error) {
	return nil, nil
}
