package main

import (
	"encoding/json"
	"fmt"
)

type Span struct {
	Start int
	End   int
}

// Node represents a portion of the Pushup syntax, like a chunk of HTML,
// or a Go expression to be evaluated, or a control flow construct like `if'
// or `for'.
type Node interface {
	Pos() Span
}

// BEGIN GENERATED CODE NODE DEFINITIONS -- DO NOT EDIT
type NodeLiteral struct {
	Text string
	Span Span
}

func (n NodeLiteral) Pos() Span {
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
		Span Span
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
	Span Span
}

func (n NodeGoStrExpr) Pos() Span {
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
		Span Span
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
	Span    Span
}

func (n NodeGoCode) Pos() Span {
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
		Span    Span
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
	Then *NodeBlock
	Alt  Node
}

func (n NodeIf) Pos() Span {
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
		n.Then = wrapped.Node.(*NodeBlock)
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
	Block  *NodeBlock
}

func (n NodeFor) Pos() Span {
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
		n.Block = wrapped.Node.(*NodeBlock)
	}

	return nil
}

var _ Node = (*NodeFor)(nil)

type NodePartial struct {
	Name  string
	Span  Span
	Block *NodeBlock
}

func (n NodePartial) Pos() Span {
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
		Span  Span
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
		n.Block = wrapped.Node.(*NodeBlock)
	}

	return nil
}

var _ Node = (*NodePartial)(nil)

type NodeBlock struct {
	Nodes []Node
}

func (n NodeBlock) Pos() Span {
	return n.Nodes[0].Pos()
}

func (n NodeBlock) MarshalJSON() ([]byte, error) {
	type t NodeBlock

	return json.Marshal(struct {
		Type string
		Node t
	}{
		Type: "NodeBlock",
		Node: t{
			Nodes: n.Nodes,
		},
	})
}

func (n *NodeBlock) UnmarshalJSON(data []byte) error {
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

var _ Node = (*NodeBlock)(nil)

type NodeElement struct {
	Tag           Tag
	StartTagNodes []Node
	Children      []Node
	Span          Span
}

func (n NodeElement) Pos() Span {
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
		Tag           Tag
		StartTagNodes []json.RawMessage
		Children      []json.RawMessage
		Span          Span
	}
	var t raw

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	n.Tag = t.Tag

	for _, raw := range t.StartTagNodes {
		var wrapped NodeWrapper
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return err
		}
		n.StartTagNodes = append(n.StartTagNodes, wrapped.Node)
	}

	for _, raw := range t.Children {
		var wrapped NodeWrapper
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return err
		}
		n.Children = append(n.Children, wrapped.Node)
	}

	n.Span = t.Span

	return nil
}

var _ Node = (*NodeElement)(nil)

type NodeImport struct {
	Decl ImportDecl
	Span Span
}

func (n NodeImport) Pos() Span {
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
		Span Span
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

	case "NodeBlock":
		var node NodeBlock
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

type NodeList []Node

func (n NodeList) Pos() Span { return n[0].Pos() }

type visitor interface {
	visit(Node) visitor
}

type inspector func(Node) bool

func (f inspector) visit(n Node) visitor {
	if f(n) {
		return f
	}
	return nil
}

func inspect(n Node, f func(Node) bool) {
	walk(inspector(f), n)
}

func walkNodeList(v visitor, list []Node) {
	for _, n := range list {
		walk(v, n)
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
	case *NodeBlock:
		walkNodeList(v, n.Nodes)
	case *NodeImport:
		// no children
	case NodeList:
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

type SyntaxTree struct {
	Nodes []Node
}

func (st *SyntaxTree) UnmarshalJSON(data []byte) error {
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
		st.Nodes = append(st.Nodes, wrapped.Node)
	}

	return nil
}

func optimize(tree *SyntaxTree) *SyntaxTree {
	tree.Nodes = coalesceLiterals(tree.Nodes)
	return tree
}

// coalesceLiterals is an optimization that coalesces consecutive HTML literal
// nodes together by concatenating their strings together in a single node.
// TODO(paulsmith): further optimization could be had by descending in to child
// nodes, refactor this using inspect().
func coalesceLiterals(nodes []Node) []Node {
	if len(nodes) > 0 {
		n := 0
		for range nodes[:len(nodes)-1] {
			this, thisOk := nodes[n].(*NodeLiteral)
			next, nextOk := nodes[n+1].(*NodeLiteral)
			if thisOk && nextOk && len(this.Text) < 512 {
				this.Text += next.Text
				this.Span.End = next.Span.End
				nodes = append(nodes[:n+1], nodes[n+2:]...)
			} else {
				n++
			}
		}
		nodes = nodes[:n+1]
	}
	return nodes
}
