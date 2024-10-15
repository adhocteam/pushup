package internal

import (
	"fmt"
	"os"
)

func Build(rootDir string) error {
	fmt.Fprintf(os.Stderr, "\x1b[33mBuilding: %s\x1b[0m\n", rootDir)
	return nil
}
