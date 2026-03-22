package image

import (
	"fmt"
	"os"
	"sync"

	"github.com/timzifer/lux/draw"
)

// Store manages loaded images and their CPU-side pixel data.
// Images are identified by draw.ImageID handles.
// Thread-safe for concurrent access.
type Store struct {
	mu      sync.RWMutex
	nextID  draw.ImageID
	entries map[draw.ImageID]*Entry
}

// Entry holds the decoded pixel data for a single image.
type Entry struct {
	ID     draw.ImageID
	Width  int
	Height int
	RGBA   []byte // Pre-multiplied RGBA8 pixel data, row-major, ready for GPU upload
	dirty  bool   // true = needs GPU upload
}

// NewStore creates an empty image store.
func NewStore() *Store {
	return &Store{
		nextID:  1,
		entries: make(map[draw.ImageID]*Entry),
	}
}

// LoadFromBytes decodes image data (PNG or JPEG, auto-detected via magic bytes)
// and returns a handle for use with Canvas.DrawImage.
func (s *Store) LoadFromBytes(data []byte) (draw.ImageID, error) {
	w, h, rgba, err := decodeRaster(data)
	if err != nil {
		return 0, fmt.Errorf("image: load: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID
	s.nextID++
	s.entries[id] = &Entry{
		ID:     id,
		Width:  w,
		Height: h,
		RGBA:   rgba,
		dirty:  true,
	}
	return id, nil
}

// LoadFromFile reads a file and decodes it as a raster image.
func (s *Store) LoadFromFile(path string) (draw.ImageID, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("image: read %s: %w", path, err)
	}
	return s.LoadFromBytes(data)
}

// LoadFromRGBA creates an image entry from raw RGBA pixel data.
func (s *Store) LoadFromRGBA(width, height int, rgba []byte) (draw.ImageID, error) {
	expected := width * height * 4
	if len(rgba) != expected {
		return 0, fmt.Errorf("image: RGBA data length %d, expected %d (%dx%dx4)", len(rgba), expected, width, height)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID
	s.nextID++
	buf := make([]byte, len(rgba))
	copy(buf, rgba)
	s.entries[id] = &Entry{
		ID:     id,
		Width:  width,
		Height: height,
		RGBA:   buf,
		dirty:  true,
	}
	return id, nil
}

// Get returns the entry for the given ID, or nil if not found.
func (s *Store) Get(id draw.ImageID) *Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entries[id]
}

// Size returns the dimensions of the image, or (0, 0) if not found.
func (s *Store) Size(id draw.ImageID) (w, h int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e := s.entries[id]
	if e == nil {
		return 0, 0
	}
	return e.Width, e.Height
}

// Remove deletes an image from the store. The caller must also remove
// any corresponding GPU texture via the renderer.
func (s *Store) Remove(id draw.ImageID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, id)
}

// Dirty reports whether the image needs GPU upload.
func (e *Entry) Dirty() bool { return e.dirty }

// ClearDirty marks the image as uploaded.
func (e *Entry) ClearDirty() { e.dirty = false }
