# Syntax

Pushup is a mix of a new syntax consisting of Pushup directives and keywords,
Go code, and HTML markup.

### How it works

Parsing a `.up` file always starts out in HTML mode, so you can just put
plain HTML in a file and that's a valid Pushup page.

When the parser encounters a '^' character (caret, ASCII 0x5e) while in
HTML mode, it switches to parsing Pushup syntax, which consists of simple
directives, control flow statements, block delimiters, and Go expressions. It
then switches to the Go code parser. Once it detects the end of the directive,
statement, or expression, it switches back to HTML mode, and parsing continues
in a similar fashion.

Pushup uses the tokenizers from the [go/scanner][scannerpkg] and
[golang.org/x/net/html][htmlpkg] packages, so it should be able to handle
any valid syntax from either language.

[scannerpkg]: https://pkg.go.dev/go/scanner#Scanner
[htmlpkg]: https://pkg.go.dev/golang.org/x/net/html#Tokenizer

### Directives

#### `^import`

Use `^import` to import a Go package into the current Pushup page. The syntax
for `^import` is the same as a regular [Go import declaration](https://go.dev/ref/spec#Import_declarations)

Example:

```pushup
^import "strings"
^import "strconv"
```

```pushup
^import . "strings"
```

### Go code blocks

#### `^{`

To include statements of Go in a Pushup page, type `^{` followed by your
Go code, terminating with a closing `}`.

The scope of a `^{ ... }` in the compiled Go code is equal to its surrounding
markup, so you can define a variable and immediately use it:

```pushup
^{
	name := "world"
}
<h1>Hello, ^name!</h1>
```

Because the Pushup parser is only looking for a balanced closing `}`, blocks
can be one-liners:

```pushup
^{ name := "world"; greeting := "Hello" }
<h1>^greeting, ^name!</h1>
```

A Pushup page can have zero or many `^{ ... }` blocks.

#### `^handler`

A handler is similar to `^{ ... }`. The difference is that there may be at most
one handler per page, and it is run prior to any other code or markup on the
page.

A handler is the appropriate place to do "controller"-like (in the MVC sense)
actions, such as HTTP redirects and errors. In other words, any control flow
based on the nature of the request, for example, redirecting after a successful
POST to create a new object in a CRUD operation.

Example:

```pushup
^handler {
    if req.Method == "POST" && formValid(req) {
		if err := createObjectFromForm(req.Form); err == nil {
			return http.Redirect(w, req, "/success/", http.StatusSeeOther)
			return nil
		} else {
			// error handling
			...
	}
	...
}
...
```

Note that handlers (and all Pushup code) run in a method on a receiver that
implements Pushup's `Responder` interface, which is

```go
interface Responder {
	Respond(http.ResponseWriter, *http.Request) error
}
```

To exit from a page early in a handler (i.e., prior to any normal content being
rendered), return from the method with a nil (for success) or an error (which
will in general respond with HTTP 500 to the client).

### Control flow statements

#### `^if`

`^if` takes a boolean Go expression and a block to conditionally render.

Example:

```pushup
^if query := req.FormValue("query"); query != "" {
	<p>Query: ^query</p>
}
```

#### `^for`

`^for` takes a Go "for" statement condition, clause, or range, and a block,
and repeatedly executes the block.

Example:

```pushup
^for i := 0; i < 10; i++ {
	<p>Number ^i</p>
}
```

### Expressions

#### Simple expressions

Simple Go expressions can be written with just `^` followed by the expression.
"Simple" means:

-   variable names (eg., `^x`)
-   dotted field name access of structs (eg., `^account.name`)
-   function and method calls (eg., `^strings.Repeat("x", 3)`)
-   index expressions (eg., `a[x]`)

Example:

```pushup
^{ name := "Paul" }
<p>Hello, ^name!</p>
```

Outputs:

```html
<p>Hello, Paul!</p>
```

Notice that the parser stops on the "!" because it knows it is not part of a
Go variable name.

Example:

```pushup
<p>The URL path: ^req.URL.Path</p>
```

Outputs:

```html
<p>The URL path: /foo/bar</p>
```

Example:

```pushup
^import "strings"
<p>^strings.Repeat("Hello", 3)</p>
```

Outputs:

```html
<p>HelloHelloHello</p>
```

#### Explicit expressions

Explicit expressions are written with `^` and followed by any valid Go
expression grouped by parentheses.

Example:

```pushup
^{ numPeople := 4 }
<p>With ^numPeople people there are ^(numPeople * 2) hands</p>
```

Outputs:

```html
<p>With 4 people there are 8 hands</p>
```

### Inline partials

#### `^partial`

Pushup pages can declare and define inline partials with the `^partial`
keyword.

```pushup
...
<section>
    <p>Elements</p>
    ^partial list {
            <ul>
                <li>Ag</li>
                <li>Na</li>
                <li>C</li>
            </ul>
    }
</section>
...
```

A request to the page containing the initial partial will render normally,
as if the block where not wrapped in `^partial list {` ... `}`.

A request to the page with the name of the partial appended to the URL path
will respond with just the content scoped by the partial block.

For example, if the page above had the route `/elements/`, then a request to
`/elements/list` would output:

```html
<ul>
    <li>Ag</li>
    <li>Na</li>
    <li>C</li>
</ul>
```

Inline partials can nest arbitrarily deep.

```pushup
...
^partial leagues {
    <p>Leagues</p>
    ^partial teams {
        <p>Teams</p>
        ^partial players {
            <p>Players</p>
        }
    }
}
...
```

To request a nested partial, make sure the URL path is preceded by
each containing partial's name and a forward slash, for example,
`/sports/leagues/teams/players`.


