package form

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/timzifer/lux/a11y"
	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/theme"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/icons"
)

// ── Layout constants ─────────────────────────────────────────────

const (
	filePickerFieldW  = 300
	filePickerPanelW  = 480
	filePickerPanelH  = 400
	filePickerRowH    = 28
	filePickerHeaderH = 30
	filePickerPathH   = 32
	filePickerFooterH = 36
	filePickerPad     = 8
	filePickerIconW   = 24
	filePickerNameCol = 240
	filePickerSizeCol = 80
)

// ── Mode & Sort ──────────────────────────────────────────────────

// FilePickerMode selects the picker behaviour.
type FilePickerMode int

const (
	FilePickerOpen      FilePickerMode = iota // select a file to open
	FilePickerSave                            // enter a filename to save
	FilePickerDirectory                       // select a directory
)

// FileSortKey selects how entries are ordered.
type FileSortKey int

const (
	SortByName    FileSortKey = iota
	SortBySize
	SortByModTime
)

// ── FileFilter ───────────────────────────────────────────────────

// FileFilter limits visible files by extension.
type FileFilter struct {
	Label      string   // human-readable, e.g. "Go Files"
	Extensions []string // e.g. [".go"], use ["*"] for all
}

// matchesFilter returns true if name passes the filter.
func matchesFilter(name string, f FileFilter) bool {
	if len(f.Extensions) == 0 {
		return true
	}
	for _, ext := range f.Extensions {
		if ext == "*" {
			return true
		}
		if strings.EqualFold(filepath.Ext(name), ext) {
			return true
		}
	}
	return false
}

// ── FileEntry ────────────────────────────────────────────────────

// FileEntry holds metadata for one directory entry.
type FileEntry struct {
	Name    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

// sortEntries sorts a slice of FileEntry in-place.
// Directories always come first regardless of sort key.
func sortEntries(entries []FileEntry, key FileSortKey, asc bool) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		// directories first
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		var less bool
		switch key {
		case SortBySize:
			less = a.Size < b.Size
		case SortByModTime:
			less = a.ModTime.Before(b.ModTime)
		default: // SortByName
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		if !asc {
			less = !less
		}
		return less
	})
}

// ── FilePickerState ──────────────────────────────────────────────

// FilePickerState holds the mutable state for a FilePicker overlay.
type FilePickerState struct {
	Open        bool
	CurrentDir  string
	Entries     []FileEntry
	SelectedIdx int
	FilterIdx   int
	SortKey     FileSortKey
	SortAsc     bool
	SaveName    string // filename typed in save mode
	scrollY     int    // scroll offset in pixels
}

// NewFilePickerState creates a FilePickerState rooted at dir.
// If dir is empty, the current working directory is used.
func NewFilePickerState(dir string) *FilePickerState {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	s := &FilePickerState{
		CurrentDir:  dir,
		SortAsc:     true,
		SelectedIdx: -1,
	}
	return s
}

// loadDir reads the current directory and populates Entries, applying
// the given filters and the current sort key.
func (s *FilePickerState) loadDir(filters []FileFilter, dirOnly bool) {
	s.Entries = s.Entries[:0]
	s.SelectedIdx = -1
	s.scrollY = 0

	des, err := os.ReadDir(s.CurrentDir)
	if err != nil {
		return
	}

	var activeFilter FileFilter
	if len(filters) > 0 && s.FilterIdx < len(filters) {
		activeFilter = filters[s.FilterIdx]
	}

	for _, de := range des {
		name := de.Name()
		if strings.HasPrefix(name, ".") {
			continue // skip hidden
		}
		isDir := de.IsDir()
		if dirOnly && !isDir {
			continue
		}
		if !isDir && activeFilter.Label != "" && !matchesFilter(name, activeFilter) {
			continue
		}
		var sz int64
		var mt time.Time
		if info, err := de.Info(); err == nil {
			sz = info.Size()
			mt = info.ModTime()
		}
		s.Entries = append(s.Entries, FileEntry{
			Name:    name,
			IsDir:   isDir,
			Size:    sz,
			ModTime: mt,
		})
	}
	sortEntries(s.Entries, s.SortKey, s.SortAsc)
}

// ── Icon mapping ─────────────────────────────────────────────────

// fileIcon returns a Phosphor icon codepoint for the given filename.
func fileIcon(name string, isDir bool) string {
	if isDir {
		return icons.Folder
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go", ".py", ".js", ".ts", ".rs", ".c", ".h", ".cpp", ".java", ".rb", ".sh", ".lua":
		return icons.FileText
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".bmp", ".ico":
		return icons.ImageSquare
	case ".md", ".txt", ".csv", ".json", ".xml", ".yaml", ".yml", ".toml", ".html", ".css":
		return icons.FileText
	default:
		return icons.File
	}
}

// ── Size formatting ──────────────────────────────────────────────

func formatSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// ── FilePicker Element ───────────────────────────────────────────

// FilePicker shows a text field with the selected path and opens a
// browsing overlay when clicked.
type FilePicker struct {
	ui.BaseElement
	Value    string
	OnSelect func(string)
	State    *FilePickerState
	Disabled bool
	Mode     FilePickerMode
	Filters  []FileFilter
}

// FilePickerOption configures a FilePicker.
type FilePickerOption func(*FilePicker)

// WithFilePickerState links the picker to external state.
func WithFilePickerState(s *FilePickerState) FilePickerOption {
	return func(e *FilePicker) { e.State = s }
}

// WithOnFileSelect sets the selection callback.
func WithOnFileSelect(fn func(string)) FilePickerOption {
	return func(e *FilePicker) { e.OnSelect = fn }
}

// WithFilePickerMode sets open/save/directory mode.
func WithFilePickerMode(m FilePickerMode) FilePickerOption {
	return func(e *FilePicker) { e.Mode = m }
}

// WithFileFilters adds extension filters.
func WithFileFilters(filters ...FileFilter) FilePickerOption {
	return func(e *FilePicker) { e.Filters = filters }
}

// WithFilePickerDisabled disables the picker.
func WithFilePickerDisabled() FilePickerOption {
	return func(e *FilePicker) { e.Disabled = true }
}

// NewFilePicker creates a file picker element.
func NewFilePicker(value string, opts ...FilePickerOption) ui.Element {
	el := FilePicker{Value: value}
	for _, o := range opts {
		o(&el)
	}
	return el
}

// ── Layout (closed state + overlay) ──────────────────────────────

func (n FilePicker) LayoutSelf(ctx *ui.LayoutContext) ui.Bounds {
	area := ctx.Area
	canvas := ctx.Canvas
	tokens := ctx.Tokens
	ix := ctx.IX
	overlays := ctx.Overlays
	focus := ctx.Focus

	style := tokens.Typography.Body
	h := int(style.Size) + textFieldPadY*2
	w := filePickerFieldW
	if area.W < w {
		w = area.W
	}

	fieldRect := draw.R(float32(area.X), float32(area.Y), float32(w), float32(h))

	// Hit target.
	var hoverOpacity float32
	if n.Disabled {
		ix.RegisterHit(fieldRect, nil)
	} else {
		state := n.State
		filters := n.Filters
		dirOnly := n.Mode == FilePickerDirectory
		var clickFn func()
		if state != nil {
			clickFn = func() {
				state.Open = !state.Open
				if state.Open {
					state.loadDir(filters, dirOnly)
				}
			}
		}
		hoverOpacity = ix.RegisterHit(fieldRect, clickFn)
	}

	isOpen := n.State != nil && n.State.Open && !n.Disabled

	// Focus.
	var focused bool
	if focus != nil && !n.Disabled {
		uid := focus.NextElementUID()
		focus.RegisterFocusable(uid, ui.FocusOpts{Focusable: true, TabIndex: 0, FocusOnClick: true})
		focused = focus.IsElementFocused(uid)
	}

	// Border.
	borderColor := tokens.Colors.Stroke.Border
	if n.Disabled {
		borderColor = ui.DisabledColor(borderColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(fieldRect, tokens.Radii.Input, draw.SolidPaint(borderColor))

	// Fill.
	fillColor := tokens.Colors.Surface.Elevated
	if hoverOpacity > 0 {
		fillColor = ui.LerpColor(fillColor, tokens.Colors.Surface.Hovered, hoverOpacity)
	}
	if n.Disabled {
		fillColor = ui.DisabledColor(fillColor, tokens.Colors.Surface.Base)
	}
	canvas.FillRoundRect(
		draw.R(float32(area.X+1), float32(area.Y+1), float32(max(w-2, 0)), float32(max(h-2, 0))),
		maxf(tokens.Radii.Input-1, 0), draw.SolidPaint(fillColor))

	// Display text (selected path or placeholder).
	displayText := n.Value
	if displayText == "" {
		switch n.Mode {
		case FilePickerSave:
			displayText = "Save file…"
		case FilePickerDirectory:
			displayText = "Select directory…"
		default:
			displayText = "Select file…"
		}
	}
	textX := area.X + textFieldPadX
	textY := area.Y + textFieldPadY
	textColor := tokens.Colors.Text.Primary
	if n.Value == "" {
		textColor = tokens.Colors.Text.Secondary
	}
	if n.Disabled {
		textColor = tokens.Colors.Text.Disabled
	}

	// Clip text to leave room for icon.
	iconStyle := tokens.Typography.LabelSmall
	iconW := int(iconStyle.Size) + textFieldPadX
	clipW := w - textFieldPadX*2 - iconW
	if clipW > 0 {
		clipRect := draw.R(float32(textX), float32(area.Y), float32(clipW), float32(h))
		canvas.PushClip(clipRect)
		canvas.DrawText(displayText, draw.Pt(float32(textX), float32(textY)), style, textColor)
		canvas.PopClip()
	}

	// Folder icon on the right.
	iconStyle.FontFamily = "Phosphor"
	iconX := area.X + w - textFieldPadX - int(iconStyle.Size)
	iconColor := tokens.Colors.Text.Secondary
	if n.Disabled {
		iconColor = tokens.Colors.Text.Disabled
	}
	canvas.DrawText(icons.FolderOpen, draw.Pt(float32(iconX), float32(textY)), iconStyle, iconColor)

	// Focus ring.
	if focused || isOpen {
		ui.DrawFocusRing(canvas, fieldRect, tokens.Radii.Input, tokens)
	}

	// ── Overlay panel ────────────────────────────────────────────
	if isOpen && overlays != nil {
		state := n.State
		onSelect := n.OnSelect
		filters := n.Filters
		mode := n.Mode
		dirOnly := mode == FilePickerDirectory
		winW := overlays.WindowW
		winH := overlays.WindowH

		overlays.Push(ui.OverlayEntry{
			Render: func(canvas draw.Canvas, tokens theme.TokenSet, ix *ui.Interactor) {
				// Backdrop click closes.
				ix.RegisterHit(draw.R(0, 0, float32(winW), float32(winH)), func() {
					state.Open = false
				})

				panelW := filePickerPanelW
				panelH := filePickerPanelH
				if mode == FilePickerSave {
					panelH += filePickerRowH // extra row for save name
				}
				if len(filters) > 0 {
					panelH += filePickerRowH // extra row for filter dropdown
				}

				// Centre the panel.
				px := (winW - panelW) / 2
				py := (winH - panelH) / 2
				if py < 0 {
					py = 0
				}

				panelRect := draw.R(float32(px), float32(py), float32(panelW), float32(panelH))
				canvas.FillRoundRect(panelRect, tokens.Radii.Input, draw.SolidPaint(tokens.Colors.Surface.Elevated))
				canvas.StrokeRoundRect(panelRect, tokens.Radii.Input,
					draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1})

				bodyStyle := tokens.Typography.Body
				labelStyle := tokens.Typography.Label
				smallStyle := tokens.Typography.LabelSmall

				curY := py + filePickerPad

				// ── Path bar ─────────────────────────────────
				pathBarRect := draw.R(float32(px+filePickerPad), float32(curY), float32(panelW-filePickerPad*2), float32(filePickerPathH))
				canvas.FillRoundRect(pathBarRect, 4, draw.SolidPaint(tokens.Colors.Surface.Base))

				// Home button.
				homeRect := draw.R(float32(px+filePickerPad), float32(curY), float32(filePickerPathH), float32(filePickerPathH))
				homeHo := ix.RegisterHit(homeRect, func() {
					home, err := os.UserHomeDir()
					if err == nil {
						state.CurrentDir = home
						state.loadDir(filters, dirOnly)
					}
				})
				if homeHo > 0 {
					canvas.FillRoundRect(homeRect, 4, draw.SolidPaint(tokens.Colors.Surface.Hovered))
				}
				homeIconStyle := smallStyle
				homeIconStyle.FontFamily = "Phosphor"
				canvas.DrawText(icons.House, draw.Pt(
					float32(px+filePickerPad)+float32(filePickerPathH)/2-homeIconStyle.Size/2,
					float32(curY)+float32(filePickerPathH)/2-homeIconStyle.Size/2),
					homeIconStyle, tokens.Colors.Text.Primary)

				// Up button.
				upRect := draw.R(float32(px+filePickerPad+filePickerPathH), float32(curY), float32(filePickerPathH), float32(filePickerPathH))
				upHo := ix.RegisterHit(upRect, func() {
					parent := filepath.Dir(state.CurrentDir)
					if parent != state.CurrentDir {
						state.CurrentDir = parent
						state.loadDir(filters, dirOnly)
					}
				})
				if upHo > 0 {
					canvas.FillRoundRect(upRect, 4, draw.SolidPaint(tokens.Colors.Surface.Hovered))
				}
				canvas.DrawText(icons.ArrowLeft, draw.Pt(
					float32(px+filePickerPad+filePickerPathH)+float32(filePickerPathH)/2-homeIconStyle.Size/2,
					float32(curY)+float32(filePickerPathH)/2-homeIconStyle.Size/2),
					homeIconStyle, tokens.Colors.Text.Primary)

				// Path text.
				pathTextX := px + filePickerPad + filePickerPathH*2 + 4
				pathClipW := panelW - filePickerPad*2 - filePickerPathH*2 - 4
				if pathClipW > 0 {
					pathClip := draw.R(float32(pathTextX), float32(curY), float32(pathClipW), float32(filePickerPathH))
					canvas.PushClip(pathClip)
					canvas.DrawText(state.CurrentDir,
						draw.Pt(float32(pathTextX), float32(curY)+float32(filePickerPathH)/2-smallStyle.Size/2),
						smallStyle, tokens.Colors.Text.Secondary)
					canvas.PopClip()
				}

				curY += filePickerPathH + 4

				// ── Column headers ───────────────────────────
				headerY := curY
				headerRect := draw.R(float32(px+filePickerPad), float32(headerY), float32(panelW-filePickerPad*2), float32(filePickerHeaderH))
				canvas.FillRect(headerRect, draw.SolidPaint(tokens.Colors.Surface.Base))

				colX := px + filePickerPad + 4
				headerLabelY := float32(headerY) + float32(filePickerHeaderH)/2 - labelStyle.Size/2

				// Name header (sortable).
				nameHeaderRect := draw.R(float32(colX), float32(headerY), float32(filePickerNameCol), float32(filePickerHeaderH))
				nameHo := ix.RegisterHit(nameHeaderRect, func() {
					if state.SortKey == SortByName {
						state.SortAsc = !state.SortAsc
					} else {
						state.SortKey = SortByName
						state.SortAsc = true
					}
					sortEntries(state.Entries, state.SortKey, state.SortAsc)
				})
				nameLabel := "Name"
				if state.SortKey == SortByName {
					if state.SortAsc {
						nameLabel += " \u2191"
					} else {
						nameLabel += " \u2193"
					}
				}
				if nameHo > 0 {
					canvas.FillRect(nameHeaderRect, draw.SolidPaint(tokens.Colors.Surface.Hovered))
				}
				canvas.DrawText(nameLabel, draw.Pt(float32(colX), headerLabelY), labelStyle, tokens.Colors.Text.Secondary)

				// Size header.
				sizeX := colX + filePickerNameCol
				sizeHeaderRect := draw.R(float32(sizeX), float32(headerY), float32(filePickerSizeCol), float32(filePickerHeaderH))
				sizeHo := ix.RegisterHit(sizeHeaderRect, func() {
					if state.SortKey == SortBySize {
						state.SortAsc = !state.SortAsc
					} else {
						state.SortKey = SortBySize
						state.SortAsc = true
					}
					sortEntries(state.Entries, state.SortKey, state.SortAsc)
				})
				sizeLabel := "Size"
				if state.SortKey == SortBySize {
					if state.SortAsc {
						sizeLabel += " \u2191"
					} else {
						sizeLabel += " \u2193"
					}
				}
				if sizeHo > 0 {
					canvas.FillRect(sizeHeaderRect, draw.SolidPaint(tokens.Colors.Surface.Hovered))
				}
				canvas.DrawText(sizeLabel, draw.Pt(float32(sizeX), headerLabelY), labelStyle, tokens.Colors.Text.Secondary)

				// Modified header.
				modX := sizeX + filePickerSizeCol
				modW := panelW - filePickerPad*2 - (modX - px - filePickerPad)
				modHeaderRect := draw.R(float32(modX), float32(headerY), float32(modW), float32(filePickerHeaderH))
				modHo := ix.RegisterHit(modHeaderRect, func() {
					if state.SortKey == SortByModTime {
						state.SortAsc = !state.SortAsc
					} else {
						state.SortKey = SortByModTime
						state.SortAsc = true
					}
					sortEntries(state.Entries, state.SortKey, state.SortAsc)
				})
				modLabel := "Modified"
				if state.SortKey == SortByModTime {
					if state.SortAsc {
						modLabel += " \u2191"
					} else {
						modLabel += " \u2193"
					}
				}
				if modHo > 0 {
					canvas.FillRect(modHeaderRect, draw.SolidPaint(tokens.Colors.Surface.Hovered))
				}
				canvas.DrawText(modLabel, draw.Pt(float32(modX), headerLabelY), labelStyle, tokens.Colors.Text.Secondary)

				curY = headerY + filePickerHeaderH

				// ── File list (scrollable area) ──────────────
				listH := panelH - (curY - py) - filePickerFooterH - filePickerPad
				if mode == FilePickerSave {
					listH -= filePickerRowH
				}
				if len(filters) > 0 {
					listH -= filePickerRowH
				}
				listRect := draw.R(float32(px+filePickerPad), float32(curY), float32(panelW-filePickerPad*2), float32(listH))
				canvas.PushClip(listRect)

				contentH := len(state.Entries) * filePickerRowH
				maxScroll := contentH - listH
				if maxScroll < 0 {
					maxScroll = 0
				}
				if state.scrollY > maxScroll {
					state.scrollY = maxScroll
				}
				if state.scrollY < 0 {
					state.scrollY = 0
				}

				for i, entry := range state.Entries {
					ey := curY + i*filePickerRowH - state.scrollY
					if ey+filePickerRowH < curY || ey > curY+listH {
						continue // off-screen
					}

					entryRect := draw.R(float32(px+filePickerPad), float32(ey), float32(panelW-filePickerPad*2), float32(filePickerRowH))

					idx := i
					e := entry
					entryClick := func() {
						if e.IsDir {
							state.CurrentDir = filepath.Join(state.CurrentDir, e.Name)
							state.loadDir(filters, dirOnly)
						} else if mode != FilePickerDirectory {
							state.SelectedIdx = idx
							if mode == FilePickerSave {
								state.SaveName = e.Name
							}
						}
					}
					ho := ix.RegisterHit(entryRect, entryClick)

					// Selection / hover highlight.
					if i == state.SelectedIdx {
						canvas.FillRect(entryRect, draw.SolidPaint(tokens.Colors.Accent.Primary))
					} else if ho > 0 {
						canvas.FillRect(entryRect, draw.SolidPaint(tokens.Colors.Surface.Hovered))
					}

					textCol := tokens.Colors.Text.Primary
					if i == state.SelectedIdx {
						textCol = tokens.Colors.Text.OnAccent
					}

					rowY := float32(ey) + float32(filePickerRowH)/2 - smallStyle.Size/2

					// Icon.
					iconStyle := smallStyle
					iconStyle.FontFamily = "Phosphor"
					iconGlyph := fileIcon(e.Name, e.IsDir)
					iconCol := tokens.Colors.Text.Secondary
					if e.IsDir {
						iconCol = tokens.Colors.Accent.Primary
					}
					if i == state.SelectedIdx {
						iconCol = tokens.Colors.Text.OnAccent
					}
					canvas.DrawText(iconGlyph,
						draw.Pt(float32(colX), rowY),
						iconStyle, iconCol)

					// Name.
					nameClipRect := draw.R(float32(colX+filePickerIconW), float32(ey), float32(filePickerNameCol-filePickerIconW), float32(filePickerRowH))
					canvas.PushClip(nameClipRect)
					canvas.DrawText(e.Name,
						draw.Pt(float32(colX+filePickerIconW), rowY),
						smallStyle, textCol)
					canvas.PopClip()

					// Size (files only).
					if !e.IsDir {
						canvas.DrawText(formatSize(e.Size),
							draw.Pt(float32(sizeX), rowY),
							smallStyle, textCol)
					}

					// Modified date.
					if !e.ModTime.IsZero() {
						canvas.DrawText(e.ModTime.Format("2006-01-02"),
							draw.Pt(float32(modX), rowY),
							smallStyle, textCol)
					}
				}

				canvas.PopClip()

				curY += listH

				// ── Filter selector (optional) ───────────────
				if len(filters) > 0 {
					curY += 2
					filterY := curY
					for fi, f := range filters {
						filterLabel := f.Label
						if fi == state.FilterIdx {
							filterLabel = "\u25CF " + filterLabel
						} else {
							filterLabel = "\u25CB " + filterLabel
						}
						fRect := draw.R(float32(px+filePickerPad), float32(filterY), float32(panelW-filePickerPad*2)/float32(len(filters)), float32(filePickerRowH))
						fidx := fi
						fho := ix.RegisterHit(fRect, func() {
							if state.FilterIdx != fidx {
								state.FilterIdx = fidx
								state.loadDir(filters, dirOnly)
							}
						})
						if fho > 0 {
							canvas.FillRect(fRect, draw.SolidPaint(tokens.Colors.Surface.Hovered))
						}
						canvas.DrawText(filterLabel,
							draw.Pt(float32(px+filePickerPad)+float32(fi)*float32(panelW-filePickerPad*2)/float32(len(filters))+4,
								float32(filterY)+float32(filePickerRowH)/2-smallStyle.Size/2),
							smallStyle, tokens.Colors.Text.Secondary)
					}
					curY += filePickerRowH
				}

				// ── Save name input (save mode only) ─────────
				if mode == FilePickerSave {
					curY += 2
					saveRect := draw.R(float32(px+filePickerPad), float32(curY), float32(panelW-filePickerPad*2), float32(filePickerRowH))
					canvas.FillRoundRect(saveRect, 4, draw.SolidPaint(tokens.Colors.Surface.Base))
					canvas.StrokeRoundRect(saveRect, 4,
						draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1})
					saveText := state.SaveName
					if saveText == "" {
						saveText = "filename…"
					}
					saveColor := tokens.Colors.Text.Primary
					if state.SaveName == "" {
						saveColor = tokens.Colors.Text.Secondary
					}
					canvas.DrawText(saveText,
						draw.Pt(float32(px+filePickerPad+4), float32(curY)+float32(filePickerRowH)/2-smallStyle.Size/2),
						smallStyle, saveColor)
					curY += filePickerRowH
				}

				// ── Footer: Select / Cancel ──────────────────
				footerY := py + panelH - filePickerFooterH - filePickerPad/2
				btnW := 80
				btnH := 28
				btnGap := 8

				// Cancel button.
				cancelX := px + panelW - filePickerPad - btnW
				cancelRect := draw.R(float32(cancelX), float32(footerY), float32(btnW), float32(btnH))
				cancelHo := ix.RegisterHit(cancelRect, func() {
					state.Open = false
				})
				cancelFill := tokens.Colors.Surface.Base
				if cancelHo > 0 {
					cancelFill = tokens.Colors.Surface.Hovered
				}
				canvas.FillRoundRect(cancelRect, tokens.Radii.Input, draw.SolidPaint(cancelFill))
				canvas.StrokeRoundRect(cancelRect, tokens.Radii.Input,
					draw.Stroke{Paint: draw.SolidPaint(tokens.Colors.Stroke.Border), Width: 1})
				cancelM := canvas.MeasureText("Cancel", bodyStyle)
				canvas.DrawText("Cancel",
					draw.Pt(float32(cancelX)+float32(btnW)/2-cancelM.Width/2,
						float32(footerY)+float32(btnH)/2-bodyStyle.Size/2),
					bodyStyle, tokens.Colors.Text.Primary)

				// Select/Open/Save button.
				var selectLabel string
				switch mode {
				case FilePickerSave:
					selectLabel = "Save"
				case FilePickerDirectory:
					selectLabel = "Select"
				default:
					selectLabel = "Open"
				}
				selectX := cancelX - btnGap - btnW
				selectRect := draw.R(float32(selectX), float32(footerY), float32(btnW), float32(btnH))
				selectClick := func() {
					var result string
					switch mode {
					case FilePickerDirectory:
						result = state.CurrentDir
					case FilePickerSave:
						if state.SaveName != "" {
							result = filepath.Join(state.CurrentDir, state.SaveName)
						}
					default: // Open
						if state.SelectedIdx >= 0 && state.SelectedIdx < len(state.Entries) {
							e := state.Entries[state.SelectedIdx]
							if !e.IsDir {
								result = filepath.Join(state.CurrentDir, e.Name)
							}
						}
					}
					if result != "" {
						state.Open = false
						if onSelect != nil {
							onSelect(result)
						}
					}
				}
				selectHo := ix.RegisterHit(selectRect, selectClick)
				selectFill := tokens.Colors.Accent.Primary
				if selectHo > 0 {
					selectFill = ui.LerpColor(selectFill, tokens.Colors.Surface.Hovered, selectHo*0.3)
				}
				canvas.FillRoundRect(selectRect, tokens.Radii.Input, draw.SolidPaint(selectFill))
				selectM := canvas.MeasureText(selectLabel, bodyStyle)
				canvas.DrawText(selectLabel,
					draw.Pt(float32(selectX)+float32(btnW)/2-selectM.Width/2,
						float32(footerY)+float32(btnH)/2-bodyStyle.Size/2),
					bodyStyle, tokens.Colors.Text.OnAccent)
			},
		})
	}

	return ui.Bounds{X: area.X, Y: area.Y, W: w, H: h}
}

// TreeEqual implements ui.TreeEqualizer.
func (n FilePicker) TreeEqual(other ui.Element) bool {
	nb, ok := other.(FilePicker)
	return ok && n.Value == nb.Value && n.Mode == nb.Mode && n.Disabled == nb.Disabled
}

// ResolveChildren implements ui.ChildResolver. FilePicker is a leaf.
func (n FilePicker) ResolveChildren(resolve func(ui.Element, int) ui.Element) ui.Element {
	return n
}

// WalkAccess implements ui.AccessWalker.
func (n FilePicker) WalkAccess(b *ui.AccessTreeBuilder, parentIdx int32) {
	label := "File picker"
	if n.Mode == FilePickerDirectory {
		label = "Directory picker"
	}
	b.AddNode(a11y.AccessNode{
		Role:   a11y.RoleCombobox,
		Label:  label,
		Value:  n.Value,
		States: a11y.AccessStates{Disabled: n.Disabled},
	}, parentIdx, a11y.Rect{})
}
