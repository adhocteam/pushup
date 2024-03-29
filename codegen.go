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

type importDecl struct {
	pkgName string
	path    string
}

type layout struct {
	imports []importDecl
	nodes   []node
}

func newLayoutFromTree(tree *syntaxTree) (*layout, error) {
	layout := &layout{}
	n := 0
	var f inspector = func(e node) bool {
		switch e := e.(type) {
		case *nodeImport:
			layout.imports = append(layout.imports, e.decl)
		default:
			layout.nodes = append(layout.nodes, e)
			n++
		}
		return false
	}
	inspect(nodeList(tree.nodes), f)
	layout.nodes = layout.nodes[:n]
	return layout, nil
}

type layoutCodeGen struct {
	layout  *layout
	pfile   projectFile
	imports map[importDecl]bool

	// source code of .up file, needed for mapping line numbers back to
	// original source in stack traces.
	source string

	// buffer for the package clauses and import declarations at the top of
	// a Go source file.
	outb bytes.Buffer

	// buffer for the main body of a Go source file, i.e., the top-level
	// declarations.
	bodyb bytes.Buffer

	ioWriterVar           string
	lineDirectivesEnabled bool
}

func newLayoutCodeGen(layout *layout, pfile projectFile, source string) *layoutCodeGen {
	l := &layoutCodeGen{
		layout:                layout,
		pfile:                 pfile,
		source:                source,
		imports:               make(map[importDecl]bool),
		ioWriterVar:           "w",
		lineDirectivesEnabled: true,
	}
	for _, im := range layout.imports {
		l.imports[im] = true
	}
	return l
}

func lineCount(s string) int {
	return strings.Count(s, "\n") + 1
}

func (c *layoutCodeGen) lineNo(s span) int {
	return lineCount(c.source[:s.start+1])
}

func (g *layoutCodeGen) outPrintf(format string, args ...any) {
	fmt.Fprintf(&g.outb, format, args...)
}

func (g *layoutCodeGen) bodyPrintf(format string, args ...any) {
	fmt.Fprintf(&g.bodyb, format, args...)
}

func (g *layoutCodeGen) used(path ...string) {
	for _, p := range path {
		g.imports[importDecl{path: strconv.Quote(p), pkgName: ""}] = true
	}
}

func (g *layoutCodeGen) nodeLineNo(e node) {
	if g.lineDirectivesEnabled {
		g.emitLineDirective(g.lineNo(e.Pos()))
	}
}

func (g *layoutCodeGen) emitLineDirective(n int) {
	g.bodyPrintf("//line %s:%d\n", g.pfile.relpath(), n)
}

func (g *layoutCodeGen) generate() {
	nodes := g.layout.nodes
	g.genNode(nodeList(nodes))
}

func (g *layoutCodeGen) genNode(n node) {
	var f inspector
	f = func(e node) bool {
		// TODO(paulsmith): these could be functions so they can be reused
		switch e := e.(type) {
		case *nodeLiteral:
			g.used("io")
			g.nodeLineNo(e)
			g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(e.str))
		case *nodeElement:
			g.used("io")
			g.nodeLineNo(e)
			f(nodeList(e.startTagNodes))
			f(nodeList(e.children))
			g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(e.tag.end()))
			return false
		case *nodeGoStrExpr:
			g.nodeLineNo(e)
			g.bodyPrintf("printEscaped(%s, %s)\n", g.ioWriterVar, e.expr)
		case *nodeGoCode:
			if e.context != inlineGoCode {
				panic(fmt.Sprintf("internal error: expected inlineGoCode, got %v", e.context))
			}
			srcLineNo := g.lineNo(e.Pos())
			lines := strings.Split(e.code, "\n")
			for _, line := range lines {
				// FIXME(paulsmith): leaky abstraction
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
			// FIXME(paulsmith): prune these out in newLayoutFromTree
			panic("partials are not allowed in layouts")
		case *nodeLayout:
			// nothing to do
		case *nodeImport:
			// nothing to do
		}
		return true
	}
	inspect(n, f)
}

func layoutName(relpath string) string {
	ext := filepath.Ext(relpath)
	if ext != upFileExt {
		panic("internal error: unexpected file extension " + ext)
	}
	return strings.TrimSuffix(relpath, ext)
}

const methodReceiverName = "up"

func genCodeLayout(g *layoutCodeGen) ([]byte, error) {
	// FIXME(paulsmith): need way to specify this as user
	packageName := "build"

	g.outPrintf("// this file is mechanically generated, do not edit!\n")
	g.outPrintf("// version: ")
	printVersion(&g.outb)
	g.outPrintf("\n")
	g.outPrintf("package %s\n\n", packageName)

	type field struct {
		name string
		typ  string
	}

	fields := []field{}

	typename := generatedTypename(g.pfile, upFileLayout)

	g.bodyPrintf("type %s struct {\n", typename)
	for _, field := range fields {
		g.bodyPrintf("%s %s\n", field.name, field.typ)
	}
	g.bodyPrintf("}\n")

	g.bodyPrintf("func (%s *%s) buildCliArgs() []string {\n", methodReceiverName, typename)
	g.bodyPrintf("  return %#v\n", os.Args)
	g.bodyPrintf("}\n\n")

	g.bodyPrintf("func init() {\n")
	g.bodyPrintf("  layouts[\"%s\"] = new(%s)\n", layoutName(g.pfile.relpath()), typename)
	g.bodyPrintf("}\n\n")

	g.used("net/http", "html/template")
	g.bodyPrintf("func (%s *%s) Respond(w http.ResponseWriter, req *http.Request, sections map[string]chan template.HTML) error {\n", methodReceiverName, typename)

	// sections support
	g.used("html/template")
	g.bodyPrintf(`
sectionDefined := func(name string) bool {
	_, ok := sections[name]
	return ok
}
_ = sectionDefined

outputSection := func(name string) template.HTML {
	return <-sections[name]
}
`)

	// Make a new scope for the user's code block and HTML. This will help (but not fully prevent)
	// name collisions with the surrounding code.
	g.bodyPrintf("\n// Begin user Go code and HTML\n")
	g.bodyPrintf("{\n")

	g.generate()

	// Close the scope we started for the user code and HTML.
	g.bodyPrintf("// End user Go code and HTML\n")
	g.bodyPrintf("}\n")

	g.bodyPrintf("return nil\n")
	g.bodyPrintf("}\n")

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
		return nil, fmt.Errorf("gofmt the generated code: %w", err)
	}

	return formatted, nil
}

// page represents a Pushup page that has been parsed and is ready for code
// generation.
type page struct {
	layout   string
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
	page := &page{
		layout:   "default",
		sections: make(map[string]*nodeBlock),
	}

	layoutSet := false
	n := 0
	var err error

	// this pass over the syntax tree nodes enforces invariants (only one
	// handler may be declared per page, layout may only be set once) and
	// aggregates imports and sections for easier access in the subsequent
	// code generation phase. as a result, some nodes are removed from the
	// tree.
	var f inspector
	f = func(e node) bool {
		switch e := e.(type) {
		case *nodeImport:
			page.imports = append(page.imports, e.decl)
		case *nodeLayout:
			if layoutSet {
				err = fmt.Errorf("layout already set as %q", page.layout)
				return false
			}
			if e.name == "!" {
				page.layout = ""
			} else {
				page.layout = e.name
			}
			layoutSet = true
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
			case *nodeLayout:
				// nothing to do
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

func newPageCodeGen(page *page, pfile projectFile, source string) *pageCodeGen {
	g := &pageCodeGen{
		page:                  page,
		pfile:                 pfile,
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

func (g *pageCodeGen) genNode(n node) {
	var f inspector
	f = func(e node) bool {
		switch e := e.(type) {
		case *nodeLiteral:
			g.used("io")
			g.nodeLineNo(e)
			g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(e.str))
		case *nodeElement:
			g.used("io")
			g.nodeLineNo(e)
			f(nodeList(e.startTagNodes))
			f(nodeList(e.children))
			g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(e.tag.end()))
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
		case *nodeLayout:
			// nothing to do
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
// relpath is the path to the Pushup file, relative to its containing app
// directory in the Pushup project (so that part should not be part of the
// path).
func routeForPage(relpath string) string {
	var dirs []string
	dir := filepath.Dir(relpath)
	if dir != "." {
		dirs = strings.Split(dir, string([]rune{os.PathSeparator}))
	}
	file := filepath.Base(relpath)
	base := strings.TrimSuffix(file, filepath.Ext(file))
	var route string
	if base != "index" {
		dirs = append(dirs, base)
	}
	for i := range dirs {
		if strings.HasPrefix(dirs[i], "$") {
			dirs[i] = ":" + dirs[i][1:]
		}
	}
	route = "/" + strings.Join(dirs, "/")
	if base == "index" && route[len(route)-1] != '/' {
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

	// FIXME(paulsmith): need way to specify this as user
	packageName := "build"

	g.outPrintf("// this file is mechanically generated, do not edit!\n")
	g.outPrintf("// version: ")
	printVersion(&g.outb)
	g.outPrintf("\n")
	g.outPrintf("package %s\n\n", packageName)

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

		// TODO(paulsmith): this is where a flag that could conditionally toggle the rendering
		// of the layout could go - maybe a special header in request object?
		g.used("log", "sync", "context", "time")
		g.bodyPrintf(
			`
			var wg sync.WaitGroup
			layout := getLayout("%s")
			ctx, cancel := context.WithTimeout(req.Context(), time.Second * 5)
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer cancel()
				if err := layout.Respond(w, req.WithContext(ctx), sections); err != nil {
					log.Printf("error responding with layout: %%v", err)
					panic(err)
				}
			}()
		`, g.page.layout)

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

		// Wait for layout to finish rendering
		g.bodyPrintf("wg.Wait()\n")

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
	relpath := pfile.relpath()
	ext := filepath.Ext(relpath)
	relpath = relpath[:len(relpath)-len(ext)]
	typename := typenameFromPath(relpath)
	var suffix string
	switch ftype {
	case upFilePage:
		suffix = "Page"
	case upFileLayout:
		suffix = "Layout"
	default:
		panic("unhandled file type")
	}
	result := typename + suffix
	return result
}

func generatedTypenamePartial(partial *partial, pfile projectFile) string {
	relpath := pfile.relpath()
	ext := filepath.Ext(relpath)
	relpath = relpath[:len(relpath)-len(ext)]
	typename := typenameFromPath(strings.Join([]string{relpath, partial.urlpath()}, "/"))
	result := typename + "Partial"
	return result
}

func typenameFromPath(path string) string {
	path = strings.ReplaceAll(path, "$", "DollarSign_")
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
