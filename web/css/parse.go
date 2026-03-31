package css

import (
	"strings"

	"github.com/tdewolff/parse/v2"
	tdcss "github.com/tdewolff/parse/v2/css"
)

// ParseStyleAttribute parses an inline style="..." attribute value
// into a StyleDeclaration.
func ParseStyleAttribute(s string) StyleDeclaration {
	decl := NewDecl()
	p := tdcss.NewParser(parse.NewInput(strings.NewReader(s)), true)
	for {
		gt, _, data := p.Next()
		if gt == tdcss.ErrorGrammar {
			break
		}
		if gt == tdcss.DeclarationGrammar {
			prop := strings.TrimSpace(string(data))
			val := tokensToString(p.Values())
			if prop != "" {
				decl.Set(prop, val)
			}
		}
	}
	return decl
}

// ParseStyleSheet parses a CSS stylesheet string (e.g. from a <style> block)
// into a StyleSheet.
func ParseStyleSheet(cssText string) (*StyleSheet, error) {
	sheet := &StyleSheet{}
	p := tdcss.NewParser(parse.NewInput(strings.NewReader(cssText)), false)
	var currentSelector string
	for {
		gt, _, data := p.Next()
		if gt == tdcss.ErrorGrammar {
			break
		}
		switch gt {
		case tdcss.BeginRulesetGrammar:
			currentSelector = selectorFromTokens(data, p.Values())
		case tdcss.DeclarationGrammar:
			prop := strings.TrimSpace(string(data))
			val := tokensToString(p.Values())
			if prop != "" && currentSelector != "" {
				found := false
				for i := range sheet.Rules {
					if sheet.Rules[i].Selector == currentSelector {
						sheet.Rules[i].Decl.Set(prop, val)
						found = true
						break
					}
				}
				if !found {
					rule := StyleRule{
						Selector: currentSelector,
						Decl:     NewDecl(),
					}
					rule.Decl.Set(prop, val)
					sheet.Rules = append(sheet.Rules, rule)
				}
			}
		case tdcss.EndRulesetGrammar:
			currentSelector = ""
		}
	}
	return sheet, nil
}

// tokensToString joins CSS value tokens into a single string.
func tokensToString(tokens []tdcss.Token) string {
	var parts []string
	for _, t := range tokens {
		s := string(t.Data)
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.TrimSpace(strings.Join(parts, ""))
}

// selectorFromTokens builds a selector string from the data and value tokens.
func selectorFromTokens(data []byte, tokens []tdcss.Token) string {
	var parts []string
	if len(data) > 0 {
		parts = append(parts, string(data))
	}
	for _, t := range tokens {
		parts = append(parts, string(t.Data))
	}
	return strings.TrimSpace(strings.Join(parts, ""))
}
