package up

import (
	"strings"

	"github.com/adhocteam/pushup/internal/ast"
)

type RouteType int

const (
	PageRoute RouteType = iota
	PartialRoute
)

type CompileUnit struct {
	Project *Project
	File    *File

	// Derived info
	TypeName string // eg., "AboutPage"
	Package  string // eg., "myapp/pages"
	Route    string // eg., "/about"

	// AST -> analyzer
	Imports            []ast.ImportDecl
	Nodes              *ast.NodeList
	ComponentCallSites map[*ast.NodeElement]*ast.Tag

	// Partials is a map of all top-level inline partials in this page (if
	// File.Kind is a Page).
	Partials map[*ast.NodePartial]*Partial
}

func NewCompileUnit(project *Project, file *File) *CompileUnit {
	unit := &CompileUnit{
		Project:            project,
		File:               file,
		ComponentCallSites: make(map[*ast.NodeElement]*ast.Tag),
		Partials:           make(map[*ast.NodePartial]*Partial),
	}
	return unit
}

// Partial represents an inline partial in a Pushup page.
type Partial struct {
	Node     ast.Node
	Name     string // eg., "list"
	Parent   *Partial
	Children []*Partial

	// Derived info
	Route string // a net/http ServeMux pattern, eg., "/about/list", "/{$}"
}

// URLPath produces the URL path segment for the partial. this takes in to
// account its ancestor partials, so nested partials have the full path from
// their containing inline partials. note that the returned string is not
// prefixed with the host page's URL path.
func (p *Partial) URLPath() string {
	segments := []string{p.Name}
	for parent := p.Parent; parent != nil; parent = parent.Parent {
		segments = append([]string{parent.Name}, segments...)
	}
	return strings.Join(segments, "/")
}
