package command

import (
	"context"
	"fmt"

	"github.com/adhocteam/pushup/internal/devserver"
)

func DevServer(root string, port int) error {
	devServer := devserver.New(root, port)
	if err := devServer.Run(context.Background()); err != nil {
		return fmt.Errorf("running dev server: %w", err)
	}
	return nil
}
