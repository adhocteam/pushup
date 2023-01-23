package main

import (
	"bytes"
	"fmt"
	goparser "go/parser"
	"go/scanner"
	"go/token"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

func parse(source string) (tree *syntaxTree, err error) {
	p := newParser(source)
	defer func() {
		if e := recover(); e != nil {
			if se, ok := e.(syntaxError); ok {
				tree = nil
				err = se
			} else {
				panic(e)
			}
		}
	}()
	tree = p.htmlParser.parseDocument()
	return
}

// parser is the main Pushup parser. it is comprised of an HTML parser and a Go
// parser, and handles Pushup template language syntax, too. it starts in HTML
// mode, and switches to parsing Go code when it encounters the transition
// symbol.
type parser struct {
	src string
	// byte offset into src representing the maximum position read
	offset int

	htmlParser *htmlParser
	codeParser *codeParser
}

func newParser(source string) *parser {
	p := new(parser)
	p.src = source
	p.offset = 0
	p.htmlParser = &htmlParser{parser: p}
	p.codeParser = &codeParser{parser: p}
	return p
}

// remainingSource returns the source code starting from the internal byte
// offset all the way to the end.
func (p *parser) remainingSource() string {
	return p.sourceFrom(p.offset)
}

// sourceFrom returns the source code starting from the given byte offset. it
// returns the empty string if the offset is greater than the source code's
// length.
func (p *parser) sourceFrom(offset int) string {
	if len(p.src) >= offset {
		return p.src[offset:]
	}
	return ""
}

// advanceOffset advances the internal byte offset position by delta amount.
func (p *parser) advanceOffset(delta int) {
	p.offset += delta
}

// syntaxError represents a synax error in the Pushup template language.
type syntaxError struct {
	// err is the underlying error that caused this syntax error
	err error
	// lineNo and column are the positions in the source code where the
	// error occurred
	lineNo int
	column int
}

func (e syntaxError) Error() string {
	// TODO(paulsmith): add source file name
	return fmt.Sprintf("%d:%d: %s", e.lineNo, e.column, e.err.Error())
}

// errorf signals that a syntax error in the Pushup template language has been
// detected. The Pushup parser uses panic mode error handling, so a function
// calling the parser higher up in the call stack can recover from the panic
// and test for a syntax error (syntaxError type).
func (p *parser) errorf(format string, args ...any) {
	offset := p.offset
	if offset >= len(p.src) {
		offset = len(p.src) - 1
	}
	upToErr := p.src[:offset]
	lineNo := strings.Count(upToErr, "\n") + 1
	lastNL := strings.LastIndex(upToErr, "\n")
	column := p.offset + 1
	if lastNL > -1 {
		column = p.offset - lastNL
	}
	panic(syntaxError{fmt.Errorf(format, args...), lineNo, column})
}

// htmlParser is the Pushup HTML parser. It wraps the golang.org/x/net/html
// tokenizer, which is an HTML 5 specification-compliant parser. It changes
// control to the Go code parser (codeParser type) if it encounters the
// transition symbol in the course of tokenizing HTML documents.
type htmlParser struct {
	// a pointer to the main Pushup parser
	parser *parser

	// current token
	toktyp  html.TokenType
	tagname []byte
	err     error
	raw     string
	attrs   []*attr

	// the global parser offset at the beginning of a new token
	start int
}

func (p *htmlParser) errorf(format string, args ...any) {
	p.parser.errorf(format, args...)
}

func (p *htmlParser) advance() {
	// NOTE(paulsmith): we're re-creating a tokenizer each time through
	// the loop, with the starting point of the source text moved up by the
	// length of the previous token, in order to synchronize the position
	// in the source between the code parser and the HTML parser. this is
	// probably inefficient and could be done "better" and more efficiently
	// by reusing the tokenizer, as for sure it generates more garbage. but
	// would need to profile to see if this is actually a big problem to
	// end users, and in any case, it's only during compilation, so doesn't
	// impact the runtime web application.
	tokenizer := html.NewTokenizer(strings.NewReader(p.parser.remainingSource()))
	tokenizer.SetMaxBuf(0) // unlimited buffer size
	p.toktyp = tokenizer.Next()
	p.err = tokenizer.Err()
	p.raw = string(tokenizer.Raw())
	p.attrs = nil
	var hasAttr bool
	p.tagname, hasAttr = tokenizer.TagName()
	if hasAttr && p.err == nil {
		p.attrs, p.err = scanAttrs(p.raw)
	}
	p.start = p.parser.offset
	p.parser.advanceOffset(len(p.raw))
}

func isAllWhitespace(s string) bool {
	for s != "" {
		r, size := utf8.DecodeRuneInString(s)
		if !unicode.IsSpace(r) {
			return false
		}
		s = s[size:]
	}
	return true
}

func (p *htmlParser) skipWhitespace() []*nodeLiteral {
	var result []*nodeLiteral
	for p.toktyp == html.TextToken && isAllWhitespace(p.raw) {
		n := nodeLiteral{str: p.raw, pos: span{start: p.start, end: p.parser.offset}}
		result = append(result, &n)
		p.advance()
	}
	return result
}

// transition character: transitions the parser from HTML markup to Go code: ^
const (
	transSym    = '^'
	transSymStr = string(transSym)
	transSymEsc = transSymStr + transSymStr
)

func (p *htmlParser) parseAttributeNameOrValue(nameOrValue string, nameOrValueStartPos, nameOrValueEndPos int, pos int) ([]node, int) {
	var nodes []node
	if strings.ContainsRune(nameOrValue, transSym) {
		for pos < nameOrValueEndPos && strings.ContainsRune(nameOrValue, transSym) {
			if idx := strings.IndexRune(nameOrValue, transSym); idx > 0 {
				nodes = append(nodes, p.emitLiteralFromRange(pos, pos+idx))
				pos += idx
				nameOrValue = nameOrValue[idx:]
			}
			if strings.HasPrefix(nameOrValue, transSymStr+transSymStr) {
				nodes = append(nodes, p.emitLiteralFromRange(pos, pos+1))
				pos += 2
				nameOrValue = nameOrValue[2:]
			} else {
				pos++
				saveParser := p.parser
				p.parser = newParser(nameOrValue[1:])
				nodes = append(nodes, p.transition())
				bytesRead := p.parser.offset
				pos += bytesRead
				p.parser = saveParser
				nameOrValue = nameOrValue[bytesRead:]
			}
		}
	} else {
		nodes = append(nodes, p.emitLiteralFromRange(nameOrValueStartPos, nameOrValueEndPos))
		pos = nameOrValueEndPos
	}
	return nodes, pos
}

func (p *htmlParser) emitLiteralFromRange(start, end int) node {
	e := new(nodeLiteral)
	e.str = p.raw[start:end]
	e.pos.start = p.start + start
	e.pos.end = p.start + end
	return e
}

func (p *htmlParser) parseStartTag() []node {
	var nodes []node

	if len(p.attrs) == 0 {
		nodes = append(nodes, p.emitLiteralFromRange(0, len(p.raw)))
	} else {
		// bytesRead keeps track of how far we've parsed into this p.raw string
		bytesRead := 0

		for _, attr := range p.attrs {
			name := attr.name.string
			value := attr.value.string
			nameStartPos := int(attr.name.start)
			valStartPos := int(attr.value.start)
			nameEndPos := nameStartPos + len(name)
			valEndPos := valStartPos + len(value)

			// emit raw chars between tag name or last attribute and this
			// attribute
			if n := nameStartPos - bytesRead; n > 0 {
				nodes = append(nodes, p.emitLiteralFromRange(bytesRead, bytesRead+n))
				bytesRead += n
			}

			// emit attribute name
			nameNodes, newPos := p.parseAttributeNameOrValue(name, nameStartPos, nameEndPos, bytesRead)
			nodes = append(nodes, nameNodes...)
			bytesRead = newPos

			if valStartPos > bytesRead {
				// emit any chars, including equals and quotes, between
				// attribute name and attribute value, if any
				nodes = append(nodes, p.emitLiteralFromRange(bytesRead, valStartPos))
				bytesRead = valStartPos

				// emit attribute value
				valNodes, newPos := p.parseAttributeNameOrValue(value, valStartPos, valEndPos, bytesRead)
				nodes = append(nodes, valNodes...)
				bytesRead = newPos
			}
		}

		// emit anything from the last attribute to the close of the tag
		nodes = append(nodes, p.emitLiteralFromRange(bytesRead, len(p.raw)))
	}

	return nodes
}

func (p *htmlParser) emitLiteral() node {
	e := new(nodeLiteral)
	e.pos.start = p.start
	e.pos.end = p.parser.offset
	e.str = p.raw
	return e
}

func (p *htmlParser) parseLayout() node {
	idx := strings.IndexRune(p.raw, transSym)
	s := p.raw[idx+1+len("layout"):]
	n := 0
	if len(s) < 1 || s[0] != ' ' {
		p.errorf(transSymStr + "layout must be followed by a space")
	}
	s = s[1:]
	n++
	e := new(nodeLayout)
	if len(s) > 0 && s[0] == '!' {
		e.name = "!"
		n++
	} else {
		var name []rune
		for {
			r, size := utf8.DecodeRuneInString(s)
			if r == 0 {
				break
			}
			if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' || r == '-' || r == '.' {
				name = append(name, r)
				s = s[size:]
				n += size
			} else {
				break
			}
		}
		e.name = string(name)
	}
	e.pos.start = p.start + idx + 1
	newOffset := e.pos.start + len("layout") + n
	e.pos.end = newOffset
	p.parser.offset = newOffset
	return e
}

func (p *htmlParser) parseTextToken() []node {
	if !strings.ContainsRune(p.raw, transSym) {
		return []node{p.emitLiteral()}
	}

	if escaped := strings.Index(p.raw, transSymEsc); escaped >= 0 {
		// it's an escaped transition symbol
		nodes := []node{}
		if escaped > 0 {
			// emit the leading text before the doubled escape
			e := new(nodeLiteral)
			e.pos.start = p.start
			e.pos.end = p.start + escaped
			e.str = p.raw[:escaped]
			nodes = append(nodes, e)
		}
		e := new(nodeLiteral)
		e.pos.start = p.start + escaped
		e.pos.end = p.start + escaped + 2
		e.str = transSymStr
		nodes = append(nodes, e)
		p.parser.offset = p.start + escaped + 2
		return nodes
	}

	idx := strings.IndexRune(p.raw, transSym)
	if strings.HasPrefix(p.raw[idx+1:], "layout") {
		return []node{p.parseLayout()}
	}
	nodes := []node{}
	newOffset := p.start + idx + 1
	p.parser.offset = newOffset
	leading := p.raw[:idx]
	if idx > 0 {
		e := new(nodeLiteral)
		e.pos.start = p.start
		e.pos.end = p.start + len(leading)
		e.str = leading
		nodes = append(nodes, e)
	}
	// NOTE(paulsmith): this bubbles up nil due to parseImportKeyword,
	// the result of which we don't treat as a node in the syntax tree
	if e := p.transition(); e != nil {
		nodes = append(nodes, e)
	}
	return nodes
}

func (p *htmlParser) parseDocument() *syntaxTree {
	tree := new(syntaxTree)

tokenLoop:
	for {
		p.advance()
		if p.toktyp == html.ErrorToken {
			if p.err == io.EOF {
				break tokenLoop
			} else {
				p.errorf("HTML tokenizer: %w", p.err)
			}
		}
		switch p.toktyp {
		case html.StartTagToken, html.SelfClosingTagToken:
			tree.nodes = append(tree.nodes, p.parseStartTag()...)
		case html.EndTagToken, html.DoctypeToken, html.CommentToken:
			tree.nodes = append(tree.nodes, p.emitLiteral())
		case html.TextToken:
			tree.nodes = append(tree.nodes, p.parseTextToken()...)
		default:
			panic("")
		}
	}

	return tree
}

func (p *htmlParser) transition() node {
	codeParser := p.parser.codeParser
	codeParser.reset()
	e := codeParser.parseCode()
	return e
}

type tag struct {
	name  string
	attrs []*attr
}

func (t tag) String() string {
	if len(t.attrs) == 0 {
		return t.name
	}
	buf := bytes.NewBufferString(t.name)
	for _, a := range t.attrs {
		buf.WriteByte(' ')
		buf.WriteString(a.name.string)
		buf.WriteString(`="`)
		buf.WriteString(html.EscapeString(a.value.string))
		buf.WriteByte('"')
	}
	return buf.String()
}

func (t tag) start() string {
	return "<" + t.String() + ">"
}

func (t tag) end() string {
	return "</" + t.name + ">"
}

func newTag(tagname []byte, attrs []*attr) tag {
	return tag{name: string(tagname), attrs: attrs}
}

func (p *htmlParser) match(typ html.TokenType) bool {
	return p.toktyp == typ
}

func (p *htmlParser) parseElement() node {
	var result *nodeElement

	// FIXME(paulsmith): handle self-closing elements
	if !p.match(html.StartTagToken) {
		p.errorf("expected an HTML element start tag, got %s", p.toktyp)
	}

	result = new(nodeElement)
	result.tag = newTag(p.tagname, p.attrs)
	result.pos.start = p.parser.offset - len(p.raw)
	result.pos.end = p.parser.offset
	result.startTagNodes = p.parseStartTag()
	p.advance()

	result.children = p.parseChildren()

	if !p.match(html.EndTagToken) {
		p.errorf("expected an HTML element end tag, got %q", p.toktyp)
	}

	if result.tag.name != string(p.tagname) {
		p.errorf("expected </%s> end tag, got </%s>", result.tag.name, p.tagname)
	}

	// <text></text> elements are just for parsing
	if string(p.tagname) == "text" {
		return &nodeBlock{nodes: result.children}
	}

	return result
}

func (p *htmlParser) parseChildren() []node {
	var result []node // either *nodeElement or *nodeLiteral
	var elemStack []*nodeElement
loop:
	for {
		switch p.toktyp {
		case html.ErrorToken:
			if p.err == io.EOF {
				break loop
			} else {
				p.errorf("HTML tokenizer: %w", p.err)
			}
		case html.SelfClosingTagToken:
			elem := new(nodeElement)
			elem.tag = newTag(p.tagname, p.attrs)
			elem.pos.start = p.parser.offset - len(p.raw)
			elem.pos.end = p.parser.offset
			elem.startTagNodes = p.parseStartTag()
			p.advance()
			result = append(result, elem)
		case html.StartTagToken:
			elem := new(nodeElement)
			elem.tag = newTag(p.tagname, p.attrs)
			elem.pos.start = p.parser.offset - len(p.raw)
			elem.pos.end = p.parser.offset
			elem.startTagNodes = p.parseStartTag()
			p.advance()
			elem.children = p.parseChildren()
			result = append(result, elem)
			elemStack = append(elemStack, elem)
		case html.EndTagToken:
			if len(elemStack) == 0 {
				return result
			}
			elem := elemStack[len(elemStack)-1]
			if elem.tag.name == string(p.tagname) {
				elemStack = elemStack[:len(elemStack)-1]
				p.advance()
			} else {
				p.errorf("mismatch end tag, expected </%s>, got </%s>", elem.tag.name, p.tagname)
			}
		case html.TextToken:
			// TODO(paulsmith): de-dupe this logic
			if idx := strings.IndexRune(p.raw, transSym); idx >= 0 {
				if idx < len(p.raw)-1 && p.raw[idx+1] == transSym {
					// it's an escaped transition sym
					// TODO(paulsmith): emit transSym literal text expression
				} else {
					newOffset := p.start + idx + 1
					p.parser.offset = newOffset
					leading := p.raw[:idx]
					if idx > 0 {
						var htmlNode nodeLiteral
						htmlNode.pos.start = p.start
						htmlNode.pos.end = p.start + len(leading)
						htmlNode.str = leading
						result = append(result, &htmlNode)
					}
					e := p.transition()
					result = append(result, e)
				}
			} else {
				result = append(result, p.emitLiteral())
			}
			p.advance()
		case html.CommentToken:
			p.advance()
		case html.DoctypeToken:
			p.errorf("doctype token may not be a child of an element")
		default:
			panic(fmt.Sprintf("unexpected HTML token type %v", p.toktyp))
		}
	}

	return result
}

type Optional[T any] struct {
	value *T
}

func None[T any]() Optional[T] {
	return Optional[T]{}
}

func Some[T any](val T) Optional[T] {
	return Optional[T]{value: &val}
}

func Value[T any](o Optional[T]) (T, bool) {
	if o.value != nil {
		return *o.value, true
	} else {
		var zero T
		return zero, false
	}
}

type codeParser struct {
	parser         *parser
	baseOffset     int
	file           *token.File
	scanner        *scanner.Scanner
	bufferedToken  Optional[goToken]
	acceptedToken  Optional[goToken]
	lookaheadToken Optional[goToken]
}

func (p *codeParser) reset() {
	p.baseOffset = p.parser.offset
	fset := token.NewFileSet()
	source := p.parser.remainingSource()
	p.file = fset.AddFile("", fset.Base(), len(source))
	p.scanner = new(scanner.Scanner)
	p.scanner.Init(p.file, []byte(source), nil, scanner.ScanComments)
	p.bufferedToken = None[goToken]()
	p.acceptedToken = None[goToken]()
	p.lookaheadToken = None[goToken]()
}

func (p *codeParser) errorf(format string, args ...any) {
	p.parser.errorf(format, args...)
}

func (p *codeParser) sourceFrom(pos token.Pos) string {
	return p.parser.sourceFrom(p.baseOffset + p.file.Offset(pos))
}

func (p *codeParser) sourceRange(start, end int) string {
	return p.parser.src[start:end]
}

func (p *codeParser) lookahead() goToken {
	if tok, ok := Value(p.bufferedToken); ok {
		p.bufferedToken = None[goToken]()
		return tok
	}
	var t goToken
	var lit string
	t.pos, t.tok, lit = p.scanner.Scan()
	// from go/scanner docs:
	// If the returned token is a literal (token.IDENT, token.INT, token.FLOAT,
	// token.IMAG, token.CHAR, token.STRING) or token.COMMENT, the literal string
	// has the corresponding value.
	//
	// If the returned token is a keyword, the literal string is the keyword.
	//
	// If the returned token is token.SEMICOLON, the corresponding
	// literal string is ";" if the semicolon was present in the source,
	// and "\n" if the semicolon was inserted because of a newline or
	// at EOF.
	//
	// If the returned token is token.ILLEGAL, the literal string is the
	// offending character.
	//
	// In all other cases, Scan returns an empty literal string.
	if t.tok.IsLiteral() || t.tok.IsKeyword() || t.tok == token.SEMICOLON || t.tok == token.COMMENT || t.tok == token.ILLEGAL {
		t.lit = lit
	} else {
		t.lit = t.tok.String()
	}
	return t
}

type goToken struct {
	pos token.Pos
	tok token.Token
	lit string
}

func (t goToken) String() string {
	return t.lit
}

func (p *codeParser) peek() goToken {
	if tok, ok := Value(p.lookaheadToken); ok {
		return tok
	}
	tok := p.lookahead()
	p.lookaheadToken = Some(tok)
	return tok
}

// charAt() returns the byte at the offset in the input source string. because
// the Go tokenizer discards white space, we need this method in order to
// check for, for example, a space after an identifier in parsing an implicit
// expression, because that would denote the end of that simple expression in
// Pushup syntax.
func (p *codeParser) charAt(offset int) byte {
	if len(p.parser.src) > offset {
		return p.parser.src[offset]
	}
	return 0
}

func (p *codeParser) prev() goToken {
	if tok, ok := Value(p.acceptedToken); ok {
		return tok
	}
	panic("internal error: expected some accepted token, got none")
}

// sync synchronizes the global offset position in the main Pushup parser with
// the Go code scanner.
func (p *codeParser) sync() goToken {
	t := p.peek()
	// the Go scanner skips over whitespace so we need to be careful about the
	// logic for advancing the main parser internal source offset.
	p.parser.offset = p.tokenOffset(t) + len(t.String())
	return t
}

// advance consumes the lookahead token (which should be accessed via p.peek())
func (p *codeParser) advance() {
	p.acceptedToken = Some(p.sync())
	p.lookaheadToken = Some(p.lookahead())
}

// backup undoes a call to p.advance(). may only be called once between calls
// to p.advance(). must have called p.advance() at least once prior.
func (p *codeParser) backup() {
	if _, ok := Value(p.bufferedToken); ok {
		panic("internal error: p.backup() called more than once before p.advance()")
	}
	if _, ok := Value(p.lookaheadToken); !ok {
		panic("internal error: p.backup() called before p.advance()")
	}
	if tok, ok := Value(p.acceptedToken); ok {
		p.parser.offset = p.tokenOffset(tok)
	} else {
		panic("internal error: expected some accepted token, got none")
	}
	p.bufferedToken = p.lookaheadToken
	p.lookaheadToken = p.acceptedToken
}

func (p *codeParser) transition() *nodeBlock {
	htmlParser := p.parser.htmlParser
	htmlParser.advance()
	var stmtBlock nodeBlock
	ws := htmlParser.skipWhitespace()
	for _, n := range ws {
		stmtBlock.nodes = append(stmtBlock.nodes, n)
	}
	elem := htmlParser.parseElement()
	stmtBlock.nodes = append(stmtBlock.nodes, elem)
	p.reset()
	return &stmtBlock
}

func (p *codeParser) parseCode() node {
	// starting at the token just past the transSym indicating a transition from HTML
	// parsing to Go code parsing
	var e node
	tok := p.peek().tok
	lit := p.peek().lit
	if tok == token.IF {
		p.advance()
		e = p.parseIfStmt()
	} else if tok == token.IDENT && lit == "handler" {
		p.advance()
		e = p.parseHandlerKeyword()
		// NOTE(paulsmith): there is a tricky bit here where an implicit
		// expression in the form of an identifier token is next and we would
		// not be able to distinguish it from a keyword. this is also a problem
		// for name collisions because a user could create a variable named the
		// same as a keyword and then later try to use it in an implicit
		// expression, but it would be parsed with the keyword parsing flow
		// (which probably would lead to an infinite loop because it wouldn't
		// terminate and the user would be left with an unresponsive Pushup
		// compiler). a fix could be to have a notion of allowed contexts in
		// which a keyword block or an implicit expression could be used in the
		// surrounding markup, and only parse for either depending on which
		// context is current.
	} else if tok == token.IDENT && lit == "section" {
		p.advance()
		e = p.parseSectionKeyword()
	} else if tok == token.IDENT && lit == "partial" {
		p.advance()
		e = p.parsePartialKeyword()
	} else if tok == token.LBRACE {
		e = p.parseCodeBlock()
	} else if tok == token.IMPORT {
		p.advance()
		e = p.parseImportKeyword()
	} else if tok == token.FOR {
		p.advance()
		e = p.parseForStmt()
	} else if tok == token.LPAREN {
		p.advance()
		e = p.parseExplicitExpression()
	} else if tok == token.IDENT {
		e = p.parseImplicitExpression()
	} else if tok == token.INT || tok == token.FLOAT || tok == token.STRING {
		p.errorf("Go integer, float, and string literals must be grouped by parens")
	} else if tok == token.EOF {
		p.errorf("unexpected EOF in code parser")
	} else if tok == token.NOT || tok == token.REM || tok == token.AND || tok == token.CHAR {
		p.errorf("invalid '%s' Go token while parsing code", tok.String())
	} else {
		p.errorf("expected Pushup keyword or expression, got %q", tok.String())
	}
	return e
}

func (p *codeParser) parseIfStmt() *nodeIf {
	var stmt nodeIf
	start := p.peek().pos
	maxread := start
	lastlit := p.peek().String()
loop:
	for {
		switch p.peek().tok {
		case token.EOF:
			p.errorf("premature end of conditional in IF statement")
		case token.LBRACE:
			// conditional expression has been scanned
			break loop
			// TODO(paulsmith): add cases for tokens that are illegal in an expression
		}
		maxread = p.peek().pos
		lastlit = p.peek().String()
		p.advance()
	}
	n := (p.file.Offset(maxread) - p.file.Offset(start)) + len(lastlit)
	offset := p.baseOffset + p.file.Offset(start)
	stmt.cond = new(nodeGoStrExpr)
	stmt.cond.pos.start = offset
	stmt.cond.pos.end = offset + n
	stmt.cond.expr = p.sourceFrom(start)[:n]
	if _, err := goparser.ParseExpr(stmt.cond.expr); err != nil {
		p.errorf("parsing Go expression in IF conditional: %w", err)
	}
	stmt.then = p.parseStmtBlock()
	// parse ^else clause
	if p.peek().tok == token.XOR {
		p.advance()
		if p.peek().tok == token.ELSE {
			p.advance()
			if p.peek().tok == token.XOR {
				p.advance()
				if p.peek().tok == token.IF {
					p.advance()
					stmt.alt = p.parseIfStmt()
				} else {
					p.errorf("expected `if' after transition character, got %v", p.peek().String())
				}
			} else {
				stmt.alt = p.parseStmtBlock()
			}
		}
	}
	return &stmt
}

func (p *codeParser) parseForStmt() *nodeFor {
	var stmt nodeFor
	start := p.peek().pos
loop:
	for {
		switch p.peek().tok {
		case token.EOF:
			p.errorf("premature end of clause in FOR statement")
		case token.LBRACE:
			break loop
		default:
			p.advance()
		}
	}
	n := (p.file.Offset(p.prev().pos) - p.file.Offset(start)) + len(p.prev().String())
	offset := p.baseOffset + p.file.Offset(start)
	stmt.clause = new(nodeGoCode)
	stmt.clause.pos.start = offset
	stmt.clause.pos.end = offset + n
	stmt.clause.code = p.sourceFrom(start)[:n]
	stmt.block = p.parseStmtBlock()
	return &stmt
}

func (p *codeParser) parseStmtBlock() *nodeBlock {
	// we are sitting on the opening '{' token here
	if p.peek().tok != token.LBRACE {
		p.errorf("expected '{', got '%s'", p.peek().String())
	}
	p.advance()
	var block *nodeBlock
	switch p.peek().tok {
	// check for a transition, i.e., stay in code parser
	case token.XOR:
		p.advance()
		code := p.parseCode()
		if p.peek().tok == token.SEMICOLON {
			p.advance()
		}
		block = &nodeBlock{nodes: []node{code}}
	case token.EOF:
		p.errorf("premature end of block in IF statement")
	default:
		block = p.transition()
	}
	// we should be at the closing '}' token here
	if p.peek().tok != token.RBRACE {
		if p.peek().tok == token.LSS {
			p.errorf("there must be a single HTML element inside a Go code block, try wrapping them in a <text></text> pseudo-element")
		} else {
			p.errorf("expected closing '}', got %v", p.peek())
		}
	}
	p.advance()
	return block
}

// TODO(paulsmith): extract a common function with parseCodeKeyword
func (p *codeParser) parseHandlerKeyword() *nodeGoCode {
	result := &nodeGoCode{context: handlerGoCode}
	// we are one token past the 'handler' keyword
	if p.peek().tok != token.LBRACE {
		p.errorf("expected '{', got '%s'", p.peek().tok)
	}
	depth := 1
	p.advance()
	result.pos.start = p.parser.offset
	start := p.peek().pos
loop:
	for {
		switch p.peek().tok {
		case token.LBRACE:
			depth++
		case token.RBRACE:
			depth--
			if depth == 0 {
				break loop
			}
		}
		p.advance()
	}
	n := (p.file.Offset(p.prev().pos) - p.file.Offset(start)) + len(p.prev().String())
	if p.peek().tok != token.RBRACE {
		panic("")
	}
	p.advance()
	result.code = p.sourceFrom(start)[:n]
	result.pos.end = result.pos.start + n
	return result
}

func (p *codeParser) parseSectionKeyword() *nodeSection {
	// enter function one past the "section" IDENT token
	// FIXME(paulsmith): we are currently requiring that the name of the
	// partial be a valid Go identifier, but there is no reason that need be
	// the case. perhaps a string is better here.
	if p.peek().tok != token.IDENT {
		p.errorf("expected IDENT, got %s", p.peek().tok.String())
	}
	result := &nodeSection{name: p.peek().lit}
	result.pos.start = p.parser.offset
	p.advance()
	result.pos.end = p.parser.offset
	result.block = p.parseStmtBlock()
	return result
}

func (p *codeParser) parsePartialKeyword() *nodePartial {
	// enter function one past the "partial" IDENT token
	// FIXME(paulsmith): we are currently requiring that the name of the
	// partial be a valid Go identifier, but there is no reason that need be
	// the case. authors may want to, for example, have a name that is contains
	// dashes or other punctuation (which would need to be URL-escaped for the
	// routing of partials). perhaps a string is better here.
	if p.peek().tok != token.IDENT {
		p.errorf("expected IDENT, got %s", p.peek().tok.String())
	}
	result := &nodePartial{name: p.peek().lit}
	result.pos.start = p.parser.offset
	p.advance()
	result.pos.end = p.parser.offset
	result.block = p.parseStmtBlock()
	return result
}

func (p *codeParser) parseCodeBlock() *nodeGoCode {
	result := &nodeGoCode{context: inlineGoCode}
	if p.peek().tok != token.LBRACE {
		p.errorf("expected '{', got '%s'", p.peek().tok)
	}
	depth := 1
	p.advance()
	result.pos.start = p.parser.offset
	start := p.peek().pos
	maxread := start
	lastlit := p.peek().String()
loop:
	for {
		switch p.peek().tok {
		case token.LBRACE:
			depth++
		case token.RBRACE:
			depth--
			if depth == 0 {
				break loop
			}
		case token.EOF:
			p.errorf("unexpected EOF parsing code block")
		}
		maxread = p.peek().pos
		lastlit = p.peek().String()
		p.advance()
	}
	n := (p.file.Offset(maxread) - p.file.Offset(start)) + len(lastlit)
	if p.peek().tok != token.RBRACE {
		panic("")
	}
	p.advance()
	result.code = p.sourceFrom(start)[:n]
	result.pos.end = result.pos.start + n
	return result
}

func (p *codeParser) parseImportKeyword() *nodeImport {
	/*
		examples:
		TRANS_SYMimport   "lib/math"         math.Sin
		TRANS_SYMimport m "lib/math"         m.Sin
		TRANS_SYMimport . "lib/math"         Sin
	*/
	e := new(nodeImport)
	// we are one token past the 'import' keyword
	switch p.peek().tok {
	case token.STRING:
		e.decl.path = p.peek().lit
		p.advance()
	case token.IDENT:
		e.decl.pkgName = p.peek().lit
		p.advance()
		if p.peek().tok != token.STRING {
			p.errorf("expected string, got %s", p.peek().tok)
		}
		e.decl.path = p.peek().lit
	case token.PERIOD:
		e.decl.pkgName = "."
		p.advance()
		if p.peek().tok != token.STRING {
			p.errorf("expected string, got %s", p.peek().tok)
		}
		e.decl.path = p.peek().lit
	default:
		p.errorf("unexpected token type after "+transSymStr+"import: %s", p.peek().tok)
	}
	return e
}

func (p *codeParser) parseExplicitExpression() *nodeGoStrExpr {
	// one token past the opening '('
	result := new(nodeGoStrExpr)
	result.pos.start = p.parser.offset
	start := p.peek().pos
	maxread := start
	lastlit := p.peek().String()
	depth := 1
loop:
	for {
		switch p.peek().tok {
		case token.LPAREN:
			depth++
		case token.RPAREN:
			depth--
			if depth == 0 {
				break loop
			}
		case token.ILLEGAL:
			p.errorf("illegal Go token %q", p.peek().String())
		case token.EOF:
			p.errorf("unterminated explicit expression, expected closing ')'")
		}
		maxread = p.peek().pos
		lastlit = p.peek().String()
		p.advance()
	}
	n := (p.file.Offset(maxread) - p.file.Offset(start)) + len(lastlit)
	if p.peek().tok != token.RPAREN {
		panic(fmt.Sprintf("internal error: expected ')', got '%s'", p.peek().String()))
	}
	_ = p.sync()
	result.expr = p.sourceFrom(start)[:n]
	result.pos.end = result.pos.start + n
	if _, err := goparser.ParseExpr(result.expr); err != nil {
		p.errorf("illegal Go expression: %w", err)
	}
	return result
}

// offset is the current global offset into the original source code of the Pushup file.
//
//nolint:unused
func (p *codeParser) offset() int {
	return p.parser.offset
}

// tokenOffset is the global offset into the original source code for this token.
func (p *codeParser) tokenOffset(tok goToken) int {
	return p.baseOffset + p.file.Offset(tok.pos)
}

func (p *codeParser) parseImplicitExpression() *nodeGoStrExpr {
	if p.peek().tok != token.IDENT {
		panic("internal error: expected Go identifier start implicit expression")
	}
	result := new(nodeGoStrExpr)
	end := p.tokenOffset(p.peek())
	result.pos.start = end
	identLen := len(p.peek().String())
	end += identLen
	p.advance()
	if !unicode.IsSpace(rune(p.charAt(result.pos.start + identLen))) {
	Loop:
		for {
			if p.peek().tok == token.LPAREN {
				nested := 1
				end++
				p.advance()
				for {
					if p.peek().tok == token.RPAREN {
						end++
						p.advance()
						nested--
						if nested == 0 {
							goto Loop
						}
					} else if p.peek().tok == token.ILLEGAL {
						p.errorf("illegal Go token %q", p.peek().String())
					} else if p.peek().tok == token.EOF {
						p.errorf("unexpected EOF, want ')'")
					}
					end = p.tokenOffset(p.peek()) + len(p.peek().String())
					p.advance()
				}
			} else if p.peek().tok == token.LBRACK { // '['
				nested := 1
				end++
				p.advance()
				for {
					if p.peek().tok == token.RBRACK {
						end++
						p.advance()
						nested--
						if nested == 0 {
							goto Loop
						}
					} else if p.peek().tok == token.ILLEGAL {
						p.errorf("illegal Go token %q", p.peek().String())
					} else if p.peek().tok == token.EOF {
						p.errorf("unexpected EOF, want ')'")
					}
					end = p.tokenOffset(p.peek()) + len(p.peek().String())
					p.advance()
				}
			} else if p.peek().tok == token.PERIOD {
				last := p.peek().pos
				p.advance()
				end++
				// if space between period and next token, regardless of what
				// it is, need to break. the period needs to be pushed back on
				// to the stream to be parsed.
				if p.peek().pos-last > 1 || p.peek().tok != token.IDENT {
					p.backup()
					end--
					break
				}
				adv := len(p.peek().String())
				end += adv
				if unicode.IsSpace(rune(p.charAt(end))) {
					// done
					p.advance()
					break
				}
				p.advance()
			} else {
				break
			}
		}
	}
	result.expr = p.sourceRange(result.pos.start, end)
	result.pos.end = end
	if _, err := goparser.ParseExpr(result.expr); err != nil {
		p.errorf("illegal Go expression %q: %w", result.expr, err)
	}
	return result
}
