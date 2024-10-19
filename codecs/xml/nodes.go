package xml

import (
	"bytes"
	"io"
	"slices"
)

type Node interface {
	Tag() string
	Leaf() bool
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
}

func NewElement(name, namespace string) *Element {
	return &Element{
		Name:      name,
		Namespace: namespace,
	}
}

func (e *Element) Tag() string {
	return e.Name
}

func (e *Element) Leaf() bool {
	return len(e.Nodes) == 0
}

func (e *Element) Find(name string, depth int) Node {
	return nil
}

func (e *Element) FindAll(name string, depth int) []Node {
	return nil
}

func (e *Element) Append(node Node) {
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
	e.Nodes = e.Nodes[:0]
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
}

func NewInstruction(target string) *Instruction {
	return &Instruction{
		Name: target,
	}
}

func (i *Instruction) Tag() string {
	return i.Name
}

func (i *Instruction) Leaf() bool {
	return true
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

type CharData struct {
	Content string
}

func NewCharacterData(chardata string) *CharData {
	return &CharData{
		Content: chardata,
	}
}

func (c *CharData) Tag() string {
	return "CDATA"
}

func (c *CharData) Leaf() bool {
	return true
}

type Text struct {
	Content string
}

func NewText(text string) *Text {
	return &Text{
		Content: text,
	}
}

func (t *Text) Tag() string {
	return "text"
}

func (t *Text) Leaf() bool {
	return true
}

type Comment struct {
	Content string
}

func NewComment(comment string) *Comment {
	return &Comment{
		Content: comment,
	}
}

func (c *Comment) Tag() string {
	return "comment"
}

func (c *Comment) Leaf() bool {
	return true
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
