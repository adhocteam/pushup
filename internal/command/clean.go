package command

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/adhocteam/pushup/internal/up"
)

func Clean(root string) error {
	logger := slog.Default()
	logger.Info("Cleaning", "root", root)
	for file := range up.Find(root, "go") {
		logger.Info("Removing generated file", "file", file)
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("removing %q: %w", file, err)
		}
	}
	return nil
}
