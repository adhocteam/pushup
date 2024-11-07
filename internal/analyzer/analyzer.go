package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/adhocteam/pushup/internal/ast"
	"github.com/adhocteam/pushup/internal/up"
)

type Analyzer interface {
	Analyze(doc *ast.Document, unit *up.CompileUnit) error
}

type analyzeFunc func(*ast.Document, *up.CompileUnit) error

func (f analyzeFunc) Analyze(doc *ast.Document, unit *up.CompileUnit) error {
	return f(doc, unit)
}

func analyze(doc *ast.Document, unit *up.CompileUnit) error {
	unit.TypeName = deriveTypeName(unit.File)
	if unit.File.Kind == up.Page {
		unit.Route = routeForPage(unit.File.RelPath)
	}

	n := 0
	var err error

	var checkComponentCallSite ast.Inspector
	checkComponentCallSite = func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.NodeElement:
			if strings.Contains(node.Tag.Name, ".") {
				nodes := node.StartTagNodes.Nodes
				tagName := nodes[0].(*ast.NodeLiteral).Text
				node.Tag.Name, err = extractComponentName(tagName)
				unit.ComponentCallSites[node] = &node.Tag
			}
		}
		return true
	}
	ast.Inspect(doc.Nodes, checkComponentCallSite)
	if err != nil {
		return fmt.Errorf("checking component call sites: %w", err)
	}

	// This pass over the syntax tree nodes enforces invariants (only one
	// handler may be declared per page) and aggregates imports
	// for easier access in the subsequent code generation phase. as a
	// result, some nodes are removed from the tree.
	var f ast.Inspector
	f = func(e ast.Node) bool {
		switch e := e.(type) {
		case *ast.NodeImport:
			unit.Imports = append(unit.Imports, e.Decl)
		case *ast.NodeGoCode:
			if e.Context == ast.HandlerGoCode {
				if unit.Handler != nil {
					err = fmt.Errorf("only one handler per page can be defined")
					return false
				}
				unit.Handler = e
			} else {
				doc.Nodes.SetAt(e, n)
				n++
			}
		case *ast.NodeElement, *ast.NodeLiteral, *ast.NodePartial, *ast.NodeFor, *ast.NodeIf:
			doc.Nodes.SetAt(e, n)
			n++
		case *ast.NodeList:
			for node := range e.All() {
				f(node)
			}
		case *ast.NodeParam:
			if unit.File.Kind == up.Page {
				err = fmt.Errorf("^param declarations are not permitted in Pushup pages")
				return false
			}
		default:
			panic(fmt.Sprintf("unhandled node type: %T", e))
		}
		// don't recurse into child nodes
		return false
	}

	ast.Inspect(doc.Nodes, f)
	if err != nil {
		return fmt.Errorf("inspecting AST: %w", err)
	}

	unit.Nodes = doc.Nodes.Slice(0, n)

	err = nil

	var checkChildrenSlot ast.Inspector
	var childrenSlotSeen bool
	checkChildrenSlot = func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.NodeElement:
			if unit.File.Kind == up.Component && node.Tag.Name == "children" {
				if !childrenSlotSeen {
					// is a slot for transclusion from callers, it may not have
					// children of its own
					if node.Children.Len() > 0 {
						err = fmt.Errorf("<children/> may not have any child nodes of itself")
						return false
					}
					childrenSlotSeen = true
				} else {
					err = fmt.Errorf("components may only use <children/> slot once")
					return false
				}
			}
		}
		return true
	}
	ast.Inspect(doc.Nodes, checkChildrenSlot)
	if err != nil {
		return fmt.Errorf("checking component <children/> use: %w", err)
	}

	// This pass is for inline partials. It needs to be separate because the
	// traversal of the tree is slightly different than the pass above.
	{
		var currentPartial *up.Partial
		var err error

		var f ast.Inspector
		f = func(e ast.Node) bool {
			switch e := e.(type) {
			case *ast.NodeLiteral:
			case *ast.NodeElement:
				f(e.StartTagNodes)
				f(e.Children)
				return false
			case *ast.NodeGoStrExpr:
			case *ast.NodeGoCode:
			case *ast.NodeIf:
				f(e.Then)
				if e.Alt != nil {
					f(e.Alt)
				}
				return false
			case *ast.NodeList:
				for node := range e.All() {
					f(node)
				}
				return false
			case *ast.NodeFor:
				f(e.Block)
				return false
			case *ast.NodePartial:
				if unit.File.Kind == up.Component {
					err = fmt.Errorf("partials are not allowed in components, only pages")
					return false
				}
				p := &up.Partial{
					Node:   e,
					Name:   e.Name,
					Parent: currentPartial,
				}
				p.TypeName = derivePartialTypeName(unit.File, p)
				p.Route = routeForPartial(unit.Route, p)
				if currentPartial != nil {
					currentPartial.Children = append(currentPartial.Children, p)
				}
				prevPartial := currentPartial
				currentPartial = p
				f(e.Block)
				currentPartial = prevPartial
				unit.Partials = append(unit.Partials, p)
				return false
			case *ast.NodeImport:
				// nothing to do
			case *ast.NodeParam:
				// nothing to do
			default:
				panic(fmt.Sprintf("unhandled node type: %T", e))
			}
			return false
		}

		ast.Inspect(unit.Nodes, f)
		if err != nil {
			return fmt.Errorf("inspecting doc in partials pass: %w", err)
		}
	}

	return nil
}

var Analyze = analyzeFunc(analyze)

// deriveTypeName returns the name of the type of the Go struct that
// holds the generated code for the Pushup page and related methods.
func deriveTypeName(file *up.File) string {
	filename := filepath.Base(file.Path)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	typename := cleanTitleCase(filename)
	if file.Kind == up.Page {
		typename += "Page"
	}
	return typename
}

func derivePartialTypeName(file *up.File, partial *up.Partial) string {
	filename := filepath.Base(file.Path)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	return cleanTitleCase(filename) + "Page" + cleanTitleCase(partial.URLPath()) + "Partial"
}

// routeForPage produces the URL path route from the name of the Pushup page.
// path is the path to the Pushup page file.
func routeForPage(path string) string {
	path, err := filepath.Rel("pages", path)
	if err != nil {
		panic(fmt.Sprintf("path to page is not relative to '%s' directory", "pages"))
	}

	var dirs []string
	dir := filepath.Dir(path)
	if dir != "." {
		dirs = strings.Split(dir, string([]rune{os.PathSeparator}))
	}
	file := filepath.Base(path)
	name := strings.TrimSuffix(file, filepath.Ext(file))
	var route string
	if name != "index" {
		dirs = append(dirs, name)
	}
	for i := range dirs {
		if strings.HasSuffix(dirs[i], "__param") {
			dirs[i] = ":" + strings.TrimSuffix(dirs[i], "__param")
		}
	}
	route = "/" + strings.Join(dirs, "/")
	if name == "index" && route[len(route)-1] != '/' {
		// indexes always have a trailing slash
		route += "/"
	}
	return route
}

func routeForPartial(pagePath string, partial *up.Partial) string {
	if pagePath == "" {
		pagePath = "/"
	}

	if !strings.HasSuffix(pagePath, "/") {
		pagePath += "/"
	}

	return pagePath + partial.URLPath()
}

func cleanTitleCase(s string) string {
	buf := make([]rune, len(s))
	i := 0
	wordBoundary := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if wordBoundary {
				wordBoundary = false
				buf[i] = unicode.ToUpper(r)
			} else {
				buf[i] = r
			}
			i++
		} else {
			wordBoundary = true
		}
	}
	return string(buf[:i])
}

func extractComponentName(tagName string) (string, error) {
	if tagName[0] != '<' {
		return "", fmt.Errorf("expected leading '<'")
	}
	tagName = tagName[1:]
	dot := strings.IndexRune(tagName, '.')
	if dot < 1 || dot == -1 {
		return "", fmt.Errorf("expected dotted name")
	}
	if space := strings.IndexRune(tagName, ' '); space != -1 {
		return tagName[:space], nil
	}
	firstNonWord := strings.IndexFunc(tagName[dot+1:], func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	return tagName[:dot+1+firstNonWord], nil
}
