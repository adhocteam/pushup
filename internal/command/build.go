package command

import (
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"

	"github.com/adhocteam/pushup/internal/compile"
)

func Build(root string) error {
	// TODO: take a logger optionally from the caller
	logger := slog.Default()
	logger.Info("Building", "root", root)

	var upfiles []string
	err := findUpFiles(root, func(path string) error {
		logger.Debug("Found .up file", "path", path)
		upfiles = append(upfiles, path)
		return nil
	})
	if err != nil {
		return fmt.Errorf("finding .up files: %w", err)
	}

	for _, file := range upfiles {
		result, err := compile.Compile(file)
		if err != nil {
			return fmt.Errorf("compiling %q: %w", file, err)
		}
		logger.Info("Compiled", "source", file, "pkg", result.PkgName)
	}

	return nil
}

func findUpFiles(root string, callback func(string) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && filepath.Ext(path) == ".up" {
			return callback(path)
		}

		return nil
	})
}
