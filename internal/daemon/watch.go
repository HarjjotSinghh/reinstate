package daemon

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher delivers a Change whenever a file under one of the roots is
// written, created, renamed, or removed. Files the agents write beside
// their sessions but that never belong to a session (lock files, editor
// temp files) are ignored by suffix.
type Watcher struct {
	Events <-chan Change
	// Mode is "fsnotify" or "poll".
	Mode string
	stop func()
}

// Stop releases the watcher. Events is closed afterwards.
func (w *Watcher) Stop() {
	if w != nil && w.stop != nil {
		w.stop()
	}
}

// ignoredSuffixes are written constantly by agents and editors and never
// change a session's content.
var ignoredSuffixes = []string{".lock", ".tmp", ".swp", ".swx", "~", ".db-shm"}

func ignored(path string) bool {
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") && !strings.HasSuffix(base, ".jsonl") && !strings.HasSuffix(base, ".json") {
		return true
	}
	for _, suffix := range ignoredSuffixes {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	return false
}

// PollEvery is the fallback scan interval when fsnotify is unavailable.
const PollEvery = 15 * time.Second

// Watch starts watching roots (recursively). It prefers fsnotify and falls
// back to polling when the OS watcher cannot be created or a root cannot
// be added (for instance when the file-descriptor limit is reached on a
// kqueue host). Missing roots are polled for until they appear.
func Watch(ctx context.Context, roots []string, logger *log.Logger) *Watcher {
	if logger == nil {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	if w, err := watchNotify(ctx, roots, logger); err == nil {
		return w
	} else {
		logger.Printf("fsnotify unavailable (%v); polling every %s", err, PollEvery)
	}
	return watchPoll(ctx, roots, PollEvery, nil, logger)
}

// WatchPolling always scans on a timer, for hosts where the OS watcher
// misbehaves (network home directories, some containers).
func WatchPolling(ctx context.Context, roots []string, logger *log.Logger) *Watcher {
	if logger == nil {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	return watchPoll(ctx, roots, PollEvery, nil, logger)
}

func watchNotify(ctx context.Context, roots []string, logger *log.Logger) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	added := 0
	for _, root := range roots {
		if err := addTree(fsw, root); err != nil {
			_ = fsw.Close()
			return nil, err
		}
		if _, statErr := os.Stat(root); statErr == nil {
			added++
		}
	}
	if added == 0 && len(roots) > 0 {
		// Nothing exists yet; polling notices when a root appears.
		_ = fsw.Close()
		return nil, errors.New("no watch root exists yet")
	}
	out := make(chan Change, 64)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer close(out)
		defer func() { _ = fsw.Close() }()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-fsw.Events:
				if !ok {
					return
				}
				if ev.Has(fsnotify.Create) {
					if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
						// A new project or session directory: watch it too.
						if err := addTree(fsw, ev.Name); err != nil {
							logger.Printf("watch %s: %v", ev.Name, err)
						}
					}
				}
				if ignored(ev.Name) {
					continue
				}
				select {
				case out <- Change{Path: ev.Name}:
				default:
					// The loop coalesces anyway; a dropped event is covered
					// by the one already queued.
				}
			case err, ok := <-fsw.Errors:
				if !ok {
					return
				}
				logger.Printf("watch: %v", err)
			}
		}
	}()
	return &Watcher{Events: out, Mode: "fsnotify", stop: cancel}, nil
}

// addTree watches root and every directory below it. A missing root is
// not an error: it is watched once its parent reports it created.
func addTree(fsw *fsnotify.Watcher, root string) error {
	info, err := os.Stat(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fsw.Add(filepath.Dir(root))
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable subtree: skip, keep the rest
		}
		if !d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") && path != root {
			return filepath.SkipDir
		}
		return fsw.Add(path)
	})
}

// fileStamp is what polling compares between scans.
type fileStamp struct {
	size    int64
	modTime time.Time
}

// watchPoll scans the roots every interval and reports paths whose size or
// modification time changed. tick overrides the interval source in tests.
func watchPoll(ctx context.Context, roots []string, every time.Duration, tick <-chan time.Time, logger *log.Logger) *Watcher {
	out := make(chan Change, 64)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer close(out)
		previous := scan(roots)
		var ticker *time.Ticker
		if tick == nil {
			ticker = time.NewTicker(every)
			defer ticker.Stop()
			tick = ticker.C
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick:
			}
			current := scan(roots)
			for path, stamp := range current {
				if old, ok := previous[path]; ok && old == stamp {
					continue
				}
				select {
				case out <- Change{Path: path}:
				default:
				}
			}
			for path := range previous {
				if _, ok := current[path]; !ok {
					select {
					case out <- Change{Path: path}:
					default:
					}
				}
			}
			previous = current
		}
	}()
	return &Watcher{Events: out, Mode: "poll", stop: cancel}
}

func scan(roots []string) map[string]fileStamp {
	stamps := map[string]fileStamp{}
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") && path != root {
					return filepath.SkipDir
				}
				return nil
			}
			if ignored(path) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			stamps[path] = fileStamp{size: info.Size(), modTime: info.ModTime()}
			return nil
		})
	}
	return stamps
}
