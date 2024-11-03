package ast

import (
	"encoding/gob"
	"fmt"
	"iter"

	"github.com/adhocteam/pushup/internal/source"
)

// ImportDecl represents a Go import declaration.
type ImportDecl struct {
	PkgName string
	Path    string
}

// VarDecl represents a Go variable declaration
type VarDecl struct {
	Ident string // i.e, the variable's name
	Expr  string // i.e., the variable's type
}

// Node represents a portion of the Pushup syntax, like a chunk of HTML,
// or a Go expression to be evaluated, or a control flow construct like `if'
// or `for'.
type Node interface {
	Pos() source.Span
}

// BEGIN GENERATED CODE NODE DEFINITIONS -- DO NOT EDIT
// see types_test.go

type NodeLiteral struct {
	Text string
	Span source.Span
}

func (n NodeLiteral) Pos() source.Span {
	return n.Span
}

var _ Node = (*NodeLiteral)(nil)

type NodeGoStrExpr struct {
	Expr string
	Span source.Span
}

func (n NodeGoStrExpr) Pos() source.Span {
	return n.Span
}

var _ Node = (*NodeGoStrExpr)(nil)

type NodeGoCode struct {
	Context GoCodeContext
	Code    string
	Span    source.Span
}

func (n NodeGoCode) Pos() source.Span {
	return n.Span
}

var _ Node = (*NodeGoCode)(nil)

type NodeIf struct {
	Cond *NodeGoStrExpr
	Then *NodeList
	Alt  Node
}

func (n NodeIf) Pos() source.Span {
	return n.Cond.Pos()
}

var _ Node = (*NodeIf)(nil)

type NodeFor struct {
	Clause *NodeGoCode
	Block  *NodeList
}

func (n NodeFor) Pos() source.Span {
	return n.Clause.Pos()
}

var _ Node = (*NodeFor)(nil)

type NodePartial struct {
	Name  string
	Span  source.Span
	Block *NodeList
}

func (n NodePartial) Pos() source.Span {
	return n.Span
}

var _ Node = (*NodePartial)(nil)

type NodeList struct {
	Nodes []Node
}

func (n NodeList) Pos() source.Span {
	return n.Nodes[0].Pos()
}

var _ Node = (*NodeList)(nil)

type NodeElement struct {
	Tag           Tag
	StartTagNodes *NodeList
	Children      *NodeList
	Span          source.Span
	IsSelfClosing bool
}

func (n NodeElement) Pos() source.Span {
	return n.Span
}

var _ Node = (*NodeElement)(nil)

type NodeImport struct {
	Decl ImportDecl
	Span source.Span
}

func (n NodeImport) Pos() source.Span {
	return n.Span
}

var _ Node = (*NodeImport)(nil)

type NodeParam struct {
	Decl VarDecl
	Use  bool
	Span source.Span
}

func (n NodeParam) Pos() source.Span {
	return n.Span
}

var _ Node = (*NodeParam)(nil)

func init() {
	gob.Register(&Document{})
	gob.Register(&NodeLiteral{})
	gob.Register(&NodeGoStrExpr{})
	gob.Register(&NodeGoCode{})
	gob.Register(&NodeIf{})
	gob.Register(&NodeFor{})
	gob.Register(&NodePartial{})
	gob.Register(&NodeList{})
	gob.Register(&NodeElement{})
	gob.Register(&NodeImport{})
	gob.Register(&NodeParam{})
}

// END GENERATED CODE NODE DEFINITIONS -- DO NOT EDIT

// Extra NodeList methods

func NewNodeList(nodes ...Node) *NodeList {
	nl := &NodeList{
		Nodes: make([]Node, len(nodes)),
	}
	copy(nl.Nodes, nodes)
	return nl
}

func (nl *NodeList) All() iter.Seq[Node] {
	return func(yield func(Node) bool) {
		for _, node := range nl.Nodes {
			if !yield(node) {
				return
			}
		}
	}
}

func (nl *NodeList) Append(nodes ...Node) {
	nl.Nodes = append(nl.Nodes, nodes...)
}

func (nl *NodeList) AppendFromList(other *NodeList) {
	nl.Nodes = append(nl.Nodes, other.Nodes...)
}

func (nl *NodeList) SetAt(node Node, i int) {
	nl.Nodes[i] = node
}

func (nl *NodeList) Slice(start, end int) *NodeList {
	return &NodeList{Nodes: nl.Nodes[start:end]}
}

func (nl *NodeList) Len() int {
	return len(nl.Nodes)
}

type visitor interface {
	visit(Node) visitor
}

type Inspector func(Node) bool

func (f Inspector) visit(n Node) visitor {
	if f(n) {
		return f
	}
	return nil
}

func Inspect(n Node, f func(Node) bool) {
	walk(Inspector(f), n)
}

func walkNodeList(v visitor, list *NodeList) {
	for node := range list.All() {
		walk(v, node)
	}
}

func walk(v visitor, n Node) {
	if v = v.visit(n); v == nil {
		return
	}

	switch n := n.(type) {
	case *NodeElement:
		walkNodeList(v, n.StartTagNodes)
		walkNodeList(v, n.Children)
	case *NodeLiteral:
		// no children
	case *NodeGoStrExpr:
		// no children
	case *NodeGoCode:
		// no children
	case *NodeIf:
		walk(v, n.Cond)
		walk(v, n.Then)
		if n.Alt != nil {
			walk(v, n.Alt)
		}
	case *NodeFor:
		walk(v, n.Clause)
		walk(v, n.Block)
	case *NodeImport:
		// no children
	case *NodeList:
		walkNodeList(v, n)
	case *NodePartial:
		walk(v, n.Block)
	case *NodeParam:
		// no children
	default:
		panic(fmt.Sprintf("unhandled type %T", n))
	}
	v.visit(nil)
}

type GoCodeContext int

const (
	InlineGoCode GoCodeContext = iota
	HandlerGoCode
)

type NodeWrapper struct {
	Type string
	Node Node
}

// Document represents a complete Pushup page or component.
type Document struct {
	Nodes *NodeList
}

func NewDocument() *Document {
	return &Document{Nodes: NewNodeList()}
}

func (d *Document) Pos() source.Span {
	return d.Nodes.Slice(0, 1).Pos()
}

var _ Node = (*Document)(nil)
