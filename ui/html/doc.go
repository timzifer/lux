// Package html provides an HTML viewer widget for the Lux UI framework.
//
// Unlike richtext.FromHTML, which flattens HTML into a single
// AttributedString (losing structural information like tables, forms, and
// nested block layout), this package builds a full ui.Element tree from
// parsed HTML, mapping HTML elements to their native Lux widget
// counterparts.
//
// Architecture (RFC-998 Phase 1 — Static HTML/CSS Viewer):
//
//	HTML string
//	    │
//	    ▼
//	dom.ParseHTML()  →  *dom.Node tree (internal shadow DOM)
//	    │
//	    ▼
//	<style> extraction  →  []*css.StyleSheet
//	    │
//	    ▼
//	Document{Root, Sheets}
//	    │
//	    ▼
//	builder.buildElement()  →  recursive DOM→Element conversion
//	    │
//	    ▼
//	ui.Element tree (rendered by the Lux framework)
//
// The package is designed as the foundation for a future browser engine
// (see RFC-998). The Document/Builder/Widget separation allows for
// incremental re-rendering, DOM mutations, and event handling in later
// phases.
//
// Usage:
//
//	el := html.View(`<h1>Hello</h1><p>World</p>`)
//
//	// With options:
//	el := html.View(htmlStr, html.WithOnLink(func(href string) {
//	    fmt.Println("navigating to", href)
//	}))
//
//	// From pre-parsed document:
//	doc, _ := html.Parse(htmlStr)
//	el := html.ViewFromDocument(doc)
package html
