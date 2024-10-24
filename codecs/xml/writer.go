package xml

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Writer struct {
	writer *bufio.Writer

	Compact  bool
	Indent   string
	NoProlog bool
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		writer: bufio.NewWriter(w),
		Indent: "  ",
	}
}

func (w *Writer) Write(doc *Document) error {
	if w.Compact {
		w.Indent = ""
	}
	if err := w.writeProlog(); err != nil {
		return err
	}
	w.writeNL()
	return w.writeNode(doc.root, -1)
}

func (w *Writer) writeNode(node Node, depth int) error {
	switch node := node.(type) {
	case *Element:
		return w.writeElement(node, depth+1)
	case *CharData:
		return w.writeCharData(node, depth+1)
	case *Text:
		return w.writeLiteral(node, depth+1)
	case *Instruction:
		return w.writeInstruction(node, depth+1)
	case *Comment:
		return w.writeComment(node, depth+1)
	default:
		return fmt.Errorf("node: unknown type")
	}
}

func (w *Writer) writeElement(node *Element, depth int) error {
	w.writeNL()

	prefix := strings.Repeat(w.Indent, depth)
	if prefix != "" {
		w.writer.WriteString(prefix)
	}
	w.writer.WriteRune(langle)
	if node.Namespace != "" {
		w.writer.WriteString(node.Namespace)
		w.writer.WriteRune(colon)
	}
	w.writer.WriteString(node.Name)
	level := depth + 1
	if len(node.Attrs) == 1 {
		level = 0
	}
	if err := w.writeAttributes(node.Attrs, level); err != nil {
		return err
	}
	if len(node.Nodes) == 0 {
		w.writer.WriteRune(slash)
		w.writer.WriteRune(rangle)
		return w.writer.Flush()
	}
	w.writer.WriteRune(rangle)
	for _, n := range node.Nodes {
		if err := w.writeNode(n, depth+1); err != nil {
			return err
		}
	}
	if n := len(node.Nodes); n > 0 {
		_, ok := node.Nodes[n-1].(*Text)
		if !ok {
			w.writeNL()
			w.writer.WriteString(prefix)
		}
	}
	w.writer.WriteRune(langle)
	w.writer.WriteRune(slash)
	if node.Namespace != "" {
		w.writer.WriteString(node.Namespace)
		w.writer.WriteRune(colon)
	}
	w.writer.WriteString(node.Name)
	w.writer.WriteRune(rangle)
	return w.writer.Flush()
}

func (w *Writer) writeLiteral(node *Text, _ int) error {
	_, err := w.writer.WriteString(node.Content)
	return err
}

func (w *Writer) writeCharData(node *CharData, _ int) error {
	w.writer.WriteRune(langle)
	w.writer.WriteRune(bang)
	w.writer.WriteRune(lsquare)
	w.writer.WriteString("CDATA")
	w.writer.WriteRune(lsquare)
	w.writer.WriteString(node.Content)
	w.writer.WriteRune(rsquare)
	w.writer.WriteRune(rsquare)
	w.writer.WriteRune(rangle)
	return nil
}

func (w *Writer) writeComment(node *Comment, depth int) error {
	w.writeNL()
	prefix := strings.Repeat(w.Indent, depth)
	w.writer.WriteString(prefix)
	w.writer.WriteRune(langle)
	w.writer.WriteRune(bang)
	w.writer.WriteRune(dash)
	w.writer.WriteRune(dash)
	w.writer.WriteString(node.Content)
	w.writer.WriteRune(dash)
	w.writer.WriteRune(dash)
	w.writer.WriteRune(rangle)
	return nil
}

func (w *Writer) writeInstruction(node *Instruction, depth int) error {
	if depth > 0 {
		w.writeNL()
	}
	prefix := strings.Repeat(w.Indent, depth)
	if prefix != "" {
		w.writer.WriteString(prefix)
	}
	w.writer.WriteRune(langle)
	w.writer.WriteRune(question)
	w.writer.WriteString(node.Name)
	if err := w.writeAttributes(node.Attrs, 0); err != nil {
		return err
	}
	w.writer.WriteRune(question)
	w.writer.WriteRune(rangle)
	return w.writer.Flush()
}

func (w *Writer) writeProlog() error {
	if w.NoProlog {
		return nil
	}
	prolog := Instruction{
		Name: "xml",
		Attrs: []Attribute{
			{Name: "version", Value: SupportedVersion},
			{Name: "encoding", Value: "UTF-8"},
		},
	}
	return w.writeInstruction(&prolog, 0)
}

func (w *Writer) writeAttributes(attrs []Attribute, depth int) error {
	prefix := strings.Repeat(w.Indent, depth)
	for _, a := range attrs {
		if depth == 0 || w.Compact {
			w.writer.WriteRune(' ')
		} else {
			w.writeNL()
			w.writer.WriteString(prefix)
		}
		if a.Namespace != "" {
			w.writer.WriteString(a.Namespace)
			w.writer.WriteRune(colon)
		}
		w.writer.WriteString(a.Name)
		w.writer.WriteRune(equal)
		w.writer.WriteRune(quote)
		w.writer.WriteString(a.Value)
		w.writer.WriteRune(quote)
	}
	return nil
}

func (w *Writer) writeNL() {
	if w.Compact {
		return
	}
	w.writer.WriteRune('\n')
}