package up

import (
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"strings"
)

type Kind int

const (
	Page Kind = iota
	Component
)

type Ext int

const (
	DotUp Ext = iota
	DotUpDotGo
)

func (e Ext) String() string {
	switch e {
	case DotUp:
		return ".up"
	case DotUpDotGo:
		return ".up.go"
	default:
		panic("unimplemented")
	}
}

func (e Ext) Has(file string) bool {
	switch e {
	case DotUp:
		return filepath.Ext(file) == e.String()
	case DotUpDotGo:
		return strings.HasSuffix(file, e.String())
	default:
		panic("unimplemented")
	}
}

// File represents a Pushup file (.up extension)
type File struct {
	Path    string // Full path to the file
	RelPath string // Path relative to the project root
	Kind    Kind   // Type of Pushup file - page or component
	Content []byte // File content
}

func Find(root string, ext Ext) iter.Seq[string] {
	return func(yield func(string) bool) {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				if ext.Has(path) {
					if !yield(path) {
						return filepath.SkipAll
					}
				}
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
	}
}

func IsPage(file string) bool {
	// TODO: the value that "pages" is currently hard-coded to represent may be
	// set by configuration - see routeForPage in analyzer/analyzer.go
	return strings.HasPrefix(file, "pages"+string(os.PathSeparator))
}
