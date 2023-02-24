module github.com/adhocteam/pushup/example

go 1.18

require github.com/mattn/go-sqlite3 v1.14.14

require github.com/adhocteam/pushup v0.0.0

require (
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	golang.org/x/mod v0.7.0 // indirect
	golang.org/x/net v0.0.0-20220706163947-c90051bbdb60 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
)

replace github.com/adhocteam/pushup v0.0.0 => ../
