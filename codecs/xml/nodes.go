package xml

import (
	"bytes"
	"fmt"
	"io"
	"slices"
)

type Node interface {
	LocalName() string
	QName() string
	Leaf() bool
	Position() int
	Parent() Node
	Value() string

	setParent(Node)
	setPosition(int)
}

type Attribute struct {
	Namespace string
	Name      string
	Value     string
}

func NewAttribute(value, name, namespace string) Attribute {
	return Attribute{
		Name:      name,
		Namespace: namespace,
		Value:     value,
	}
}

type Element struct {
	Namespace string
	Name      string
	Attrs     []Attribute
	Nodes     []Node

	parent   Node
	position int
}

func NewElement(name, namespace string) *Element {
	return &Element{
		Name:      name,
		Namespace: namespace,
	}
}

func (e *Element) LocalName() string {
	return e.Name
}

func (e *Element) QName() string {
	if e.Namespace == "" {
		return e.LocalName()
	}
	return fmt.Sprintf("%s:%s", e.Namespace, e.Name)
}

func (e *Element) Root() bool {
	return e.parent == nil
}

func (e *Element) Leaf() bool {
	return len(e.Nodes) == 0
}

func (e *Element) Value() string {
	if len(e.Nodes) != 1 {
		return ""
	}
	el, ok := e.Nodes[0].(*Text)
	if !ok {
		return ""
	}
	return el.Content
}

func (e *Element) Has(name string) bool {
	return e.Find(name, 0) != nil
}

func (e *Element) Find(name string, depth int) Node {
	ix := slices.IndexFunc(e.Nodes, func(n Node) bool {
		return n.LocalName() == name
	})
	if ix < 0 {
		return nil
	}
	return e.Nodes[ix]
}

func (e *Element) FindAll(name string, depth int) []Node {
	return nil
}

func (e *Element) GetElementById(id string) (Node, error) {
	return nil, nil
}

func (e *Element) GetElementsByTagName(tag string) ([]Node, error) {
	return nil, nil
}

func (e *Element) Append(node Node) {
	node.setParent(e)
	node.setPosition(len(e.Nodes))
	e.Nodes = append(e.Nodes, node)
}

func (e *Element) Insert(node Node, index int) {
	if index < 0 || index > len(e.Nodes) {
		return
	}
	e.Nodes = slices.Insert(e.Nodes, index, node)
}

func (e *Element) Len() int {
	return len(e.Nodes)
}

func (e *Element) Clear() {
	for i := range e.Nodes {
		e.Nodes[i].setParent(nil)
	}
	e.Nodes = e.Nodes[:0]
}

func (e *Element) Position() int {
	return e.position
}

func (e *Element) Parent() Node {
	return e.parent
}

func (e *Element) setPosition(pos int) {
	e.position = pos
}

func (e *Element) setParent(parent Node) {
	e.parent = parent
}

func (e *Element) SetAttribute(attr Attribute) error {
	ix := slices.IndexFunc(e.Attrs, func(a Attribute) bool {
		return a.Namespace == attr.Namespace && a.Name == attr.Name
	})
	if ix < 0 {
		e.Attrs = append(e.Attrs, attr)
	} else {
		e.Attrs[ix] = attr
	}
	return nil
}

type Instruction struct {
	Name  string
	Attrs []Attribute

	parent   Node
	position int
}

func NewInstruction(target string) *Instruction {
	return &Instruction{
		Name: target,
	}
}

func (i *Instruction) LocalName() string {
	return i.Name
}

func (i *Instruction) QName() string {
	return i.Name
}

func (i *Instruction) Leaf() bool {
	return true
}

func (i *Instruction) Value() string {
	return ""
}

func (i *Instruction) SetAttribute(attr Attribute) error {
	ix := slices.IndexFunc(i.Attrs, func(a Attribute) bool {
		return a.Namespace == attr.Namespace && a.Name == attr.Name
	})
	if ix < 0 {
		i.Attrs = append(i.Attrs, attr)
	} else {
		i.Attrs[ix] = attr
	}
	return nil
}

func (i *Instruction) Position() int {
	return i.position
}

func (i *Instruction) Parent() Node {
	return i.parent
}

func (i *Instruction) setPosition(pos int) {
	i.position = pos
}

func (i *Instruction) setParent(parent Node) {
	i.parent = parent
}

type CharData struct {
	Content string

	parent   Node
	position int
}

func NewCharacterData(chardata string) *CharData {
	return &CharData{
		Content: chardata,
	}
}

func (c *CharData) LocalName() string {
	return ""
}

func (c *CharData) QName() string {
	return ""
}

func (c *CharData) Leaf() bool {
	return true
}

func (c *CharData) Value() string {
	return c.Content
}

func (c *CharData) Position() int {
	return c.position
}

func (c *CharData) Parent() Node {
	return c.parent
}

func (c *CharData) setPosition(pos int) {
	c.position = pos
}

func (c *CharData) setParent(parent Node) {
	c.parent = parent
}

type Text struct {
	Content string

	parent   Node
	position int
}

func NewText(text string) *Text {
	return &Text{
		Content: text,
	}
}

func (t *Text) LocalName() string {
	return ""
}

func (t *Text) QName() string {
	return ""
}

func (t *Text) Leaf() bool {
	return true
}

func (t *Text) Value() string {
	return t.Content
}

func (t *Text) Position() int {
	return t.position
}

func (t *Text) Parent() Node {
	return t.parent
}

func (t *Text) setPosition(pos int) {
	t.position = pos
}

func (t *Text) setParent(parent Node) {
	t.parent = parent
}

type Comment struct {
	Content string

	parent   Node
	position int
}

func NewComment(comment string) *Comment {
	return &Comment{
		Content: comment,
	}
}

func (c *Comment) LocalName() string {
	return ""
}

func (c *Comment) QName() string {
	return ""
}

func (c *Comment) Leaf() bool {
	return true
}

func (c *Comment) Value() string {
	return c.Content
}

func (c *Comment) Position() int {
	return c.position
}

func (c *Comment) Parent() Node {
	return c.parent
}

func (c *Comment) setPosition(pos int) {
	c.position = pos
}

func (c *Comment) setParent(parent Node) {
	c.parent = parent
}

type Document struct {
	root Node
}

func NewDocument(root Node) *Document {
	return &Document{
		root: root,
	}
}

func (d *Document) Write(w io.Writer) error {
	return NewWriter(w).Write(d)
}

func (d *Document) WriteString() (string, error) {
	var (
		buf bytes.Buffer
		err = d.Write(&buf)
	)
	return buf.String(), err
}

func (d *Document) LookupString(query string) ([]Node, error) {
	expr, err := Compile(query)
	if err != nil {
		return nil, err
	}
	return d.Lookup(expr)
}

func (d *Document) Lookup(expr Expr) ([]Node, error) {
	return expr.Eval(d.root)
}

func (d *Document) GetElementById(id string) (Node, error) {
	return nil, nil
}

func (d *Document) GetElementsByTagName(tag string) ([]Node, error) {
	return nil, nil
}

func (d *Document) Find(name string, depth int) Node {
	if el, ok := d.root.(*Element); ok {
		return el.Find(name, depth)
	}
	return nil
}

func (d *Document) FindAll(name string, depth int) []Node {
	if el, ok := d.root.(*Element); ok {
		return el.FindAll(name, depth)
	}
	return nil
}

func (d *Document) Append(node Node) {
	if el, ok := d.root.(*Element); ok {
		el.Append(node)
	}
}

func (d *Document) Insert(node Node, index int) {
	if el, ok := d.root.(*Element); ok {
		el.Insert(node, index)
	}
}
