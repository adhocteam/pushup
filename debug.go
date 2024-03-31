package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const padding = " "

func prettyPrintTree(t *SyntaxTree) {
	depth := -1
	var w io.Writer = os.Stdout
	//nolint:errcheck
	pad := func() { w.Write([]byte(strings.Repeat(padding, depth))) }
	var f inspector
	f = func(n Node) bool {
		depth++
		defer func() {
			depth--
		}()
		pad()
		switch n := n.(type) {
		case *NodeLiteral:
			if !isAllWhitespace(n.Text) {
				str := n.Text
				if len(str) > 20 {
					str = str[:20] + "..."
				}
				fmt.Fprintf(w, "\x1b[32m%q\x1b[0m\n", str)
			}
		case *NodeGoStrExpr:
			fmt.Fprintf(w, "\x1b[33m%s\x1b[0m\n", n.Expr)
		case *NodeGoCode:
			fmt.Fprintf(w, "\x1b[34m%s\x1b[0m\n", n.Code)
		case *NodeIf:
			fmt.Fprintf(w, "\x1b[35mIF\x1b[0m")
			f(n.Cond)
			pad()
			fmt.Fprintf(w, "\x1b[35mTHEN\x1b[0m\n")
			f(n.Then)
			if n.Alt != nil {
				pad()
				fmt.Fprintf(w, "\x1b[1;35mELSE\x1b[0m\n")
				f(n.Alt)
			}
			return false
		case *NodeFor:
			fmt.Fprintf(w, "\x1b[36mFOR\x1b[0m")
			f(n.Clause)
			f(n.Block)
			return false
		case *NodeElement:
			fmt.Fprintf(w, "\x1b[31m%s\x1b[0m\n", n.Tag.start())
			f(NodeList(n.Children))
			fmt.Fprintf(w, "\x1b[31m%s\x1b[0m\n", n.Tag.end())
			return false
		case *NodePartial:
			fmt.Fprintf(w, "PARTIAL %s\n", n.Name)
			f(n.Block)
			return false
		case *NodeBlock:
			f(NodeList(n.Nodes))
			return false
		case *NodeImport:
			fmt.Fprintf(w, "IMPORT ")
			if n.Decl.PkgName != "" {
				fmt.Fprintf(w, "%s", n.Decl.PkgName)
			}
			fmt.Fprintf(w, "%s\n", n.Decl.Path)
		case NodeList:
			for _, x := range n {
				f(x)
			}
			return false
		}
		return true
	}
	inspect(NodeList(t.Nodes), f)
}
