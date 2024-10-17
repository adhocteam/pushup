package command

import (
	"io/fs"
	"log/slog"
	"path/filepath"
)

func Build(root string) error {
	// TODO: take a logger optionally from the caller
	logger := slog.Default()
	logger.Info("Building", "root", root)

	findUpFiles(root, func(path string) error {
		logger.Info("Found .up file", "path", path)
		return nil
	})

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
