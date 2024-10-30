package ast

import (
	"encoding/json"
	"fmt"
	"iter"

	"github.com/adhocteam/pushup/internal/element"
	"github.com/adhocteam/pushup/internal/source"
)

// ImportDecl represents a Go import declaration.
type ImportDecl struct {
	PkgName string
	Path    string
}

// Node represents a portion of the Pushup syntax, like a chunk of HTML,
// or a Go expression to be evaluated, or a control flow construct like `if'
// or `for'.
type Node interface {
	Pos() source.Span
}

// BEGIN GENERATED CODE NODE DEFINITIONS -- DO NOT EDIT
type NodeLiteral struct {
	Text string
	Span source.Span
}

func (n NodeLiteral) Pos() source.Span {
	return n.Span
}

func (n NodeLiteral) MarshalJSON() ([]byte, error) {
	type t NodeLiteral

	return json.Marshal(struct {
		Type string
		Node t
	}{
		Type: "NodeLiteral",
		Node: t{
			Text: n.Text,
			Span: n.Span,
		},
	})
}

func (n *NodeLiteral) UnmarshalJSON(data []byte) error {
	type raw struct {
		Text string
		Span source.Span
	}
	var t raw

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	n.Text = t.Text

	n.Span = t.Span

	return nil
}

var _ Node = (*NodeLiteral)(nil)

type NodeGoStrExpr struct {
	Expr string
	Span source.Span
}

func (n NodeGoStrExpr) Pos() source.Span {
	return n.Span
}

func (n NodeGoStrExpr) MarshalJSON() ([]byte, error) {
	type t NodeGoStrExpr

	return json.Marshal(struct {
		Type string
		Node t
	}{
		Type: "NodeGoStrExpr",
		Node: t{
			Expr: n.Expr,
			Span: n.Span,
		},
	})
}

func (n *NodeGoStrExpr) UnmarshalJSON(data []byte) error {
	type raw struct {
		Expr string
		Span source.Span
	}
	var t raw

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	n.Expr = t.Expr

	n.Span = t.Span

	return nil
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

func (n NodeGoCode) MarshalJSON() ([]byte, error) {
	type t NodeGoCode

	return json.Marshal(struct {
		Type string
		Node t
	}{
		Type: "NodeGoCode",
		Node: t{
			Context: n.Context,
			Code:    n.Code,
			Span:    n.Span,
		},
	})
}

func (n *NodeGoCode) UnmarshalJSON(data []byte) error {
	type raw struct {
		Context GoCodeContext
		Code    string
		Span    source.Span
	}
	var t raw

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	n.Context = t.Context

	n.Code = t.Code

	n.Span = t.Span

	return nil
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

func (n NodeIf) MarshalJSON() ([]byte, error) {
	type t NodeIf

	return json.Marshal(struct {
		Type string
		Node t
	}{
		Type: "NodeIf",
		Node: t{
			Cond: n.Cond,
			Then: n.Then,
			Alt:  n.Alt,
		},
	})
}

func (n *NodeIf) UnmarshalJSON(data []byte) error {
	type raw struct {
		Cond json.RawMessage
		Then json.RawMessage
		Alt  json.RawMessage
	}
	var t raw

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	{
		var wrapped NodeWrapper
		if err := json.Unmarshal(t.Cond, &wrapped); err != nil {
			return err
		}
		n.Cond = wrapped.Node.(*NodeGoStrExpr)
	}

	{
		var wrapped NodeWrapper
		if err := json.Unmarshal(t.Then, &wrapped); err != nil {
			return err
		}
		n.Then = wrapped.Node.(*NodeList)
	}

	{
		var wrapped NodeWrapper
		if err := json.Unmarshal(t.Alt, &wrapped); err != nil {
			return err
		}
		n.Alt = wrapped.Node
	}

	return nil
}

var _ Node = (*NodeIf)(nil)

type NodeFor struct {
	Clause *NodeGoCode
	Block  *NodeList
}

func (n NodeFor) Pos() source.Span {
	return n.Clause.Pos()
}

func (n NodeFor) MarshalJSON() ([]byte, error) {
	type t NodeFor

	return json.Marshal(struct {
		Type string
		Node t
	}{
		Type: "NodeFor",
		Node: t{
			Clause: n.Clause,
			Block:  n.Block,
		},
	})
}

func (n *NodeFor) UnmarshalJSON(data []byte) error {
	type raw struct {
		Clause json.RawMessage
		Block  json.RawMessage
	}
	var t raw

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	{
		var wrapped NodeWrapper
		if err := json.Unmarshal(t.Clause, &wrapped); err != nil {
			return err
		}
		n.Clause = wrapped.Node.(*NodeGoCode)
	}

	{
		var wrapped NodeWrapper
		if err := json.Unmarshal(t.Block, &wrapped); err != nil {
			return err
		}
		n.Block = wrapped.Node.(*NodeList)
	}

	return nil
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

func (n NodePartial) MarshalJSON() ([]byte, error) {
	type t NodePartial

	return json.Marshal(struct {
		Type string
		Node t
	}{
		Type: "NodePartial",
		Node: t{
			Name:  n.Name,
			Span:  n.Span,
			Block: n.Block,
		},
	})
}

func (n *NodePartial) UnmarshalJSON(data []byte) error {
	type raw struct {
		Name  string
		Span  source.Span
		Block json.RawMessage
	}
	var t raw

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	n.Name = t.Name

	n.Span = t.Span

	{
		var wrapped NodeWrapper
		if err := json.Unmarshal(t.Block, &wrapped); err != nil {
			return err
		}
		n.Block = wrapped.Node.(*NodeList)
	}

	return nil
}

var _ Node = (*NodePartial)(nil)

type NodeList struct {
	Nodes []Node
}

func (n NodeList) Pos() source.Span {
	return n.Nodes[0].Pos()
}

func (n NodeList) MarshalJSON() ([]byte, error) {
	type t NodeList

	return json.Marshal(struct {
		Type string
		Node t
	}{
		Type: "NodeList",
		Node: t{
			Nodes: n.Nodes,
		},
	})
}

func (n *NodeList) UnmarshalJSON(data []byte) error {
	type raw struct {
		Nodes []json.RawMessage
	}
	var t raw

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	for _, raw := range t.Nodes {
		var wrapped NodeWrapper
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return err
		}
		n.Nodes = append(n.Nodes, wrapped.Node)
	}

	return nil
}

var _ Node = (*NodeList)(nil)

type NodeElement struct {
	Tag           element.Tag
	StartTagNodes *NodeList
	Children      *NodeList
	Span          source.Span
}

func (n NodeElement) Pos() source.Span {
	return n.Span
}

func (n NodeElement) MarshalJSON() ([]byte, error) {
	type t NodeElement

	return json.Marshal(struct {
		Type string
		Node t
	}{
		Type: "NodeElement",
		Node: t{
			Tag:           n.Tag,
			StartTagNodes: n.StartTagNodes,
			Children:      n.Children,
			Span:          n.Span,
		},
	})
}

func (n *NodeElement) UnmarshalJSON(data []byte) error {
	type raw struct {
		Tag           element.Tag
		StartTagNodes json.RawMessage
		Children      json.RawMessage
		Span          source.Span
	}
	var t raw

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	n.Tag = t.Tag

	{
		var wrapped NodeWrapper
		if err := json.Unmarshal(t.StartTagNodes, &wrapped); err != nil {
			return err
		}
		n.StartTagNodes = wrapped.Node.(*NodeList)
	}

	{
		var wrapped NodeWrapper
		if err := json.Unmarshal(t.Children, &wrapped); err != nil {
			return err
		}
		n.Children = wrapped.Node.(*NodeList)
	}

	n.Span = t.Span

	return nil
}

var _ Node = (*NodeElement)(nil)

type NodeImport struct {
	Decl ImportDecl
	Span source.Span
}

func (n NodeImport) Pos() source.Span {
	return n.Span
}

func (n NodeImport) MarshalJSON() ([]byte, error) {
	type t NodeImport

	return json.Marshal(struct {
		Type string
		Node t
	}{
		Type: "NodeImport",
		Node: t{
			Decl: n.Decl,
			Span: n.Span,
		},
	})
}

func (n *NodeImport) UnmarshalJSON(data []byte) error {
	type raw struct {
		Decl ImportDecl
		Span source.Span
	}
	var t raw

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	n.Decl = t.Decl

	n.Span = t.Span

	return nil
}

var _ Node = (*NodeImport)(nil)

func (nw *NodeWrapper) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var typeMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &typeMap); err != nil {
		return err
	}

	var typ string
	if err := json.Unmarshal(typeMap["Type"], &typ); err != nil {
		return err
	}

	var err error
	switch typ {

	case "NodeLiteral":
		var node NodeLiteral
		err = json.Unmarshal(typeMap["Node"], &node)
		nw.Node = &node

	case "NodeGoStrExpr":
		var node NodeGoStrExpr
		err = json.Unmarshal(typeMap["Node"], &node)
		nw.Node = &node

	case "NodeGoCode":
		var node NodeGoCode
		err = json.Unmarshal(typeMap["Node"], &node)
		nw.Node = &node

	case "NodeIf":
		var node NodeIf
		err = json.Unmarshal(typeMap["Node"], &node)
		nw.Node = &node

	case "NodeFor":
		var node NodeFor
		err = json.Unmarshal(typeMap["Node"], &node)
		nw.Node = &node

	case "NodePartial":
		var node NodePartial
		err = json.Unmarshal(typeMap["Node"], &node)
		nw.Node = &node

	case "NodeList":
		var node NodeList
		err = json.Unmarshal(typeMap["Node"], &node)
		nw.Node = &node

	case "NodeElement":
		var node NodeElement
		err = json.Unmarshal(typeMap["Node"], &node)
		nw.Node = &node

	case "NodeImport":
		var node NodeImport
		err = json.Unmarshal(typeMap["Node"], &node)
		nw.Node = &node

	default:
		return fmt.Errorf("unknown node type: %q", typ)
	}

	return err
}

// END GENERATED CODE NODE DEFINITIONS -- DO NOT EDIT

// Extra NodeList methods

func NewNodeList(nodes ...Node) *NodeList {
	nl := &NodeList{}
	nl.Nodes = append(nl.Nodes, nodes...)
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

func (doc *Document) UnmarshalJSON(data []byte) error {
	var t struct {
		Nodes NodeWrapper
	}

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	doc.Nodes = t.Nodes.Node.(*NodeList)

	return nil
}
