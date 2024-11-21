package parser

import "testing"

func TestParser(t *testing.T) {
	src := []byte(`<!DOCTYPE html>
<title>Pushup</title>
^{
    name := "world"
}
<p class="greeting">Hello, ^name!</p>
<ul>
^for i := range 3 {
    <li id="item-^i">
        ^(i * i)
    </li>
}
</ul>
`)

	p := New(src)
	p.Parse()
}
