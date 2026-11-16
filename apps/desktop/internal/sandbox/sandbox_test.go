package sandbox

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dortanes/ravenpass/packages/app/savedfile"
)

// fakeMarks bookmarks a folder as its name; resolving reports what the test set for it.
type fakeMarks struct {
	resolved   []string
	stale      map[string]bool
	missing    map[string]bool
	failCreate bool
	written    map[string][]byte
}

func (m *fakeMarks) create(folder string) ([]byte, error) {
	if m.failCreate {
		return nil, ErrBookmark
	}
	return []byte(folder + "#" + fmt.Sprint(len(m.resolved))), nil
}

func (m *fakeMarks) resolve(bookmark []byte) (string, bool, error) {
	folder, _, _ := bytes.Cut(bookmark, []byte("#"))
	if m.missing[string(folder)] {
		return "", false, ErrResolve
	}
	m.resolved = append(m.resolved, string(folder))
	return string(folder), m.stale[string(folder)], nil
}

func (m *fakeMarks) replace(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

func TestKeptFoldersAreReopenedAtTheNextLaunch(t *testing.T) {
	record := filepath.Join(t.TempDir(), "folders.json")
	first := open(record, &fakeMarks{})
	for _, folder := range []string{"/a", "/b", "/a"} {
		if err := first.Keep(folder); err != nil {
			t.Fatal(err)
		}
	}
	marks := &fakeMarks{}
	open(record, marks)
	if fmt.Sprint(marks.resolved) != "[/b /a]" {
		t.Fatalf("reopened %v, want each folder once, the latest pick last", marks.resolved)
	}
}

func TestAMissingOrUnreadableRecordOpensNothing(t *testing.T) {
	directory := t.TempDir()
	marks := &fakeMarks{}
	open(filepath.Join(directory, "folders.json"), marks)
	broken := filepath.Join(directory, "broken.json")
	if err := os.WriteFile(broken, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	reopened := open(broken, marks)
	if len(marks.resolved) != 0 || len(reopened.kept) != 0 {
		t.Fatalf("resolved %v from no usable record", marks.resolved)
	}
	if err := reopened.Keep("/a"); err != nil {
		t.Fatalf("a folder picked after an unreadable record was not kept: %v", err)
	}
}

func TestAFolderThatCannotBeReachedStaysKept(t *testing.T) {
	record := filepath.Join(t.TempDir(), "folders.json")
	if err := open(record, &fakeMarks{}).Keep("/drive"); err != nil {
		t.Fatal(err)
	}
	open(record, &fakeMarks{missing: map[string]bool{"/drive": true}})
	marks := &fakeMarks{}
	open(record, marks)
	if fmt.Sprint(marks.resolved) != "[/drive]" {
		t.Fatalf("resolved %v once the drive is back", marks.resolved)
	}
}

func TestAStaleBookmarkIsReplaced(t *testing.T) {
	record := filepath.Join(t.TempDir(), "folders.json")
	if err := open(record, &fakeMarks{}).Keep("/a"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	open(record, &fakeMarks{stale: map[string]bool{"/a": true}, resolved: []string{"earlier"}})
	after, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before, after) {
		t.Fatal("the stale bookmark was kept")
	}
}

func TestAFolderThatCannotBeBookmarkedIsRefused(t *testing.T) {
	record := filepath.Join(t.TempDir(), "folders.json")
	if err := open(record, &fakeMarks{failCreate: true}).Keep("/a"); !errors.Is(err, ErrBookmark) {
		t.Fatalf("Keep = %v, want ErrBookmark", err)
	}
	if _, err := os.Stat(record); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a refused folder was recorded")
	}
}

func TestTheRecordKeepsTheLatestPicksOnly(t *testing.T) {
	record := filepath.Join(t.TempDir(), "folders.json")
	folders := open(record, &fakeMarks{})
	for i := range maxFolders + 3 {
		if err := folders.Keep(fmt.Sprintf("/%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	marks := &fakeMarks{}
	open(record, marks)
	if len(marks.resolved) != maxFolders || marks.resolved[0] != "/3" {
		t.Fatalf("reopened %d folders from %s", len(marks.resolved), marks.resolved[0])
	}
}

func TestASavedFileIsReadBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "copy.rpv")
	folders := open(filepath.Join(t.TempDir(), "folders.json"), &fakeMarks{})
	if err := folders.Replace(path, []byte("vault"), 16); err != nil {
		t.Fatal(err)
	}
	if err := folders.Replace(path, nil, 16); !errors.Is(err, savedfile.ErrInvalidSize) {
		t.Fatalf("an empty file = %v, want ErrInvalidSize", err)
	}
	if err := folders.Replace(path, []byte("too large"), 4); !errors.Is(err, savedfile.ErrInvalidSize) {
		t.Fatalf("an oversized file = %v, want ErrInvalidSize", err)
	}
}
