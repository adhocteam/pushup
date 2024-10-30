package command

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/adhocteam/pushup/internal/up"
)

func Clean(root string) error {
	slog.Info("Cleaning", "root", root)
	for file := range up.Find(root, up.DotUpDotGo) {
		slog.Info("Removing generated file", "file", file)
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("removing %q: %w", file, err)
		}
	}
	// TODO: delete generated main.go file? How to identify?
	return nil
}
