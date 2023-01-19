package main

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

// implement the HTML5 spec lexing algorithm for open tags. this is necessary
// because in order to switch safely between HTML and Go code parsing in
// the Pushup parser, we need to precisely track the read character position
// internally to start (or self-closing) tags, because the transition character
// may appear inside HTML attributes. the golang.org/x/net/html tokenizer
// that forms the basis of the Pushup HTML parser, while it precisely tracks
// character position for token types indirectly via its Raw() method, does
// not help us inside a start (or self-closing) tag, including attributes. So,
// yes, we're doing extra work, re-tokenizing the tag. But it's not expensive
// work (just open and self-closing tags, not the whole doc) and there's not an
// alternative with golang.org/x/net/html.
//
// we start in the data state
//
// https://html.spec.whatwg.org/multipage/parsing.html#tag-open-state

func scanAttrs(openTag string) (attrs []*attr, err error) {
	// maintain some invariants, we are not a general-purpose HTML
	// tokenizer/parser, we are just parsing open tags.
	if len(openTag) == 0 {
		return []*attr{}, nil
	}
	if ch := openTag[0]; ch != '<' {
		return nil, openTagScanError(fmt.Sprintf("expected '<', got '%c'", ch))
	}

	l := newOpenTagLexer(openTag)
	// panic mode error handling
	defer func() {
		if e := recover(); e != nil {
			if le, ok := e.(openTagScanError); ok {
				attrs = nil
				err = le
			} else {
				panic(e)
			}
		}
	}()
	attrs = l.scan()
	return
}

type openTagScanError string

func (e openTagScanError) Error() string {
	return string(e)
}

type openTagLexer struct {
	raw         string
	pos         int
	state       openTagLexState
	returnState openTagLexState
	charRefBuf  bytes.Buffer

	nextInputChar        int
	provideCurrInputChar bool

	attrs    []*attrBuilder
	currAttr *attrBuilder
}

func newOpenTagLexer(source string) *openTagLexer {
	l := new(openTagLexer)
	l.raw = source
	l.state = openTagLexData
	return l
}

func (l *openTagLexer) consumeNextInputChar() int {
	var result int
	if l.provideCurrInputChar {
		l.provideCurrInputChar = false
		result = l.nextInputChar
	} else {
		if len(l.raw) > 0 {
			l.decodeNextInputChar()
		} else {
			l.nextInputChar = eof
		}
		result = l.nextInputChar
	}
	return result
}

func (l *openTagLexer) decodeNextInputChar() {
	l.nextInputChar = int(l.raw[0])
	l.raw = l.raw[1:]
	l.pos++
}

type attrBuilder struct {
	name  bufPos
	value bufPos
}

type bufPos struct {
	*bytes.Buffer
	start pos
}

type attr struct {
	name  stringPos
	value stringPos
}

type stringPos struct {
	string
	start pos
}

type pos int

type openTagLexState int

// NOTE(paulsmith): we only consider a subset of the HTML5 tokenization states,
// because we rely on the golang.org/x/net/html tokenizer to produce a valid
// start tag token that we scan here for attributes. so certain states are not
// considered, or are considered assertion errors if they would ordinarily be
// entered into.
const (
	openTagLexData openTagLexState = iota
	openTagLexTagOpen
	openTagLexTagName
	openTagLexBeforeAttrName
	openTagLexAttrName
	openTagLexAfterAttrName
	openTagLexBeforeAttrVal
	openTagLexAttrValDoubleQuote
	openTagLexAttrValSingleQuote
	openTagLexAttrValUnquoted
	openTagLexAfterAttrValQuoted
	openTagLexCharRef
	openTagLexNamedCharRef
	openTagLexNumericCharRef
	openTagLexSelfClosingStartTag
	openTagLexAmbiguousAmpersand
)

func (s openTagLexState) String() string {
	switch s {
	case openTagLexData:
		return "Data"
	case openTagLexTagOpen:
		return "TagOpen"
	case openTagLexTagName:
		return "TagName"
	case openTagLexBeforeAttrName:
		return "BeforeAttrName"
	case openTagLexAttrName:
		return "AttrName"
	case openTagLexAfterAttrName:
		return "AfterAttrName"
	case openTagLexBeforeAttrVal:
		return "BeforeAttrVal"
	case openTagLexAttrValDoubleQuote:
		return "AttrValDoubleQuote"
	case openTagLexAttrValSingleQuote:
		return "AttrValSingleQuote"
	case openTagLexAttrValUnquoted:
		return "AttrValUnquoted"
	case openTagLexAfterAttrValQuoted:
		return "AfterAttrValQuoted"
	case openTagLexCharRef:
		return "CharRef"
	case openTagLexNamedCharRef:
		return "NamedCharRef"
	case openTagLexNumericCharRef:
		return "NumericCharRef"
	case openTagLexSelfClosingStartTag:
		return "SelfClosingStartTag"
	case openTagLexAmbiguousAmpersand:
		return "AmbiguousAmpersand"
	default:
		panic("unexpected tag state")
	}
}

const eof = -1

func (l *openTagLexer) scan() []*attr {
loop:
	for {
		switch l.state {
		// 13.2.5.1 Data state
		// https://html.spec.whatwg.org/multipage/parsing.html#data-state
		case openTagLexData:
			ch := l.consumeNextInputChar()
			switch ch {
			case '&':
				l.returnState = openTagLexData
				l.switchState(openTagLexCharRef)
			case '<':
				l.switchState(openTagLexTagOpen)
			case 0:
				l.specParseError("unexpected-null-character")
			default:
				l.errorf("found '%c' in data state, expected '<'", ch)
			}
		// 13.2.5.6 Tag open state
		// https://html.spec.whatwg.org/multipage/parsing.html#tag-open-state
		case openTagLexTagOpen:
			ch := l.consumeNextInputChar()
			switch {
			case ch == '!':
				l.errorf("input '%c' switch to markup declaration open state", ch)
			case ch == '/':
				l.errorf("input '%c' switch to end tag open state", ch)
			case isASCIIAlpha(ch):
				l.reconsumeIn(openTagLexTagName)
			case ch == '?':
				l.errorf("input '%c' parse error", ch)
			case ch == eof:
				l.errorf("eof before tag name parse error")
			default:
				l.errorf("found '%c' in tag open state", ch)
			}
		// 13.2.5.8 Tag name state
		// https://html.spec.whatwg.org/multipage/parsing.html#tag-name-state
		case openTagLexTagName:
			ch := l.consumeNextInputChar()
			switch {
			case ch == '\t' || ch == '\n' || ch == '\f' || ch == ' ':
				l.switchState(openTagLexBeforeAttrName)
			case ch == '/':
				l.switchState(openTagLexSelfClosingStartTag)
			case ch == '>':
				break loop
			case isASCIIUpper(ch):
				// append lowercase version of current input char to current tag token's tag name
				// not needed, we know the tag name from the golang.org/x/net/html tokenizer
			case ch == 0:
				l.errorf("found null in tag name state")
			case ch == eof:
				l.errorf("found eof in tag name state")
			default:
				// append current input char to current tag token's tag name
			}
		// 13.2.5.32 Before attribute name state
		// https://html.spec.whatwg.org/multipage/parsing.html#before-attribute-name-state
		case openTagLexBeforeAttrName:
			ch := l.consumeNextInputChar()
			switch ch {
			case '\t', '\n', '\f', ' ':
				// ignore
			case '/', '>', eof:
				l.reconsumeIn(openTagLexAfterAttrName)
			case '=':
				l.errorf("found '%c' in before attribute name state", ch)
			default:
				l.newAttr()
				l.reconsumeIn(openTagLexAttrName)
			}
		// 13.2.5.33 Attribute name state
		// https://html.spec.whatwg.org/multipage/parsing.html#attribute-name-state
		case openTagLexAttrName:
			ch := l.consumeNextInputChar()
			switch {
			case ch == '\t' || ch == '\n' || ch == '\f' || ch == ' ' || ch == '/' || ch == '>' || ch == eof:
				l.reconsumeIn(openTagLexAfterAttrName)
				l.cmpAttrName()
			case ch == '=':
				l.switchState(openTagLexBeforeAttrVal)
				l.cmpAttrName()
			case isASCIIUpper(ch):
				// append lowercase version of current input character to current attr's name
				l.appendCurrName(int(unicode.ToLower(rune(ch))))
			case ch == 0:
				l.specParseError("unexpected-null-character")
			case ch == '"' || ch == '\'' || ch == '<':
				l.specParseError("unexpected-character-in-attribute-name")
				// append current input character to current attribute's name
				l.appendCurrName(ch)
			default:
				// append current input character to current attribute's name
				l.appendCurrName(ch)
			}
		// 13.2.5.34 After attribute name state
		// https://html.spec.whatwg.org/multipage/parsing.html#after-attribute-name-state
		case openTagLexAfterAttrName:
			ch := l.consumeNextInputChar()
			switch ch {
			case '\t', '\n', '\f', ' ':
				// ignore
			case '/':
				l.switchState(openTagLexSelfClosingStartTag)
			case '=':
				l.switchState(openTagLexBeforeAttrVal)
			case '>':
				break loop
			case eof:
				l.specParseError("eof-in-tag")
			default:
				l.newAttr()
				l.reconsumeIn(openTagLexAttrName)
			}
		// 13.2.5.35 Before attribute value state
		// https://html.spec.whatwg.org/multipage/parsing.html#before-attribute-value-state
		case openTagLexBeforeAttrVal:
			ch := l.consumeNextInputChar()
			switch ch {
			case '\t', '\n', '\f', ' ':
				// ignore
			case '"':
				l.switchState(openTagLexAttrValDoubleQuote)
			case '\'':
				l.switchState(openTagLexAttrValSingleQuote)
			case '>':
				l.specParseError("missing-attribute-value")
				break loop
			default:
				l.reconsumeIn(openTagLexAttrValUnquoted)
			}
		// 13.2.5.36 Attribute value (double-quoted) state
		// https://html.spec.whatwg.org/multipage/parsing.html#attribute-value-(double-quoted)-state
		case openTagLexAttrValDoubleQuote:
			ch := l.consumeNextInputChar()
			switch ch {
			case '"':
				l.switchState(openTagLexAfterAttrValQuoted)
			case '&':
				l.returnState = openTagLexAttrValDoubleQuote
				l.switchState(openTagLexCharRef)
			case 0:
				l.errorf("found null in attribute value (double-quoted) state")
			case eof:
				l.errorf("found EOF in tag")
			default:
				l.appendCurrVal(ch)
			}
		// 13.2.5.37 Attribute value (single-quoted) state
		// https://html.spec.whatwg.org/multipage/parsing.html#attribute-value-(single-quoted)-state
		case openTagLexAttrValSingleQuote:
			ch := l.consumeNextInputChar()
			switch ch {
			case '"':
				l.switchState(openTagLexAfterAttrValQuoted)
			case '&':
				l.returnState = openTagLexAttrValSingleQuote
				l.switchState(openTagLexCharRef)
			case 0:
				l.errorf("found null in attribute value (single-quoted) state")
			case eof:
				l.errorf("found EOF in tag")
			default:
				l.appendCurrVal(ch)
			}
		// 13.2.5.38 Attribute value (unquoted) state
		// https://html.spec.whatwg.org/multipage/parsing.html#attribute-value-(unquoted)-state
		case openTagLexAttrValUnquoted:
			ch := l.consumeNextInputChar()
			switch ch {
			case '\t', '\n', '\f', ' ':
				l.switchState(openTagLexBeforeAttrName)
			case '&':
				l.returnState = openTagLexAttrValUnquoted
				l.switchState(openTagLexCharRef)
			case '>':
				break loop
			case 0:
				l.errorf("found null in attribute value (unquoted) state")
			case '"', '\'', '<', '=', '`':
				l.specParseError("unexpected-null-character")
				l.appendCurrVal(ch)
			case eof:
				l.errorf("found EOF in tag")
			default:
				l.appendCurrVal(ch)
			}
		// 13.2.5.39 After attribute value (quoted) state
		// https://html.spec.whatwg.org/multipage/parsing.html#after-attribute-value-(quoted)-state
		case openTagLexAfterAttrValQuoted:
			ch := l.consumeNextInputChar()
			switch ch {
			case '\t', '\n', '\f', ' ':
				l.switchState(openTagLexBeforeAttrName)
			case '/':
				l.switchState(openTagLexSelfClosingStartTag)
			case '>':
				break loop
			case eof:
				l.errorf("found EOF in tag")
			default:
				l.specParseError("missing-whitespace-between-attributes")
				l.reconsumeIn(openTagLexBeforeAttrName)
			}
		// 13.2.5.72 Character reference state
		// https://html.spec.whatwg.org/multipage/parsing.html#character-reference-state
		case openTagLexCharRef:
			l.charRefBuf = bytes.Buffer{}
			l.charRefBuf.WriteByte('&')
			ch := l.consumeNextInputChar()
			switch {
			case isASCIIAlphanum(ch):
				l.reconsumeIn(openTagLexNamedCharRef)
			case ch == '#':
				l.charRefBuf.WriteByte(byte(ch))
				l.switchState(openTagLexNumericCharRef)
			default:
				l.flushCharRef()
				l.reconsumeIn(l.returnState)
			}
		// 13.2.5.40 Self-closing start tag state
		// https://html.spec.whatwg.org/multipage/parsing.html#self-closing-start-tag-state
		case openTagLexSelfClosingStartTag:
			ch := l.consumeNextInputChar()
			switch ch {
			case '>':
				break loop
			case eof:
				l.errorf("found EOF in tag")
			default:
				l.specParseError("unexpected-solidus-in-tag")
				l.reconsumeIn(openTagLexBeforeAttrName)
			}
		// 13.2.5.73 Named character reference state
		// https://html.spec.whatwg.org/multipage/parsing.html#named-character-reference-state
		case openTagLexNamedCharRef:
			var ch int
			for ch = l.consumeNextInputChar(); isASCIIAlphanum(ch); ch = l.consumeNextInputChar() {
				l.charRefBuf.WriteByte(byte(ch))
			}
			ident := l.charRefBuf.String()
			ref, ok := namedCharRefs[ident]
			if ok {
				lastCharMatched := ident[len(ident)-1]
				if l.consumedAsPartOfAttr() && lastCharMatched != ';' && (ch == '=' || isASCIIAlphanum(ch)) {
					l.flushCharRef()
					l.switchState(l.returnState)
				} else {
					if ch != ';' {
						l.specParseError("missing-semicolon-after-character-reference")
					}
					l.charRefBuf.Reset()
					l.charRefBuf.WriteString(ref)
					l.flushCharRef()
					l.switchState(l.returnState)
				}
			} else {
				l.flushCharRef()
				l.switchState(openTagLexAmbiguousAmpersand)
			}

		default:
			l.errorf("unimplemented parse state " + l.state.String())
		}
	}

	result := make([]*attr, len(l.attrs))
	for i := range l.attrs {
		builder := l.attrs[i]
		attr := &attr{
			name: stringPos{
				builder.name.String(),
				builder.name.start,
			},
			value: stringPos{
				builder.value.String(),
				builder.value.start,
			},
		}
		result[i] = attr
	}
	return result
}

func (l *openTagLexer) consumedAsPartOfAttr() bool {
	if l.returnState == openTagLexAttrValDoubleQuote ||
		l.returnState == openTagLexAttrValSingleQuote ||
		l.returnState == openTagLexAttrValUnquoted {
		return true
	} else {
		return false
	}
}

func (l *openTagLexer) flushCharRef() {
	if l.currAttr.value.start == 0 {
		l.currAttr.value.start = pos(l.pos - 1)
	}
	l.currAttr.value.Write(l.charRefBuf.Bytes())
}

func (l *openTagLexer) newAttr() {
	a := &attrBuilder{
		name: bufPos{
			Buffer: new(bytes.Buffer),
		},
		value: bufPos{
			Buffer: new(bytes.Buffer),
		},
	}
	l.attrs = append(l.attrs, a)
	l.currAttr = a
}

func (l *openTagLexer) appendCurrName(ch int) {
	if l.currAttr.name.start == 0 {
		l.currAttr.name.start = pos(l.pos - 1)
	}
	l.currAttr.name.WriteByte(byte(ch))
}

func (l *openTagLexer) appendCurrVal(ch int) {
	if l.currAttr.value.start == 0 {
		l.currAttr.value.start = pos(l.pos - 1)
	}
	l.currAttr.value.WriteByte(byte(ch))
}

func (l *openTagLexer) errorf(format string, args ...any) {
	panic(openTagScanError(fmt.Sprintf(format, args...)))
}

// specParseError panics with a openTagScanError type as the panic value but
// is specifically meant for signaling the known parse errors per the HTML5
// parsing specification we expect to encounter with this limited version
// that just focuses on tokenizing open tags.
func (l *openTagLexer) specParseError(code string) {
	switch code {
	case "eof-in-tag":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-eof-in-tag
		// This error occurs if the parser encounters the end of the input
		// stream in a start tag or an end tag (e.g., <div id=). Such a tag is
		// ignored.
	case "unexpected-character-in-attribute-name":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-unexpected-character-in-attribute-name
	case "duplicate-attribute":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-duplicate-attribute
		// This error occurs if the parser encounters an attribute in a tag that
		// already has an attribute with the same name. The parser ignores all such
		// duplicate occurrences of the attribute.
	case "missing-attribute-value":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-missing-attribute-value
		// This error occurs if the parser encounters a U+003E (>) code point where
		// an attribute value is expected (e.g., <div id=>). The parser treats the
		// attribute as having an empty value.
	case "missing-whitespace-between-attributes":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-missing-whitespace-between-attributes
	case "unexpected-solidus-in-tag":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-unexpected-solidus-in-tag
		// This error occurs if the parser encounters a U+002F (/) code point
		// that is not a part of a quoted attribute value and not immediately
		// followed by a U+003E (>) code point in a tag (e.g., <div / id="foo">).
		// In this case the parser behaves as if it encountered ASCII whitespace.
	case "unexpected-null-character":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-unexpected-null-character
		// This error occurs if the parser encounters a U+0000 NULL code point
		// in the input stream in certain positions. In general, such code
		// points are either ignored or, for security reasons, replaced with a
		// U+FFFD REPLACEMENT CHARACTER.
	case "missing-semicolon-after-character-reference":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-missing-semicolon-after-character-reference
		// This error occurs if the parser encounters a character reference
		// that is not terminated by a U+003B (;) code point. Usually the
		// parser behaves as if character reference is terminated by the U+003B
		// (;) code point; however, there are some ambiguous cases in which the
		// parser includes subsequent code points in the character reference.
	default:
		panic("unexpected parse error code " + code)
	}
	panic(openTagScanError(strings.ReplaceAll(code, "-", " ")))
}

func (l *openTagLexer) reconsumeIn(state openTagLexState) {
	l.provideCurrInputChar = true
	l.switchState(state)
}

func (l *openTagLexer) exitingState(state openTagLexState) {
}

func (l *openTagLexer) enteringState(state openTagLexState) {
}

func (l *openTagLexer) switchState(state openTagLexState) {
	l.exitingState(l.state)
	l.enteringState(state)
	l.state = state
}

func (l *openTagLexer) cmpAttrName() {
	for i := range l.attrs[:len(l.attrs)-1] {
		if l.currAttr.name == l.attrs[i].name {
			l.specParseError("duplicate-attribute")
			// we're supposed to ignore this per the spec but the
			// golang.org/x/net/html tokenizer doesn't, so we follow that
			// TODO(paulsmith): open issue with ^^
		}
	}
}

func isASCIIUpper(ch int) bool {
	if ch >= 'A' && ch <= 'Z' {
		return true
	}
	return false
}

func isASCIIAlpha(ch int) bool {
	if isASCIIUpper(ch) || (ch >= 'a' && ch <= 'z') {
		return true
	}
	return false
}

func isASCIIAlphanum(ch int) bool {
	if isASCIIAlpha(ch) || (ch >= '0' && ch <= '9') {
		return true
	}
	return false
}
