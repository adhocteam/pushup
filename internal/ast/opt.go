package ast

func Optimize(doc *Document) *Document {
	doc.Nodes = coalesceLiterals(doc.Nodes)
	return doc
}

// coalesceLiterals is an optimization that coalesces consecutive HTML literal
// nodes together by concatenating their strings together in a single node.
// TODO(paulsmith): further optimization could be had by descending in to child
// nodes, refactor this using inspect().
func coalesceLiterals(nodes []Node) []Node {
	if len(nodes) > 0 {
		n := 0
		for range nodes[:len(nodes)-1] {
			this, thisOk := nodes[n].(*NodeLiteral)
			next, nextOk := nodes[n+1].(*NodeLiteral)
			if thisOk && nextOk && len(this.Text) < 512 {
				this.Text += next.Text
				this.Span.End = next.Span.End
				nodes = append(nodes[:n+1], nodes[n+2:]...)
			} else {
				n++
			}
		}
		nodes = nodes[:n+1]
	}
	return nodes
}
