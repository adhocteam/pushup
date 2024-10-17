package command

import (
	"fmt"
	"log/slog"

	"github.com/adhocteam/pushup/internal/compile"
	"github.com/adhocteam/pushup/internal/up"
)

func Build(root string) error {
	// TODO: take a logger optionally from the caller
	logger := slog.Default()
	logger.Info("Building", "root", root)

	for file := range up.Find(root, "up") {
		result, err := compile.Compile(file)
		if err != nil {
			return fmt.Errorf("compiling %q: %w", file, err)
		}
		logger.Info("Compiled", "source", file, "pkg", result.PkgName)
	}

	return nil
}
