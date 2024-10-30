package codegen

import "github.com/adhocteam/pushup/internal/source"

// outputOp represents a single output operation
type outputOp struct {
	kind    outputOpKind
	content string
	span    source.Span
	expr    string // dynamic content (Go expressions)
}

type outputOpKind int

const (
	opStatic outputOpKind = iota
	opDynamic
	opFlush // forces a write, can't be coalesced
	opGoCode
	opControlStart
	opControlElse
	opControlEnd
)

type outputCollector struct {
	ops []outputOp
}

func (c *outputCollector) add(op outputOp) {
	c.ops = append(c.ops, op)
}

// optimize coalesces adjacent static operations
func (c *outputCollector) optimize() []outputOp {
	if len(c.ops) == 0 {
		return nil
	}

	result := make([]outputOp, 0, len(c.ops))
	current := c.ops[0]

	for i := 1; i < len(c.ops); i++ {
		next := c.ops[i]

		if current.kind == opStatic && next.kind == opStatic {
			current.content += next.content
			current.span.End = next.span.End
		} else {
			result = append(result, current)
			current = next
		}
	}

	result = append(result, current)
	return result
}
