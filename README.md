# Pushup - a modern web framework for Go

![workflow status](https://github.com/AdHocRandD/pushup/actions/workflows/go.yml/badge.svg)

Pushup is an experimental new project that is exploring the viability of a new
approach to web frameworks in Go.

Pushup seeks to make building page-oriented, server-side web apps using Go
easy. It embraces the server, while acknowledging the reality of modern web
apps and their improvements to UI interactivity over previous generations.

## What is Pushup?

There are three main aspects to Pushup:

1. An opinionated project/app directory structure that enables **file-based
   routing**,
1. A **lightweight syntax** alternative to traditional web framework templates
   that combines Go code for control flow and imperative, view-controller-like
   code with HTML markup, and
1. A **compiler** that parses that syntax and generates pure Go code,
   building standalone web apps on top of the Go stdlib `net/http` package.

The syntax looks like this:

```pushup

^import "time"

^{
    title := "Hello, from Pushup!"
}

<h1>^title</h1>

<p>The time is now ^(time.Now().String()).</p>

^if time.Now().Weekday() == time.Friday {
    <p>It's Friday! Enjoy the start to your weekend.</p>
} else {
    <p>Have a great day, we're glad you're here.</p>
}

```

You would then place this code in a file somewhere in your `app/pages`
directory, like `hello.pushup`. The `.pushup` extension is important and tells
the compiler that it is a Pushup page. Once you build and run your Pushup app,
that page is automatically mapped to the URL path `/hello`.

## Getting started

To make a new Pushup app, first install the main Pushup executable.

Because Pushup does not (yet) have a public repository, you need to [create a
personal access token][token] on GitHub, and configure your ~/.netrc file.

Make sure you have Go installed (at least version 1.18), and type:

```shell
GOPRIVATE=github.com/AdHocRandD/pushup go install github.com/AdHocRandD/pushup@latest
```

The `GOPRIVATE` environment variable is necessary to tell the go tool not to
try to get the module from one of the central module services, but directly
from GitHub.

The command will install the `pushup` executable locally. Make sure the
directory where the go tool installs to is in your `$PATH`, which is `$(go env GOPATH)/bin`. You can check if this is the case with:

```shell
echo $PATH | grep $(go env GOPATH)/bin > /dev/null && echo yes || echo no
```

### Creating a new Pushup project

To create a new Pushup project, use the `pushup new` command.

```shell
pushup new
```

Without any additional arguments, it will attempt to create a scaffolded new
project in the current directory. However, the directory must be completely
empty, or the command will abort. To simulataneously make a new directory
and generate a scaffolded project, pass a relative path as argument:

```shell
pushup new myproject
```

The scaffolded new project directory consists of a directory structure for
.pushup files and auxiliary project Go code, and a go.mod file.

Change to the new project directory if necessary, then do a `pushup run`,
which compiles the Pushup project to Go code, builds the app, and starts up
the server.

```shell
pushup run
```

If all goes well, you should see a message on the terminal that the Pushup app
is running and listening on a port:

```
↑↑ Pushup ready and listening on 0.0.0.0:8080 ↑↑
```

By default it listens on port 8080, but with the `-port` or `-unix-socket`
flags you can pick your own listener.

Open [http://localhost:8080/](http://localhost:8080/) in your browser to see
the default layout and a welcome index page.

## Example demo app

See the [example](./example) directory for a demo Pushup app that demonstrates
many of the concepts in Pushup and implements a few small common patterns like
some HTMX examples and a simple CRUD app.

Click on "view source" at the bottom of any page in the example app to see the
source of the .pushup page for that route, including the source of the "view
source" .pushup page itself. This is a good way to see how to write Pushup
syntax.

## Go modules and Pushup projects

Pushup treats projects as their own self-contained Go module. The build
process assumes this is the case by default. But it is possible to include a
Pushup project as part of a parent Go module. See the the `-module` option to
`pushup new`, and the `-build-pkg` option to the `pushup run` command.

## Project directory structure

Pushup projects have a particular directory structure that the compiler expects
before building. The most minimal Pushup project would look like:

```
app
├── layouts
├── pages
│   └── index.pushup
└── pkg
go.mod
```

## Pages

Pushup pages are the main units in Pushup. They are a combination of logic and
content. It may be helpful to think of them as both the controller and the view
in a MVC-like system, but colocated together in the same file.

They are also the basis of file-based routing: the name of the Pushup file,
minus the .pushup extension, is mapped to the portion of the URL path for
routing.

## Layouts

Layouts are HTML templates that used in common across multiple pages. They are
just HTML, with the exception of the required Pushup directive `^contents`,
which indicates where individual page content will be insert when rendered.

## File-based routing

Docs TKTK

## Pushup syntax

### How it works

Pushup is a mix of a new syntax consisting of Pushup directives and keywords,
Go code, and HTML markup.

Parsing a .pushup file always starts out in HTML mode, so you can just put
plain HTML in a file and that's a valid Pushup page.

When the parser encounters a '^' character (caret, ASCII 0x5e) while in
HTML mode, it switches to parsing Pushup syntax, which consists of simple
directives, control flow statements, block delimiters, and Go expressions. It
then switches to the Go code parser. Once it detects the end of the directive
or statement, it switches back to HTML mode, and parsing continues in a
similar fashion.

Pushup uses the tokenizers from the [go/scanner][scannerpkg] and
[golang.org/x/net/html][htmlpkg] packages, so it should be able to handle
any valid syntax from either language.

### Directives

#### `^import`

Docs TKTK

#### `^layout`

Docs TKTK

#### `^contents`

Docs TKTK

### Code blocks

#### `^{`

Docs TKTK

#### `^handler`

Docs TKTK

### Control flow statements

#### `^if`

Docs TKTK

#### `^for`

Docs TKTK

### Expressions

#### Simple expressions

Simple Go expressions can be written with just `^` followed by the expression.
"Simple" means variable names, and dotted field name access of structs.

Example:

```pushup
^{ name := "Paul" }
<p>Hello, ^name!</p>
```

Renders:

```html
<p>Hello, Paul!</p>
```

Notice that the parser stops on the "!" because it knows it is not part of the
variable name.

Example:

```pushup
<p>The URL path: ^req.URL.Path</p>
```

Renders:

```html
<p>The URL path: /foo/bar</p>
```

#### Explicit expressions

Explicit expressions are written with `^` and the followed by any valid Go
expression surrounded by parentheses.

This is a good way to make function/method calls.

Example:

```pushup
^import "strings"
<p>^(strings.Repeat("Hello", 3))</p>
```

Renders:

```html
<p>HelloHelloHello</p>
```

[token]: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token
[scannerpkg]: https://pkg.go.dev/go/scanner#Scanner
[htmlpkg]: https://pkg.go.dev/golang.org/x/net/html#Tokenizer
