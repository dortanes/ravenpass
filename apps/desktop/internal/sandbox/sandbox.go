// Package sandbox keeps the folders the owner picks reachable from inside the App Sandbox, where access to a picked
// folder ends with the process unless a security-scoped bookmark to it is kept.
package sandbox

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"slices"
	"sync"

	"github.com/dortanes/ravenpass/packages/app/privatefile"
	"github.com/dortanes/ravenpass/packages/app/savedfile"
)

var (
	ErrUnavailable = errors.New("this build keeps no bookmarks")
	ErrBookmark    = errors.New("the folder could not be bookmarked")
	ErrResolve     = errors.New("the bookmark no longer names a folder")
	ErrWrite       = errors.New("the file could not be written")
)

const (
	// maxFolders bounds the record; the oldest picks go first.
	maxFolders = 64
	maxRecord  = 1 << 20
)

// Contained reports whether the process is signed to run in the App Sandbox.
func Contained() bool { return contained() }

type bookmarks interface {
	create(folder string) ([]byte, error)
	// resolve starts access to the folder the bookmark names; stale asks for a new bookmark.
	resolve(bookmark []byte) (folder string, stale bool, err error)
	replace(path string, data []byte) error
}

type kept struct {
	Folder   string `json:"folder"`
	Bookmark []byte `json:"bookmark"`
}

// Folders records a bookmark to each folder the owner picked and reopens them all when it opens.
type Folders struct {
	mu    sync.Mutex
	path  string
	marks bookmarks
	kept  []kept
}

// Open reads the record at path and starts access to every folder in it; a bookmark that no longer resolves stays
// recorded, so a disk that is only absent now is reached again later. An unreadable record leaves every folder to be
// picked again, which the vault's unreachable screen asks for.
func Open(path string) *Folders { return open(path, system{}) }

func open(path string, marks bookmarks) *Folders {
	f := &Folders{path: path, marks: marks}
	data, err := privatefile.Read(path, maxRecord)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			slog.Warn("the kept folders could not be read", "err", err)
		}
		return f
	}
	if err := json.Unmarshal(data, &f.kept); err != nil {
		slog.Warn("the kept folders could not be read", "err", err)
		f.kept = nil
		return f
	}
	refreshed := false
	for i, entry := range f.kept {
		folder, stale, err := marks.resolve(entry.Bookmark)
		if err != nil {
			slog.Warn("a kept folder could not be reached", "err", err)
			continue
		}
		if !stale {
			continue
		}
		if fresh, err := marks.create(folder); err == nil {
			f.kept[i].Bookmark = fresh
			refreshed = true
		}
	}
	if refreshed {
		if err := f.save(f.kept); err != nil {
			slog.Warn("the refreshed folders could not be kept", "err", err)
		}
	}
	return f
}

// Keep bookmarks a folder the owner just picked, while the picker's access still holds.
func (f *Folders) Keep(folder string) error {
	bookmark, err := f.marks.create(folder)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	next := slices.DeleteFunc(slices.Clone(f.kept), func(entry kept) bool { return entry.Folder == folder })
	next = append(next, kept{Folder: folder, Bookmark: bookmark})
	if len(next) > maxFolders {
		next = next[len(next)-maxFolders:]
	}
	if err := f.save(next); err != nil {
		return err
	}
	f.kept = next
	return nil
}

// Replace writes data over a file the owner picked in a save panel and reads it back.
func (f *Folders) Replace(path string, data []byte, limit int64) error {
	if len(data) == 0 || int64(len(data)) > limit {
		return savedfile.ErrInvalidSize
	}
	if err := f.marks.replace(path, data); err != nil {
		return err
	}
	readback, err := savedfile.Read(path, limit)
	defer clear(readback)
	if err != nil || !bytes.Equal(readback, data) {
		return savedfile.ErrUnverified
	}
	return nil
}

func (f *Folders) save(entries []kept) error {
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	return privatefile.Write(f.path, data)
}
