package command

import (
	"fmt"
	"io/fs"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func Clean(root string) error {
	logger := slog.Default()
	logger.Info("Cleaning", "root", root)
	for file := range findUpGoFiles(root) {
		logger.Info("Removing generated file", "file", file)
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("removing %q: %w", file, err)
		}
	}
	return nil
}

func findUpGoFiles(root string) iter.Seq[string] {
	return func(yield func(string) bool) {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() && strings.HasSuffix(path, ".up.go") {
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
