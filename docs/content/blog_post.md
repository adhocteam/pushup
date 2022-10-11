Introducing Pushup

Pushup is a new page-oriented web framework for Go. What does a "page-oriented"
web framework mean? The central concept in Pushup is a page, which is a file
with the `.up` extension that combines HTML and Go, and automatically
configures a route based on its pathname.

Pushup compiles .up files to .go files, and with a small runtime API constructs
web app server as a single static binary executable.

Pushup is a server-side framework, but it makes better client-side
interactivity easier for server-side apps through support for enhanced
hypertext libraries, such as htmx.

Who is Pushup for?

Pushup is targeting Go developers.

Pushup is great for Go developers who have traditionally made JSON RPC-style,
JAM-stack-supporting APIs, but would like to add web pages to their app. Pushup
is designed to be standalone, but is easy to embed in a larger application,
either as a Go package/module, or proxied to by any HTTP server.

Pushup is great for any frontend developer who wants to add a dynamic server
component, because Pushup pages (`.up` files) are just HTML. To add
interactivity and dynamic elements, you add to your HTML a lightweight markup
that glues Go code in. All logic in the files is pure Go code (Go expressions
in if/else and for statements), and it is easy to emit Go strings as (safe)
HTML.

Pushup is great for "mildly dynamic" sites, the kind that might have targeted
by PHP in days of yore. They don't need the kind of highly-interactive UIs that
many JS frameworks enable today. If you aren't looking to recreate the browser
in your app, or the fidelity and fine control of a native app, or simply have
JS fatigue and want to render some pages and store some state, Pushup is trying
for that sweet spot and may be for you.

Why a new language?

Go gives the web developer many powerful building blocks built-in. The Go
standard package `html/template` is an excellent template language and library.
Pushup compiles down to a lightweight wrapper around `net/http` - it is not
reinventing the wheel here and wants to build on all the battle-hardened
benefits and performance of `net/http`.

Pushup's philosophy is what the standard library gives can often be too
low-level out of the box by themselves. For example, many projects implement
their own template file management. The ServeHTTP interface in `net/http` has
proved extremely powerful, extensible, and flexible, but implementations, by
their very nature, can live almost anyplace and be set up on any type, which
can be confusing for maintainers of apps with lots of routes. Speaking of
routes, despite the power of ServeMux, 3rd routing packages for Go abound.
Whenever enough reinventing of the wheel takes place, that's often a sign that
things may be at the wrong level of abstraction.

Pushup posits that, for many web apps, the **page** is the right level of
abstraction - a single file mapped to a single route, with all the
route-specific markup and logic contained in that single place.
