# Getting started

To make a new Pushup app, first install the main Pushup executable.

### Installing Pushup

#### Prerequisites

-   go 1.18 or later

Make sure the directory where the go tool installs executables is in your
`$PATH`. It is `$(go env GOPATH)/bin`. You can check if this is the case with:

```shell
echo $PATH | grep $(go env GOPATH)/bin > /dev/null && echo yes || echo no
```

#### Install via official release

Binary executables for multiple platforms are available for download on the
[project releases page](https://github.com/adhocteam/pushup/releases).

#### Install via git

```shell
git clone https://github.com/adhocteam/pushup
cd pushup
make
```

#### Install via `go install`

Make sure you have Go installed (at least version 1.18), and type:

```shell
go install github.com/adhocteam/pushup@latest
```

#### Install via `homebrew`

Coming soon.

#### Install via Linux package managers

Coming soon.

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
.up files and auxiliary project Go code, and a go.mod file.

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

### Listing routes

You can print a list of all the routes in your Pushup project with the command
`pushup routes`.

The lefthand column is the URL route, where any dynamic path segments are
denoted with a leading `:` colon. The righthand column is the corresponding
Pushup page.

For example:

```shell
$ pushup routes
/about                    about.up
/album/:id                album/$id.up
/album/delete/:id         album/delete/$id.up
/album/edit/:id           album/edit/$id.up
/album/new                album/new.up
/album/                   album/index.up
/dyn/:name                dyn/$name.up
/htmx/active-search       htmx/active-search.up
/htmx/click-to-load       htmx/click-to-load.up
/htmx/                    htmx/index.up
/htmx/value-select        htmx/value-select.up
/                         index.up
/projects/:pid/users/:uid projects/$pid/users/$uid.up
```
