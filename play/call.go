package play

type BuiltinFunc struct {
	Ident string
	Func  func([]Value) (Value, error)
}

func execParseInt(args []Value) (Value, error) {
	return nil, nil
}

func execParseFloat(args []Value) (Value, error) {
	return nil, nil
}

func execIsNaN(args []Value) (Value, error) {
	if len(args) != 1 {
		return getBool(true), nil
	}
	v, ok := args[0].(Float)
	if !ok {
		return getBool(false), nil
	}
	return getBool(math.IsNaN(v.value)), nil
}

func NewBuiltinFunc(ident string, fn func([]Value) (Value, error)) Value {
	return createBuiltinFunc(ident, fn)
}

func createBuiltinFunc(ident string, fn func([]Value) (Value, error)) Value {
	return BuiltinFunc{
		Ident: ident,
		Func:  fn,
	}
}

func (b BuiltinFunc) True() Value {
	return getBool(true)
}

func (b BuiltinFunc) Call(args []Value) (Value, error) {
	return b.Func(args)
}

type Parameter struct {
	Name string
	Value
}

type Function struct {
	Ident string
	Arrow bool
	Args  []Parameter
	Body  Node
	Env   environ.Environment[Value]
}

func (f Function) True() Value {
	return getBool(true)
}

func (f Function) Call(args []Value) (Value, error) {
	for i := range f.Args {
		var arg Value
		if i < len(args) {
			arg = args[i]
			if arg == nil || isNull(arg) || isUndefined(arg) {
				arg = f.Args[i].Value
			}
		} else {
			arg = f.Args[i].Value
		}
		if err := f.Env.Define(f.Args[i].Name, arg); err != nil {
			return nil, err
		}
	}
	arr := createArray()
	arr.Values = append(arr.Values, args...)
	f.Env.Define("arguments", arr)

	return eval(f.Body, f.Env)
}