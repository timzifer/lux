package dom

import (
	"strings"
)

// voidElements are HTML elements that must not have a closing tag.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true,
	"embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "param": true, "source": true,
	"track": true, "wbr": true,
}

// Serialize converts a DOM tree back to an HTML string.
func Serialize(n *Node) string {
	var b strings.Builder
	serialize(&b, n)
	return b.String()
}

func serialize(b *strings.Builder, n *Node) {
	switch n.Type {
	case DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSib {
			serialize(b, c)
		}
	case TextNode:
		b.WriteString(escapeText(n.Data))
	case CommentNode:
		b.WriteString("<!--")
		b.WriteString(n.Data)
		b.WriteString("-->")
	case ElementNode:
		b.WriteByte('<')
		b.WriteString(n.Tag)
		for k, v := range n.Attrs {
			b.WriteByte(' ')
			b.WriteString(k)
			b.WriteString(`="`)
			b.WriteString(escapeAttr(v))
			b.WriteByte('"')
		}
		b.WriteByte('>')
		if voidElements[n.Tag] {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSib {
			serialize(b, c)
		}
		b.WriteString("</")
		b.WriteString(n.Tag)
		b.WriteByte('>')
	}
}

func escapeText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func escapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
