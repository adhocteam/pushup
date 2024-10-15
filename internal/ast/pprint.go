package ast

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	indentSize = 2
	maxLineLen = 80
)

type prettyPrinter struct {
	w     io.Writer
	depth int
}

func NewPrettyPrinter(w io.Writer) *prettyPrinter {
	return &prettyPrinter{w: w}
}

func (p *prettyPrinter) print(format string, a ...interface{}) {
	fmt.Fprintf(p.w, format, a...)
}

func (p *prettyPrinter) println(format string, a ...interface{}) {
	p.print(strings.Repeat(" ", p.depth*indentSize))
	p.print(format+"\n", a...)
}

func (p *prettyPrinter) indent() {
	p.depth++
}

func (p *prettyPrinter) dedent() {
	p.depth--
	if p.depth < 0 {
		p.depth = 0
	}
}

func (p *prettyPrinter) PrettyPrint(t *SyntaxTree) {
	p.printNodes(NodeList(t.Nodes))
}

func (p *prettyPrinter) printNodes(nodes NodeList) {
	for _, node := range nodes {
		p.printNode(node)
	}
}

func (p *prettyPrinter) printNode(n Node) {
	switch node := n.(type) {
	case *NodeLiteral:
		p.printLiteral(node)
	case *NodeGoStrExpr:
		p.println("\x1b[33m{{ %s }}\x1b[0m", node.Expr)
	case *NodeGoCode:
		p.println("\x1b[34m{%% %s %%}\x1b[0m", node.Code)
	case *NodeIf:
		p.printIf(node)
	case *NodeFor:
		p.printFor(node)
	case *NodeElement:
		p.printElement(node)
	case *NodePartial:
		p.printPartial(node)
	case *NodeBlock:
		p.printNodes(NodeList(node.Nodes))
	case *NodeImport:
		p.printImport(node)
	case NodeList:
		p.printNodes(node)
	}
}

func (p *prettyPrinter) printLiteral(n *NodeLiteral) {
	if !isAllWhitespace(n.Text) {
		lines := strings.Split(n.Text, "\n")
		for _, line := range lines {
			if len(line) > maxLineLen {
				p.println("\x1b[32m%q\x1b[0m", line[:maxLineLen]+"...")
			} else {
				p.println("\x1b[32m%q\x1b[0m", line)
			}
		}
	}
}

func (p *prettyPrinter) printIf(n *NodeIf) {
	p.println("\x1b[35mIF\x1b[0m")
	p.indent()
	p.printNode(n.Cond)
	p.dedent()
	p.println("\x1b[35mTHEN\x1b[0m")
	p.indent()
	p.printNode(n.Then)
	p.dedent()
	if n.Alt != nil {
		p.println("\x1b[1;35mELSE\x1b[0m")
		p.indent()
		p.printNode(n.Alt)
		p.dedent()
	}
}

func (p *prettyPrinter) printFor(n *NodeFor) {
	p.println("\x1b[36mFOR\x1b[0m")
	p.indent()
	p.printNode(n.Clause)
	p.printNode(n.Block)
	p.dedent()
}

func (p *prettyPrinter) printElement(n *NodeElement) {
	p.println("\x1b[31m%s\x1b[0m", n.Tag.Start())
	p.indent()
	p.printNodes(NodeList(n.Children))
	p.dedent()
	p.println("\x1b[31m%s\x1b[0m", n.Tag.End())
}

func (p *prettyPrinter) printPartial(n *NodePartial) {
	p.println("PARTIAL %s", n.Name)
	p.indent()
	p.printNode(n.Block)
	p.dedent()
}

func (p *prettyPrinter) printImport(n *NodeImport) {
	if n.Decl.PkgName != "" {
		p.println("IMPORT %s %s", n.Decl.PkgName, n.Decl.Path)
	} else {
		p.println("IMPORT %s", n.Decl.Path)
	}
}

func isAllWhitespace(s string) bool {
	return strings.TrimSpace(s) == ""
}

func PrettyPrintTree(t *SyntaxTree) {
	NewPrettyPrinter(os.Stdout).PrettyPrint(t)
}
