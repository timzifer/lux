package html

import (
	"github.com/timzifer/lux/web/css"
	"github.com/timzifer/lux/web/dom"
)

// Document holds a parsed HTML document with its associated stylesheets.
// It serves as the internal "shadow DOM" of the HTML widget.
type Document struct {
	// Root is the top-level DOM node (DocumentNode).
	Root *dom.Node

	// Sheets contains stylesheets extracted from <style> elements
	// and any externally provided sheets.
	Sheets []*css.StyleSheet
}

// Parse parses an HTML string into a Document, extracting <style>
// blocks into stylesheets.
func Parse(htmlStr string) (*Document, error) {
	root, err := dom.ParseHTML(htmlStr)
	if err != nil {
		return nil, err
	}
	sheets := extractStyleSheets(root)
	return &Document{Root: root, Sheets: sheets}, nil
}

// ParseFragment parses an HTML fragment (no <html>/<body> wrapper
// required) into a Document. This is equivalent to Parse for most
// practical purposes since dom.ParseHTML already handles fragments.
func ParseFragment(htmlStr string) (*Document, error) {
	return Parse(htmlStr)
}

// AddStyleSheet appends an external stylesheet to the document.
func (d *Document) AddStyleSheet(sheet *css.StyleSheet) {
	d.Sheets = append(d.Sheets, sheet)
}

// AddCSS parses a CSS string and appends it as a stylesheet.
func (d *Document) AddCSS(cssText string) error {
	sheet, err := css.ParseStyleSheet(cssText)
	if err != nil {
		return err
	}
	d.Sheets = append(d.Sheets, sheet)
	return nil
}

// extractStyleSheets finds all <style> elements in the DOM tree and
// parses their text content into stylesheets.
func extractStyleSheets(root *dom.Node) []*css.StyleSheet {
	var sheets []*css.StyleSheet
	for _, styleEl := range root.GetElementsByTagName("style") {
		text := styleEl.TextContent()
		if text != "" {
			sheet, err := css.ParseStyleSheet(text)
			if err == nil {
				sheets = append(sheets, sheet)
			}
		}
	}
	return sheets
}
