package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const padding = " "

func prettyPrintTree(t *syntaxTree) {
	depth := -1
	var w io.Writer = os.Stdout
	//nolint:errcheck
	pad := func() { w.Write([]byte(strings.Repeat(padding, depth))) }
	var f inspector
	f = func(n node) bool {
		depth++
		defer func() {
			depth--
		}()
		pad()
		switch n := n.(type) {
		case *nodeLiteral:
			if !isAllWhitespace(n.str) {
				str := n.str
				if len(str) > 20 {
					str = str[:20] + "..."
				}
				fmt.Fprintf(w, "\x1b[32m%q\x1b[0m\n", str)
			}
		case *nodeGoStrExpr:
			fmt.Fprintf(w, "\x1b[33m%s\x1b[0m\n", n.expr)
		case *nodeGoCode:
			fmt.Fprintf(w, "\x1b[34m%s\x1b[0m\n", n.code)
		case *nodeIf:
			fmt.Fprintf(w, "\x1b[35mIF\x1b[0m")
			f(n.cond)
			pad()
			fmt.Fprintf(w, "\x1b[35mTHEN\x1b[0m\n")
			f(n.then)
			if n.alt != nil {
				pad()
				fmt.Fprintf(w, "\x1b[1;35mELSE\x1b[0m\n")
				f(n.alt)
			}
			return false
		case *nodeFor:
			fmt.Fprintf(w, "\x1b[36mFOR\x1b[0m")
			f(n.clause)
			f(n.block)
			return false
		case *nodeElement:
			fmt.Fprintf(w, "\x1b[31m%s\x1b[0m\n", n.tag.start())
			f(nodeList(n.children))
			fmt.Fprintf(w, "\x1b[31m%s\x1b[0m\n", n.tag.end())
			return false
		case *nodeSection:
			fmt.Fprintf(w, "SECTION %s\n", n.name)
			f(n.block)
			return false
		case *nodePartial:
			fmt.Fprintf(w, "PARTIAL %s\n", n.name)
			f(n.block)
			return false
		case *nodeBlock:
			f(nodeList(n.nodes))
			return false
		case *nodeImport:
			fmt.Fprintf(w, "IMPORT ")
			if n.decl.pkgName != "" {
				fmt.Fprintf(w, "%s", n.decl.pkgName)
			}
			fmt.Fprintf(w, "%s\n", n.decl.path)
		case nodeList:
			for _, x := range n {
				f(x)
			}
			return false
		}
		return true
	}
	inspect(nodeList(t.nodes), f)
}
