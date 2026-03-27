package form

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/timzifer/lux/ui/icons"
)

func TestFileIcon_Directory(t *testing.T) {
	got := fileIcon("mydir", true)
	if got != icons.Folder {
		t.Errorf("directory icon: got %q, want %q", got, icons.Folder)
	}
}

func TestFileIcon_GoFile(t *testing.T) {
	got := fileIcon("main.go", false)
	if got != icons.FileText {
		t.Errorf("go file icon: got %q, want %q", got, icons.FileText)
	}
}

func TestFileIcon_Image(t *testing.T) {
	for _, name := range []string{"photo.png", "pic.JPG", "art.svg"} {
		got := fileIcon(name, false)
		if got != icons.ImageSquare {
			t.Errorf("image icon for %q: got %q, want %q", name, got, icons.ImageSquare)
		}
	}
}

func TestFileIcon_GenericFile(t *testing.T) {
	got := fileIcon("data.bin", false)
	if got != icons.File {
		t.Errorf("generic file icon: got %q, want %q", got, icons.File)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, tc := range tests {
		got := formatSize(tc.bytes)
		if got != tc.want {
			t.Errorf("formatSize(%d) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
}

func TestMatchesFilter_AllFiles(t *testing.T) {
	f := FileFilter{Label: "All", Extensions: []string{"*"}}
	if !matchesFilter("test.xyz", f) {
		t.Error("wildcard filter should match any file")
	}
}

func TestMatchesFilter_SpecificExtension(t *testing.T) {
	f := FileFilter{Label: "Go", Extensions: []string{".go"}}
	if !matchesFilter("main.go", f) {
		t.Error("should match .go file")
	}
	if matchesFilter("main.rs", f) {
		t.Error("should not match .rs file")
	}
}

func TestMatchesFilter_CaseInsensitive(t *testing.T) {
	f := FileFilter{Label: "Images", Extensions: []string{".png", ".jpg"}}
	if !matchesFilter("photo.PNG", f) {
		t.Error("should match .PNG case-insensitively")
	}
}

func TestMatchesFilter_Empty(t *testing.T) {
	f := FileFilter{}
	if !matchesFilter("anything.txt", f) {
		t.Error("empty filter should match everything")
	}
}

func TestSortEntries_ByName(t *testing.T) {
	entries := []FileEntry{
		{Name: "charlie.txt"},
		{Name: "alpha.txt"},
		{Name: "bravo.txt"},
	}
	sortEntries(entries, SortByName, true)
	if entries[0].Name != "alpha.txt" || entries[1].Name != "bravo.txt" || entries[2].Name != "charlie.txt" {
		t.Errorf("sort by name asc: got %v %v %v", entries[0].Name, entries[1].Name, entries[2].Name)
	}
}

func TestSortEntries_ByNameDesc(t *testing.T) {
	entries := []FileEntry{
		{Name: "alpha.txt"},
		{Name: "charlie.txt"},
		{Name: "bravo.txt"},
	}
	sortEntries(entries, SortByName, false)
	if entries[0].Name != "charlie.txt" {
		t.Errorf("sort by name desc: got first=%v, want charlie.txt", entries[0].Name)
	}
}

func TestSortEntries_DirsFirst(t *testing.T) {
	entries := []FileEntry{
		{Name: "file.txt", IsDir: false},
		{Name: "dir", IsDir: true},
		{Name: "afile.txt", IsDir: false},
	}
	sortEntries(entries, SortByName, true)
	if !entries[0].IsDir {
		t.Error("directories should come first")
	}
}

func TestSortEntries_BySize(t *testing.T) {
	entries := []FileEntry{
		{Name: "big.dat", Size: 1000},
		{Name: "small.dat", Size: 10},
		{Name: "mid.dat", Size: 500},
	}
	sortEntries(entries, SortBySize, true)
	if entries[0].Size != 10 || entries[1].Size != 500 || entries[2].Size != 1000 {
		t.Errorf("sort by size asc: got %d %d %d", entries[0].Size, entries[1].Size, entries[2].Size)
	}
}

func TestSortEntries_ByModTime(t *testing.T) {
	now := time.Now()
	entries := []FileEntry{
		{Name: "new.txt", ModTime: now},
		{Name: "old.txt", ModTime: now.Add(-2 * time.Hour)},
		{Name: "mid.txt", ModTime: now.Add(-1 * time.Hour)},
	}
	sortEntries(entries, SortByModTime, true)
	if entries[0].Name != "old.txt" || entries[2].Name != "new.txt" {
		t.Errorf("sort by modtime asc: got %v first, %v last", entries[0].Name, entries[2].Name)
	}
}

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()

	// Create test files and dirs.
	os.Mkdir(filepath.Join(dir, "subdir"), 0o755)
	os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0o644)
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("secret"), 0o644)

	s := NewFilePickerState(dir)
	s.loadDir(nil, false)

	// Should have 3 entries (subdir, hello.go, readme.txt), not .hidden.
	if len(s.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d: %+v", len(s.Entries), s.Entries)
	}

	// First should be the directory.
	if !s.Entries[0].IsDir || s.Entries[0].Name != "subdir" {
		t.Errorf("first entry should be subdir directory, got %+v", s.Entries[0])
	}
}

func TestLoadDir_WithFilter(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o644)
	os.WriteFile(filepath.Join(dir, "photo.png"), []byte{0x89}, 0o644)

	s := NewFilePickerState(dir)
	filters := []FileFilter{{Label: "Go Files", Extensions: []string{".go"}}}
	s.loadDir(filters, false)

	if len(s.Entries) != 1 {
		t.Fatalf("expected 1 entry with .go filter, got %d", len(s.Entries))
	}
	if s.Entries[0].Name != "main.go" {
		t.Errorf("expected main.go, got %s", s.Entries[0].Name)
	}
}

func TestLoadDir_DirectoryOnly(t *testing.T) {
	dir := t.TempDir()

	os.Mkdir(filepath.Join(dir, "subdir"), 0o755)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0o644)

	s := NewFilePickerState(dir)
	s.loadDir(nil, true)

	if len(s.Entries) != 1 {
		t.Fatalf("directory-only: expected 1 entry, got %d", len(s.Entries))
	}
	if !s.Entries[0].IsDir {
		t.Error("expected directory entry only")
	}
}

func TestNewFilePickerState_DefaultDir(t *testing.T) {
	s := NewFilePickerState("")
	wd, _ := os.Getwd()
	if s.CurrentDir != wd {
		t.Errorf("default dir: got %q, want %q", s.CurrentDir, wd)
	}
}
