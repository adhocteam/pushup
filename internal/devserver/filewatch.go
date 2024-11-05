package devserver

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type fileWatcher struct {
	watcher   *fsnotify.Watcher
	debouncer *time.Timer
}

func newFileWatcher(path string) (*fileWatcher, error) {
	var err error
	w := &fileWatcher{}
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating new fsnotify watcher: %w", err)
	}
	w.debouncer = time.NewTimer(100 * time.Millisecond)
	err = w.add(path)
	return w, err
}

func (w *fileWatcher) add(root string) error {
	slog.Debug("adding to file watcher", "path", root)
	err := fs.WalkDir(os.DirFS(root), ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			path = filepath.Join(root, path)
			slog.Debug("adding to file watcher", "path", path)
			if err := w.watcher.Add(path); err != nil {
				return fmt.Errorf("adding %q to file watcher: %w", path, err)
			}
		}
		return nil
	})
	return err
}
