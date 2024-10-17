package command

import (
	"fmt"
	"io/fs"
	"iter"
	"log/slog"
	"path/filepath"

	"github.com/adhocteam/pushup/internal/compile"
)

func Build(root string) error {
	// TODO: take a logger optionally from the caller
	logger := slog.Default()
	logger.Info("Building", "root", root)

	for file := range findUpFiles(root) {
		result, err := compile.Compile(file)
		if err != nil {
			return fmt.Errorf("compiling %q: %w", file, err)
		}
		logger.Info("Compiled", "source", file, "pkg", result.PkgName)
	}

	return nil
}

func findUpFiles(root string) iter.Seq[string] {
	return func(yield func(string) bool) {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() && filepath.Ext(path) == ".up" {
				if !yield(path) {
					return filepath.SkipAll
				}
			}

			return nil
		})

		if err != nil {
			panic(err)
		}
	}
}
