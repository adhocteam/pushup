package main

import "fmt"

type span struct {
	start int
	end   int
}

// node represents a portion of the Pushup syntax, like a chunk of HTML,
// or a Go expression to be evaluated, or a control flow construct like `if'
// or `for'.
type node interface {
	Pos() span
}

type nodeList []node

func (n nodeList) Pos() span { return n[0].Pos() }

type visitor interface {
	visit(node) visitor
}

type inspector func(node) bool

func (f inspector) visit(n node) visitor {
	if f(n) {
		return f
	}
	return nil
}

func inspect(n node, f func(node) bool) {
	walk(inspector(f), n)
}

func walkNodeList(v visitor, list []node) {
	for _, n := range list {
		walk(v, n)
	}
}

func walk(v visitor, n node) {
	if v = v.visit(n); v == nil {
		return
	}

	switch n := n.(type) {
	case *nodeElement:
		walkNodeList(v, n.startTagNodes)
		walkNodeList(v, n.children)
	case *nodeLiteral:
		// no children
	case *nodeGoStrExpr:
		// no children
	case *nodeGoCode:
		// no children
	case *nodeIf:
		walk(v, n.cond)
		walk(v, n.then)
		if n.alt != nil {
			walk(v, n.alt)
		}
	case *nodeFor:
		walk(v, n.clause)
		walk(v, n.block)
	case *nodeBlock:
		walkNodeList(v, n.nodes)
	case *nodeImport:
		// no children
	case nodeList:
		walkNodeList(v, n)
	case *nodePartial:
		walk(v, n.block)
	default:
		panic(fmt.Sprintf("unhandled type %T", n))
	}
	v.visit(nil)
}

type nodeLiteral struct {
	str string
	pos span
}

func (e nodeLiteral) Pos() span { return e.pos }

var _ node = (*nodeLiteral)(nil)

type nodeGoStrExpr struct {
	expr string
	pos  span
}

func (e nodeGoStrExpr) Pos() span { return e.pos }

var _ node = (*nodeGoStrExpr)(nil)

type goCodeContext int

const (
	inlineGoCode goCodeContext = iota
	handlerGoCode
)

type nodeGoCode struct {
	context goCodeContext
	code    string
	pos     span
}

func (e nodeGoCode) Pos() span { return e.pos }

var _ node = (*nodeGoCode)(nil)

type nodeIf struct {
	cond *nodeGoStrExpr
	then *nodeBlock
	alt  node
}

func (e nodeIf) Pos() span { return e.cond.pos }

var _ node = (*nodeIf)(nil)

type nodeFor struct {
	clause *nodeGoCode
	block  *nodeBlock
}

func (e nodeFor) Pos() span { return e.clause.pos }

// nodePartial is a syntax tree node representing an inline partial in a Pushup
// page.
type nodePartial struct {
	name  string
	pos   span
	block *nodeBlock
}

func (e nodePartial) Pos() span { return e.pos }

var _ node = (*nodePartial)(nil)

// nodeBlock represents a block of nodes, i.e., a sequence of nodes that
// appear in order in the source syntax.
type nodeBlock struct {
	nodes []node
}

func (e *nodeBlock) Pos() span {
	// FIXME(paulsmith): span end all exprs
	return e.nodes[0].Pos()
}

var _ node = (*nodeBlock)(nil)

// nodeElement represents an HTML element, with a start tag, optional
// attributes, optional children, and an end tag.
type nodeElement struct {
	tag           tag
	startTagNodes []node
	children      []node
	pos           span
}

func (e nodeElement) Pos() span { return e.pos }

var _ node = (*nodeElement)(nil)

type nodeImport struct {
	decl importDecl
	pos  span
}

func (e nodeImport) Pos() span { return e.pos }

var _ node = (*nodeImport)(nil)

type syntaxTree struct {
	nodes []node
}

func optimize(tree *syntaxTree) *syntaxTree {
	tree.nodes = coalesceLiterals(tree.nodes)
	return tree
}

// coalesceLiterals is an optimization that coalesces consecutive HTML literal
// nodes together by concatenating their strings together in a single node.
// TODO(paulsmith): further optimization could be had by descending in to child
// nodes, refactor this using inspect().
func coalesceLiterals(nodes []node) []node {
	// before := len(nodes)
	if len(nodes) > 0 {
		n := 0
		for range nodes[:len(nodes)-1] {
			this, thisOk := nodes[n].(*nodeLiteral)
			next, nextOk := nodes[n+1].(*nodeLiteral)
			if thisOk && nextOk && len(this.str) < 512 {
				this.str += next.str
				this.pos.end = next.pos.end
				nodes = append(nodes[:n+1], nodes[n+2:]...)
			} else {
				n++
			}
		}
		nodes = nodes[:n+1]
	}
	return nodes
}
