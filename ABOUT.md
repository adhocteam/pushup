Let's party like it's 1999 and make a new server-side-first web framework.

This document by Paul Smith

# Pushup - a modern web framework for the Go programming language

Let's make Go the best full-stack web language. Let's recapture the best
of past web development eras, like PHP's ease of deployment, updated with
today's tech and user expectations, like great client-side interactivity,
avoiding the things that haven't been so good, like complexity and bloat,
combined with everything that makes Go great, like performance, compile-time
type safety, and fast builds.

It's a server-side web app framework first and foremost, let's just get that
clear from the get-go.

## Key features

-   Pages - individual files that combine Go code and HTML markup which
    compile down to Go packages, sort of like components
-   File-based routing by default, with dynamic parameters
-   Server-sent events integration
-   Enhanced hypertext/hypermedia (think [htmx](https://htmx.org/))
-   Compiled - single static deployable artifact contains the entire app

## Table stakes functionality

-   Form submission & struct validation
-   a11y - good defaults
-   i18n/l10n - hooks for easy translation
-   Safe HTML generation, escaping by default
-   Web security protection - CSRF, XSS
-   Health checks
-   Metrics and monitoring hooks

## Aspirational features

-   Blend the distinction between server and client code
-   WASM (run (some) Go code in the browser)
-   SQLite by default
-   sqlc-like data access layer

## Also includes nice-to-haves

-   Live-reloading dev environment
-   Great debugging and visualization tools

## Trends

A number of things are driving this.

-   Return to server-side web dev
-   Feeling of too much JavaScript, especially build complexity and laggy UX
-   Frustration with SPAs, especially bloat and UI complexity
-   Greater expectations of client interactivity
-   Mobile app competition to the web
-   Maturation of SQLite as a lightweight server-side alternative RDBMS

## Precedent

A few projects come to mind, providing inspiration, ideas and implementations
to lift from.

-   Phoenix (Elixir)
-   Razor (C# ASP.NET)
-   Hyperfiddle/Photon (Clojure)
-   Remix (JavaScript)

## Similar Go projects

Before embarking on this project, I researched what other Go web frameworks
are out there and if they might already address some of the needs and concerns
I have that motivated me. To be frank, the sheer usefulness of Go's standard
`net/http` package has meant that I haven't familiarized myself much with
alternative or higher-order frameworks. But a few caught my eye:

-   [Vugu](https://www.vugu.org/) - Vugu looks good and seems to work based on
    my kicking the tires if you want the experience of writing Go code as
    components for frontend browser behavior. But it seems fairly limited to
    that SPA-like style of interaction. Pushup wants to move things to the
    server.
-   [Vecty](https://github.com/hexops/vecty) - I like that it compiles to WASM,
    but like Vugu, it's solidly targeting the client, not the server.
-   [gox](https://github.com/8byt/gox) - this project extends the Go compiler
    to allow for a JSX-like syntax to be embedded directly in Go source code. I
    like this ambition and the specific JSX-in-Go idea. Unfortunately, it seems
    to be merely a target for Vecty.
-   [Ego](https://github.com/benbjohnson/ego) - In many ways this is the
    closest in spirit to Pushup, compiling templates down to Go, and just
    using Go for template control flow and logic. I want to take this idea
    further, and also make it much easier by adding file-based routing.

## How?

-   [HATEOAS](https://htmx.org/essays/hateoas/) - HTML-over-the-wire,
    server-side state centralization
-   Blend Go control flow and imperative "view controller" code with HTML
    markup
-   Components compile down to Go structs in separate package, wired together
    at the top-level for routing and HTTP serving

## To anticipate some questions:

-   What about `html/template`? It's a great template package, but for what I
    want to achieve, which are server-side components, I think the simplest
    and best thing is to use Go code for the dynamic bits and inline it with
    HTML. While I'm building this out and exploring the design, I think it's
    best if I can focus on the things I can control and not be distracted
    by making a different package fit what I'm trying to do. (It's entirely
    possible though that I abandon this approach and make a wrapper over
    `html/template` with a pre-processing step, I'm reserving the right
    to change my mind.) I'm inspired by what C# is doing in Blazor and the
    .razor file format/syntax here. The key is to have a compilation step that
    produces Go code from a project directory layout.
-   What about `net/http`? This will all be built on `net/http` ultimately
    for the runtime, pages compile down to methods on a type that implements
    the ServeHTTP interface.

## Inspirations

-   [Ben Hoyt's article on routing in Go](https://benhoyt.com/writings/go-routing/)
