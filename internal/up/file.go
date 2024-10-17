package up

import (
	"io/fs"
	"iter"
	"path/filepath"
	"strings"
)

func Find(root string, fileType string) iter.Seq[string] {
	return func(yield func(string) bool) {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				if (fileType == "go" && strings.HasSuffix(path, ".up.go")) ||
					(fileType == "up" && filepath.Ext(path) == ".up") {
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
