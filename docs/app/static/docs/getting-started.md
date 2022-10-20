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

The `GOPRIVATE` environment variable is necessary to tell the go tool not to
try to get the module from one of the central module services, but directly
from GitHub.

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


