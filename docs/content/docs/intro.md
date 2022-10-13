---
title: "Introduction to Pushup"
date: 2022-10-12T10:36:38-05:00
draft: false
---

Pushup is an experimental new project that is exploring the viability of a new
approach to web frameworks in Go.

Pushup seeks to make building page-oriented, server-side web apps using Go
easy. It embraces the server, while acknowledging the reality of modern web
apps and their improvements to UI interactivity over previous generations.

## What is Pushup?

Pushup is a program that compiles projects developed with the Pushup markup
language into standalone web app servers.

There are three main aspects to Pushup:

1. A **file format (.up files)** with an opinionated app directory structure that enables **file-based routing**,
1. A **lightweight markup** alternative to traditional web framework templates
   that combines Go code for control flow and imperative, view-controller-like
   code with HTML markup, and
1. A **compiler** that parses that markup and generates pure Go code,
   building standalone web apps on top of the Go stdlib `net/http` package.

### Example Pushup app directory structure

```
/path/to/mypushupapp
├── layouts
│   └── default.up
├── pages
│   └── index.up
├── pkg
│   └── app.go
└── static
    ├── app.css
    └── htmx.min.js
```

### Pages in Pushup

The core object in Pushup is the "page": a file with the `.up` extension that
is a mix of HTML, Go code, and a lightweight markup language that glues them
together. Pushup pages participate in URL routing by virtue of their path in
the filesystem. Pushup pages are compiled into pure Go which is then built
along with a thin runtime into a standalone web app server (which is all
`net/http` under the hood).

The main proposition motivating Pushup is that the page is the right level of
abstraction for most kinds of server-side web apps.

The syntax of the Pushup markup language looks like this:

```pushup

^import "time"

^{
    title := "Hello, from Pushup!"
}

<h1>^title</h1>

<p>The time is now ^time.Now().String().</p>

^if time.Now().Weekday() == time.Friday {
    <p>It's Friday! Enjoy the start to your weekend.</p>
} ^else {
    <p>Have a great day, we're glad you're here.</p>
}

```

You would then place this code in a file somewhere in your `app/pages`
directory, like `hello.up`. The `.up` extension is important and tells
the compiler that it is a Pushup page. Once you build and run your Pushup app,
that page is automatically mapped to the URL path `/hello`.

