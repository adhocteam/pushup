package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

const (
	pushupModulePath = "github.com/adhocteam/pushup"
	pushupApi        = pushupModulePath + "/api"
)

// ImportDecl represents a Go import declaration.
type ImportDecl struct {
	PkgName string
	Path    string
}

func lineCount(s string) int {
	return strings.Count(s, "\n") + 1
}

const methodReceiverName = "up"

// parsedPage represents a Pushup parsedPage that has been parsed and is ready
// for code generation.
type parsedPage struct {
	imports []ImportDecl
	handler *NodeGoCode
	nodes   []Node

	// partials is a list of all top-level inline partials in this page.
	partials []*partial
}

// partial represents an inline partial in a Pushup page.
type partial struct {
	node     Node
	name     string
	parent   *partial
	children []*partial
}

// urlpath produces the URL path segment for the partial. this takes in to
// account its ancestor partials, so nested partials have the full path from
// their containing inline partials. note that the returned string is not
// prefixed with the host page's URL path.
func (p *partial) urlpath() string {
	segments := []string{p.name}
	for parent := p.parent; parent != nil; parent = parent.parent {
		segments = append([]string{parent.name}, segments...)
	}
	return strings.Join(segments, "/")
}

// newPageFromTree produces a page which is the main prepared object for code
// generation. this requires walking the syntax tree and reorganizing things
// somewhat to make them easier to access. some node types are encountered
// sequentially in the source file, but need to be reorganized for access in
// the code generator.
func newPageFromTree(tree *SyntaxTree) (*parsedPage, error) {
	page := new(parsedPage)

	n := 0
	var err error

	// this pass over the syntax tree nodes enforces invariants (only one
	// handler may be declared per page) and aggregates imports
	// for easier access in the subsequent code generation phase. as a
	// result, some nodes are removed from the tree.
	var f inspector
	f = func(e Node) bool {
		switch e := e.(type) {
		case *NodeImport:
			page.imports = append(page.imports, e.Decl)
		case *NodeGoCode:
			if e.Context == HandlerGoCode {
				if page.handler != nil {
					err = fmt.Errorf("only one handler per page can be defined")
					return false
				}
				page.handler = e
			} else {
				tree.Nodes[n] = e
				n++
			}
		case NodeList:
			for _, x := range e {
				f(x)
			}
		default:
			tree.Nodes[n] = e
			n++
		}
		// don't recurse into child nodes
		return false
	}
	inspect(NodeList(tree.Nodes), f)
	if err != nil {
		return nil, err
	}

	page.nodes = tree.Nodes[:n]

	// this pass is for inline partials. it needs to be separate because the
	// traversal of the tree is slightly different than the pass above.
	{
		var currentPartial *partial
		var f inspector
		f = func(e Node) bool {
			switch e := e.(type) {
			case *NodeLiteral:
			case *NodeElement:
				f(NodeList(e.StartTagNodes))
				f(NodeList(e.Children))
				return false
			case *NodeGoStrExpr:
			case *NodeGoCode:
			case *NodeIf:
				f(e.Then)
				if e.Alt != nil {
					f(e.Alt)
				}
				return false
			case NodeList:
				for _, x := range e {
					f(x)
				}
				return false
			case *NodeFor:
				f(e.Block)
				return false
			case *NodeBlock:
				f(NodeList(e.Nodes))
				return false
			case *NodePartial:
				p := &partial{node: e, name: e.Name, parent: currentPartial}
				if currentPartial != nil {
					currentPartial.children = append(currentPartial.children, p)
				}
				prevPartial := currentPartial
				currentPartial = p
				f(e.Block)
				currentPartial = prevPartial
				page.partials = append(page.partials, p)
				return false
			case *NodeImport:
				// nothing to do
			}
			return false
		}
		inspect(NodeList(page.nodes), f)
	}

	return page, nil
}

type pageCodeGen struct {
	page    *parsedPage
	pfile   projectFile
	modPath string
	pkgName string
	source  string
	imports map[ImportDecl]bool

	// buffer for the comments at the very top of a Go source file.
	comments bytes.Buffer

	// buffer for the import declarations at the top of a Go source file.
	importDecls bytes.Buffer

	// buffer for the main body of a Go source file, i.e., the top-level
	// declarations.
	body bytes.Buffer

	ioWriterVar           string
	lineDirectivesEnabled bool
}

func newPageCodeGen(page *parsedPage, source string, cparams *compileParams) *pageCodeGen {
	g := &pageCodeGen{
		page:                  page,
		pfile:                 cparams.pfile,
		modPath:               cparams.modPath,
		pkgName:               cparams.pkgName,
		source:                source,
		imports:               make(map[ImportDecl]bool),
		ioWriterVar:           "w",
		lineDirectivesEnabled: true,
	}
	for _, im := range page.imports {
		g.imports[im] = true
	}
	return g
}

func (g *pageCodeGen) used(path ...string) {
	for _, p := range path {
		g.imports[ImportDecl{Path: strconv.Quote(p), PkgName: ""}] = true
	}
}

func (g *pageCodeGen) nodeLineNo(e Node) {
	if g.lineDirectivesEnabled {
		g.emitLineDirective(g.lineNo(e.Pos()))
	}
}

func (c *pageCodeGen) lineNo(s Span) int {
	return lineCount(c.source[:s.Start+1])
}

func (g *pageCodeGen) emitLineDirective(n int) {
	g.bodyPrintf("//line %s:%d\n", g.pfile.relpath(), n)
}

func (g *pageCodeGen) commentPrintf(format string, args ...any) {
	fmt.Fprintf(&g.comments, format, args...)
}

func (g *pageCodeGen) importDeclPrintf(format string, args ...any) {
	fmt.Fprintf(&g.importDecls, format, args...)
}

func (g *pageCodeGen) bodyPrintf(format string, args ...any) {
	fmt.Fprintf(&g.body, format, args...)
}

func (g *pageCodeGen) readAll() ([]byte, error) {
	bufs := []io.Reader{
		&g.comments,
		strings.NewReader("package " + g.pkgName + "\n"),
		&g.importDecls,
		&g.body,
	}
	raw, err := io.ReadAll(io.MultiReader(bufs...))
	return raw, err
}

func (g *pageCodeGen) generate() {
	nodes := g.page.nodes
	g.genNode(NodeList(nodes))
}

func (g *pageCodeGen) genElement(e *NodeElement, f inspector) {
	g.used("io")
	g.nodeLineNo(e)
	f(NodeList(e.StartTagNodes))
	f(NodeList(e.Children))
	g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(e.Tag.end()))
}

func (g *pageCodeGen) genNode(n Node) {
	var f inspector
	f = func(e Node) bool {
		switch e := e.(type) {
		case *NodeLiteral:
			g.used("io")
			g.nodeLineNo(e)
			g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(e.Text))
		case *NodeElement:
			g.genElement(e, f)
			return false
		case *NodeGoStrExpr:
			g.nodeLineNo(e)
			g.used(pushupApi)
			g.bodyPrintf("api.PrintEscaped(%s, %s)\n", g.ioWriterVar, e.Expr)
		case *NodeGoCode:
			if e.Context != InlineGoCode {
				panic("internal error: expected inlineGoCode")
			}
			srcLineNo := g.lineNo(e.Pos())
			lines := strings.Split(e.Code, "\n")
			for _, line := range lines {
				if g.lineDirectivesEnabled {
					g.emitLineDirective(srcLineNo)
				}
				g.bodyPrintf("%s\n", line)
				srcLineNo++
			}
		case *NodeIf:
			g.bodyPrintf("if %s {\n", e.Cond.Expr)
			f(e.Then)
			if e.Alt == nil {
				g.bodyPrintf("}\n")
			} else {
				g.bodyPrintf("} else {\n")
				f(e.Alt)
				g.bodyPrintf("}\n")
			}
			return false
		case NodeList:
			for _, x := range e {
				f(x)
			}
			return false
		case *NodeFor:
			g.bodyPrintf("for %s {\n", e.Clause.Code)
			f(e.Block)
			g.bodyPrintf("}\n")
			return false
		case *NodeBlock:
			f(NodeList(e.Nodes))
			return false
		case *NodePartial:
			f(e.Block)
			return false
		case *NodeImport:
			// nothing to do
		}
		return true
	}
	inspect(n, f)
}

// NOTE(paulsmith): per DOM spec, "In tree order is preorder, depth-first traversal of a tree."

func (g *pageCodeGen) genNodePartial(n Node, p *partial) {
	var f inspector
	var state int
	const (
		stateStart int = iota
		stateInPartialScope
	)
	state = stateStart
	f = func(n Node) bool {
		if n != nil {
			switch n := n.(type) {
			case *NodeLiteral:
				if state == stateInPartialScope {
					g.used("io")
					g.nodeLineNo(n)
					g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(n.Text))
				}
			case *NodeElement:
				if state == stateInPartialScope {
					g.used("io")
					g.nodeLineNo(n)
					f(NodeList(n.StartTagNodes))
				}
				f(NodeList(n.Children))
				if state == stateInPartialScope {
					g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(n.Tag.end()))
				}
				return false
			case *NodePartial:
				if n == p.node {
					state = stateInPartialScope
				}
				f(n.Block)
				state = stateStart
				return false
			case *NodeGoStrExpr:
				if state == stateInPartialScope {
					g.nodeLineNo(n)
					g.used(pushupApi)
					g.bodyPrintf("api.PrintEscaped(%s, %s)\n", g.ioWriterVar, n.Expr)
				}
			case *NodeFor:
				if state == stateInPartialScope {
					g.bodyPrintf("for %s {\n", n.Clause.Code)
					f(n.Block)
					g.bodyPrintf("}\n")
				}
				return false
			case *NodeIf:
				g.bodyPrintf("if %s {\n", n.Cond.Expr)
				f(n.Then)
				if n.Alt == nil {
					g.bodyPrintf("}\n")
				} else {
					g.bodyPrintf("} else {\n")
					f(n.Alt)
					g.bodyPrintf("}\n")
				}
				return false
			case *NodeGoCode:
				if n.Context != InlineGoCode {
					panic("internal error: expected inlineGoCode")
				}
				srcLineNo := g.lineNo(n.Pos())
				lines := strings.Split(n.Code, "\n")
				for _, line := range lines {
					if g.lineDirectivesEnabled {
						g.emitLineDirective(srcLineNo)
					}
					g.bodyPrintf("%s\n", line)
					srcLineNo++
				}
			case NodeList:
				for _, x := range n {
					f(x)
				}
			case *NodeBlock:
				for _, x := range n.Nodes {
					f(x)
				}
			default:
				panic(fmt.Sprintf("internal error: unhandled node type %T", n))
			}
		}
		return false
	}
	inspect(n, f)
}

// pkgPathForPage produces the Go package path from the filesystem path of the
// Pushup page.
func pkgPathForPage(modPath string, path string) string {
	return filepath.Join(modPath, filepath.Dir(path))
}

// TODO(paulsmith): allow "pages" to be a configurable path prefix
const pagesPathPrefix = "pages"

// routeForPage produces the URL path route from the name of the Pushup page.
// path is the path to the Pushup page file.
func routeForPage(path string) string {
	path, err := filepath.Rel(pagesPathPrefix, path)
	if err != nil {
		panic(fmt.Sprintf("path to page is not relative to '%s' directory", pagesPathPrefix))
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

func routeForPartial(relpath string, partialUrlpath string) string {
	prefix := strings.TrimSuffix(relpath, filepath.Ext(relpath))
	if filepath.Base(prefix) == "index" {
		prefix = filepath.Dir(prefix)
	}
	route := routeForPage(prefix + "/" + partialUrlpath)
	return route
}

type codeGenResult struct {
	Pages []*page
	code  []byte
}

func genCodePage(g *pageCodeGen) (*codeGenResult, error) {
	var result codeGenResult

	g.commentPrintf("// Code generated by Pushup; DO NOT EDIT.\n")
	g.commentPrintf("// Version: ")
	printVersion(&g.comments)
	g.commentPrintf("\n")

	type field struct {
		name string
		typ  string
	}

	// main page
	{
		typename := generatedTypename(g.pfile, upFilePage)
		route := routeForPage(g.pfile.relpath())
		page := page{
			PkgPath: pkgPathForPage(g.modPath, g.pfile.relpath()),
			Name:    typename,
			Route:   route,
			Role:    routePage,
		}
		result.Pages = append(result.Pages, &page)
		g.bodyPrintf("type %s struct {\n", typename)
		fields := []field{}
		for _, field := range fields {
			g.bodyPrintf("%s %s\n", field.name, field.typ)
		}
		g.bodyPrintf("}\n")

		g.bodyPrintf("func (%s *%s) buildCliArgs() []string {\n", methodReceiverName, typename)
		g.bodyPrintf("  return %#v\n", os.Args)
		g.bodyPrintf("}\n\n")

		g.used("net/http")
		g.bodyPrintf("func (%s *%s) Respond(w http.ResponseWriter, req *http.Request) error {\n", methodReceiverName, typename)

		// NOTE(paulsmith): we might want to encapsulate this in its own
		// function/method, but would have to figure out the interplay between
		// user code and control flow, i.e., return an error if the handler
		// wants to skip rendering, redirect, etc.
		if h := g.page.handler; h != nil {
			srcLineNo := g.lineNo(h.Pos())
			lines := strings.Split(h.Code, "\n")
			for _, line := range lines {
				if g.lineDirectivesEnabled {
					g.emitLineDirective(srcLineNo)
				}
				g.bodyPrintf("  %s\n", line)
				srcLineNo++
			}
		}

		// Make a new scope for the user's code block and HTML. This will help (but not fully prevent)
		// name collisions with the surrounding code.
		g.bodyPrintf("\n// Begin user Go code and HTML\n")
		g.bodyPrintf("{\n")

		// render the main body contents
		g.used("bytes")
		save := g.ioWriterVar
		g.ioWriterVar = "__pushup_b"
		g.bodyPrintf("  %s := new(bytes.Buffer)\n", g.ioWriterVar)
		g.generate()
		// flush output
		g.bodyPrintf("  io.Copy(w, %s)\n", g.ioWriterVar)
		g.ioWriterVar = save

		// Close the scope we started for the user code and HTML.
		g.bodyPrintf("// End user Go code and HTML\n")
		g.bodyPrintf("}\n")

		// return from Respond()
		g.bodyPrintf("return nil\n")
		g.bodyPrintf("}\n")
	}

	for _, partial := range g.page.partials {
		typename := generatedTypenamePartial(partial, g.pfile)
		route := routeForPartial(g.pfile.relpath(), partial.urlpath())
		page := page{
			PkgPath: pkgPathForPage(g.modPath, g.pfile.relpath()),
			Name:    typename,
			Route:   route,
			Role:    routePartial,
		}
		result.Pages = append(result.Pages, &page)
		g.bodyPrintf("type %s struct {\n", typename)
		fields := []field{}
		for _, field := range fields {
			g.bodyPrintf("%s %s\n", field.name, field.typ)
		}
		g.bodyPrintf("}\n")

		g.used("net/http")
		g.bodyPrintf("func (%s *%s) Respond(w http.ResponseWriter, req *http.Request) error {\n", methodReceiverName, typename)

		// NOTE(paulsmith): we might want to encapsulate this in its own
		// function/method, but would have to figure out the interplay between
		// user code and control flow, i.e., return an error if the handler
		// wants to skip rendering, redirect, etc.
		if h := g.page.handler; h != nil {
			srcLineNo := g.lineNo(h.Pos())
			lines := strings.Split(h.Code, "\n")
			for _, line := range lines {
				if g.lineDirectivesEnabled {
					g.emitLineDirective(srcLineNo)
				}
				g.bodyPrintf("  %s\n", line)
				srcLineNo++
			}
		}

		// Make a new scope for the user's code block and HTML. This will help (but not fully prevent)
		// name collisions with the surrounding code.
		g.bodyPrintf("// Begin user Go code and HTML\n")
		g.bodyPrintf("{\n")

		// FIXME(paulsmith): need to generate code for everything but emitting
		// top-level page values to the output
		g.genNodePartial(NodeList(g.page.nodes), partial)

		// Close the scope we started for the user code and HTML.
		g.bodyPrintf("// End user Go code and HTML\n")
		g.bodyPrintf("}\n")

		g.bodyPrintf("return nil\n")
		g.bodyPrintf("}\n")
	}

	// we write out imports at the end because we need to know what was
	// actually used by the body code
	g.importDeclPrintf("import (\n")
	for decl, ok := range g.imports {
		if ok {
			if decl.PkgName != "" {
				g.importDeclPrintf("%s ", decl.PkgName)
			}
			g.importDeclPrintf("%s\n", decl.Path)
		}
	}
	g.importDeclPrintf(")\n\n")

	g.commentPrintf("/*\n")
	if err := json.NewEncoder(&g.comments).Encode(result.Pages); err != nil {
		return nil, fmt.Errorf("encoding link metadata: %w", err)
	}
	g.commentPrintf("*/\n")

	raw, err := g.readAll()
	if err != nil {
		return nil, fmt.Errorf("reading all buffers: %w", err)
	}

	result.code, err = format.Source(raw)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil, fmt.Errorf("gofmt the generated code: %w", err)
	}

	return &result, nil
}

// generatedTypename returns the name of the type of the Go struct that
// holds the generated code for the Pushup page and related methods.
func generatedTypename(pfile projectFile, ftype upFileType) string {
	filename := filepath.Base(pfile.path)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	typename := typenameFromFilename(filename)
	if ftype == upFilePage {
		typename += "Page"
	}
	return typename
}

func generatedTypenamePartial(partial *partial, pfile projectFile) string {
	relpath := pfile.relpath()
	ext := filepath.Ext(relpath)
	relpath = relpath[:len(relpath)-len(ext)]
	typename := typenameFromFilename(strings.Join([]string{relpath, partial.urlpath()}, "/"))
	result := typename + "Partial"
	return result
}

func typenameFromFilename(path string) string {
	buf := make([]rune, len(path))
	i := 0
	wordBoundary := true
	for _, r := range path {
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
