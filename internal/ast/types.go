package ast

import (
	"encoding/json"
	"fmt"

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
	Then *NodeBlock
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
		n.Block = wrapped.Node.(*NodeBlock)
	}

	return nil
}

var _ Node = (*NodeFor)(nil)

type NodePartial struct {
	Name  string
	Span  source.Span
	Block *NodeBlock
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
		n.Block = wrapped.Node.(*NodeBlock)
	}

	return nil
}

var _ Node = (*NodePartial)(nil)

type NodeBlock struct {
	Nodes []Node
}

func (n NodeBlock) Pos() source.Span {
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
	Tag           element.Tag
	StartTagNodes []Node
	Children      []Node
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
		StartTagNodes []json.RawMessage
		Children      []json.RawMessage
		Span          source.Span
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

func (n NodeList) Pos() source.Span { return n[0].Pos() }

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

type Document struct {
	Nodes []Node
}

func (st *Document) UnmarshalJSON(data []byte) error {
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
