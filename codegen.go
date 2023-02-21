package main

import (
	"bytes"
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

// importDecl represents a Go import declaration.
type importDecl struct {
	pkgName string
	path    string
}

func lineCount(s string) int {
	return strings.Count(s, "\n") + 1
}

const methodReceiverName = "up"

// page represents a Pushup page that has been parsed and is ready for code
// generation.
type page struct {
	imports  []importDecl
	handler  *nodeGoCode
	nodes    []node
	sections map[string]*nodeBlock

	// partials is a list of all top-level inline partials in this page.
	partials []*partial
}

// partial represents an inline partial in a Pushup page.
type partial struct {
	node     node
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
func newPageFromTree(tree *syntaxTree) (*page, error) {
	page := new(page)
	page.sections = make(map[string]*nodeBlock)

	n := 0
	var err error

	// this pass over the syntax tree nodes enforces invariants (only one
	// handler may be declared per page) and aggregates imports and sections
	// for easier access in the subsequent code generation phase. as a
	// result, some nodes are removed from the tree.
	var f inspector
	f = func(e node) bool {
		switch e := e.(type) {
		case *nodeImport:
			page.imports = append(page.imports, e.decl)
		case *nodeGoCode:
			if e.context == handlerGoCode {
				if page.handler != nil {
					err = fmt.Errorf("only one handler per page can be defined")
					return false
				}
				page.handler = e
			} else {
				tree.nodes[n] = e
				n++
			}
		case nodeList:
			for _, x := range e {
				f(x)
			}
		case *nodeSection:
			page.sections[e.name] = e.block
		default:
			tree.nodes[n] = e
			n++
		}
		// don't recurse into child nodes
		return false
	}
	inspect(nodeList(tree.nodes), f)
	if err != nil {
		return nil, err
	}

	page.nodes = tree.nodes[:n]

	// this pass is for inline partials. it needs to be separate because the
	// traversal of the tree is slightly different than the pass above.
	{
		var currentPartial *partial
		var f inspector
		f = func(e node) bool {
			switch e := e.(type) {
			case *nodeLiteral:
			case *nodeElement:
				f(nodeList(e.startTagNodes))
				f(nodeList(e.children))
				return false
			case *nodeGoStrExpr:
			case *nodeGoCode:
			case *nodeIf:
				f(e.then)
				if e.alt != nil {
					f(e.alt)
				}
				return false
			case nodeList:
				for _, x := range e {
					f(x)
				}
				return false
			case *nodeFor:
				f(e.block)
				return false
			case *nodeBlock:
				f(nodeList(e.nodes))
				return false
			case *nodeSection:
				f(e.block)
				return false
			case *nodePartial:
				p := &partial{node: e, name: e.name, parent: currentPartial}
				if currentPartial != nil {
					currentPartial.children = append(currentPartial.children, p)
				}
				prevPartial := currentPartial
				currentPartial = p
				f(e.block)
				currentPartial = prevPartial
				page.partials = append(page.partials, p)
				return false
			case *nodeImport:
				// nothing to do
			}
			return false
		}
		inspect(nodeList(page.nodes), f)
	}

	return page, nil
}

type pageCodeGen struct {
	page    *page
	pfile   projectFile
	pkgName string
	source  string
	imports map[importDecl]bool

	// buffer for the package clauses and import declarations at the top of
	// a Go source file.
	outb bytes.Buffer

	// buffer for the main body of a Go source file, i.e., the top-level
	// declarations.
	bodyb bytes.Buffer

	ioWriterVar           string
	lineDirectivesEnabled bool
}

func newPageCodeGen(page *page, pfile projectFile, source string, pkgName string) *pageCodeGen {
	g := &pageCodeGen{
		page:                  page,
		pfile:                 pfile,
		pkgName:               pkgName,
		source:                source,
		imports:               make(map[importDecl]bool),
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
		g.imports[importDecl{path: strconv.Quote(p), pkgName: ""}] = true
	}
}

func (g *pageCodeGen) nodeLineNo(e node) {
	if g.lineDirectivesEnabled {
		g.emitLineDirective(g.lineNo(e.Pos()))
	}
}

func (c *pageCodeGen) lineNo(s span) int {
	return lineCount(c.source[:s.start+1])
}

func (g *pageCodeGen) emitLineDirective(n int) {
	g.bodyPrintf("//line %s:%d\n", g.pfile.relpath(), n)
}

func (g *pageCodeGen) outPrintf(format string, args ...any) {
	fmt.Fprintf(&g.outb, format, args...)
}

func (g *pageCodeGen) bodyPrintf(format string, args ...any) {
	fmt.Fprintf(&g.bodyb, format, args...)
}

func (g *pageCodeGen) generate() {
	nodes := g.page.nodes
	g.genNode(nodeList(nodes))
}

func (g *pageCodeGen) genElement(e *nodeElement, f inspector) {
	g.used("io")
	g.nodeLineNo(e)
	f(nodeList(e.startTagNodes))
	f(nodeList(e.children))
	g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(e.tag.end()))
}

func (g *pageCodeGen) genNode(n node) {
	var f inspector
	f = func(e node) bool {
		switch e := e.(type) {
		case *nodeLiteral:
			g.used("io")
			g.nodeLineNo(e)
			g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(e.str))
		case *nodeElement:
			g.genElement(e, f)
			return false
		case *nodeGoStrExpr:
			g.nodeLineNo(e)
			g.bodyPrintf("printEscaped(%s, %s)\n", g.ioWriterVar, e.expr)
		case *nodeGoCode:
			if e.context != inlineGoCode {
				panic("internal error: expected inlineGoCode")
			}
			srcLineNo := g.lineNo(e.Pos())
			lines := strings.Split(e.code, "\n")
			for _, line := range lines {
				if g.lineDirectivesEnabled {
					g.emitLineDirective(srcLineNo)
				}
				g.bodyPrintf("%s\n", line)
				srcLineNo++
			}
		case *nodeIf:
			g.bodyPrintf("if %s {\n", e.cond.expr)
			f(e.then)
			if e.alt == nil {
				g.bodyPrintf("}\n")
			} else {
				g.bodyPrintf("} else {\n")
				f(e.alt)
				g.bodyPrintf("}\n")
			}
			return false
		case nodeList:
			for _, x := range e {
				f(x)
			}
			return false
		case *nodeFor:
			g.bodyPrintf("for %s {\n", e.clause.code)
			f(e.block)
			g.bodyPrintf("}\n")
			return false
		case *nodeBlock:
			f(nodeList(e.nodes))
			return false
		case *nodeSection:
			f(e.block)
			return false
		case *nodePartial:
			f(e.block)
			return false
		case *nodeImport:
			// nothing to do
		}
		return true
	}
	inspect(n, f)
}

// NOTE(paulsmith): per DOM spec, "In tree order is preorder, depth-first traversal of a tree."

func (g *pageCodeGen) genNodePartial(n node, p *partial) {
	var f inspector
	var state int
	const (
		stateStart int = iota
		stateInPartialScope
	)
	state = stateStart
	f = func(n node) bool {
		if n != nil {
			switch n := n.(type) {
			case *nodeLiteral:
				if state == stateInPartialScope {
					g.used("io")
					g.nodeLineNo(n)
					g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(n.str))
				}
			case *nodeElement:
				if state == stateInPartialScope {
					g.used("io")
					g.nodeLineNo(n)
					f(nodeList(n.startTagNodes))
				}
				f(nodeList(n.children))
				if state == stateInPartialScope {
					g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(n.tag.end()))
				}
				return false
			case *nodePartial:
				if n == p.node {
					state = stateInPartialScope
				}
				f(n.block)
				state = stateStart
				return false
			case *nodeGoStrExpr:
				if state == stateInPartialScope {
					g.nodeLineNo(n)
					g.bodyPrintf("printEscaped(%s, %s)\n", g.ioWriterVar, n.expr)
				}
			case *nodeFor:
				if state == stateInPartialScope {
					g.bodyPrintf("for %s {\n", n.clause.code)
					f(n.block)
					g.bodyPrintf("}\n")
				}
				return false
			case *nodeIf:
				g.bodyPrintf("if %s {\n", n.cond.expr)
				f(n.then)
				if n.alt == nil {
					g.bodyPrintf("}\n")
				} else {
					g.bodyPrintf("} else {\n")
					f(n.alt)
					g.bodyPrintf("}\n")
				}
				return false
			case *nodeGoCode:
				if n.context != inlineGoCode {
					panic("internal error: expected inlineGoCode")
				}
				srcLineNo := g.lineNo(n.Pos())
				lines := strings.Split(n.code, "\n")
				for _, line := range lines {
					if g.lineDirectivesEnabled {
						g.emitLineDirective(srcLineNo)
					}
					g.bodyPrintf("%s\n", line)
					srcLineNo++
				}
			case nodeList:
				for _, x := range n {
					f(x)
				}
			case *nodeBlock:
				for _, x := range n.nodes {
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

// routeForPage produces the URL path route from the name of the Pushup page.
// path is the path to the Pushup page file.
func routeForPage(path string) string {
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

func genCodePage(g *pageCodeGen) ([]byte, error) {
	type initRoute struct {
		typename string
		route    string
		role     string
	}
	var inits []initRoute

	g.outPrintf("// this file is mechanically generated, do not edit!\n")
	g.outPrintf("// version: ")
	printVersion(&g.outb)
	g.outPrintf("\n")
	g.outPrintf("package %s\n\n", g.pkgName)

	type field struct {
		name string
		typ  string
	}

	// main page
	{
		typename := generatedTypename(g.pfile, upFilePage)
		route := routeForPage(g.pfile.relpath())
		inits = append(inits, initRoute{typename: typename, route: route, role: "routePage"})

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
			lines := strings.Split(h.code, "\n")
			for _, line := range lines {
				if g.lineDirectivesEnabled {
					g.emitLineDirective(srcLineNo)
				}
				g.bodyPrintf("  %s\n", line)
				srcLineNo++
			}
		}

		g.used("html/template")
		g.bodyPrintf("// sections\n")
		g.bodyPrintf("sections := make(map[string]chan template.HTML)\n")
		g.bodyPrintf("sections[\"contents\"] = make(chan template.HTML)\n")
		for name := range g.page.sections {
			g.bodyPrintf("sections[%s] = make(chan template.HTML)\n", strconv.Quote(name))
		}

		// Make a new scope for the user's code block and HTML. This will help (but not fully prevent)
		// name collisions with the surrounding code.
		g.bodyPrintf("\n// Begin user Go code and HTML\n")
		g.bodyPrintf("{\n")

		g.bodyPrintf("var panicked any\n")
		// render the main body contents
		// TODO(paulsmith) could do these as a incremental stream
		// so the receiving end is just pulling individual chunks off
		g.bodyPrintf("wg.Add(1)\n")
		g.bodyPrintf("go func() {\n")
		g.bodyPrintf("  defer wg.Done()\n")
		g.bodyPrintf("  defer func() {\n")
		g.bodyPrintf("    if r := recover(); r != nil {\n")
		g.bodyPrintf("      if panicked == nil {\n")
		g.bodyPrintf("	      cancel()\n")
		g.bodyPrintf("	      panicked = r\n")
		g.bodyPrintf("	    }\n")
		g.bodyPrintf("    }\n")
		g.bodyPrintf("  }()\n")
		g.used("bytes", "html/template")
		save := g.ioWriterVar
		g.ioWriterVar = "__pushup_b"
		g.bodyPrintf("  %s := new(bytes.Buffer)\n", g.ioWriterVar)
		g.generate()
		g.bodyPrintf("  sections[\"contents\"] <- template.HTML(%s.String())\n", g.ioWriterVar)
		g.bodyPrintf("}()\n\n")
		g.ioWriterVar = save

		for name, block := range g.page.sections {
			save := g.ioWriterVar
			g.ioWriterVar = "__pushup_b"
			g.bodyPrintf("wg.Add(1)\n")
			g.bodyPrintf("go func() {\n")
			g.bodyPrintf("  defer wg.Done()\n")
			g.bodyPrintf("  defer func() {\n")
			g.bodyPrintf("    if r := recover(); r != nil {\n")
			g.bodyPrintf("      if panicked != nil {\n")
			g.bodyPrintf("	      cancel()\n")
			g.bodyPrintf("	      panicked = r\n")
			g.bodyPrintf("	    }\n")
			g.bodyPrintf("    }\n")
			g.bodyPrintf("  }()\n")
			g.bodyPrintf("  %s := new(bytes.Buffer)\n", g.ioWriterVar)
			g.genNode(block)
			g.bodyPrintf("  sections[%s] <- template.HTML(%s.String())\n", strconv.Quote(name), g.ioWriterVar)
			g.bodyPrintf("}()\n")
			g.ioWriterVar = save
		}

		// Check if any of the goroutines panicked
		g.bodyPrintf("if panicked != nil {\n")
		g.bodyPrintf("  close(sections[\"contents\"])\n")
		for name := range g.page.sections {
			g.bodyPrintf("  close(sections[%s])\n", strconv.Quote(name))
		}
		g.used("fmt")
		g.bodyPrintf("  return fmt.Errorf(\"goroutine panicked: %%v\", panicked)\n")
		g.bodyPrintf("}\n")

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
		inits = append(inits, initRoute{typename: typename, route: route, role: "routePartial"})

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
			lines := strings.Split(h.code, "\n")
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
		g.genNodePartial(nodeList(g.page.nodes), partial)

		// Close the scope we started for the user code and HTML.
		g.bodyPrintf("// End user Go code and HTML\n")
		g.bodyPrintf("}\n")

		g.bodyPrintf("return nil\n")
		g.bodyPrintf("}\n")
	}

	g.bodyPrintf("\nfunc init() {\n")
	for _, initRoute := range inits {
		g.bodyPrintf("  routes.add(%s, new(%s), %s)\n", strconv.Quote(initRoute.route), initRoute.typename, initRoute.role)
	}
	g.bodyPrintf("}\n\n")

	// we write out imports at the end because we need to know what was
	// actually used by the body code
	g.outPrintf("import (\n")
	for decl, ok := range g.imports {
		if ok {
			if decl.pkgName != "" {
				g.outPrintf("%s ", decl.pkgName)
			}
			g.outPrintf("%s\n", decl.path)
		}
	}
	g.outPrintf(")\n\n")

	raw, err := io.ReadAll(io.MultiReader(&g.outb, &g.bodyb))
	if err != nil {
		return nil, fmt.Errorf("reading all buffers: %w", err)
	}

	formatted, err := format.Source(raw)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil, fmt.Errorf("gofmt the generated code: %w", err)
	}

	return formatted, nil
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
