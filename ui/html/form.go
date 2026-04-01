package html

import (
	"strings"
	"time"

	"github.com/timzifer/lux/draw"
	"github.com/timzifer/lux/ui"
	"github.com/timzifer/lux/ui/button"
	"github.com/timzifer/lux/ui/form"
	"github.com/timzifer/lux/web/css"
	"github.com/timzifer/lux/web/dom"
)

// buildFormControl converts an <input>, <select>, <textarea>, <button>,
// or <progress> DOM node into the corresponding Lux form widget.
func (b *builder) buildFormControl(node *dom.Node, style css.StyleDeclaration) ui.Element {
	tag := strings.ToLower(node.Tag)

	switch tag {
	case "input":
		return b.buildInput(node, style)
	case "select":
		return b.buildSelect(node)
	case "textarea":
		return b.buildTextArea(node, style)
	case "button":
		return b.buildButton(node, style)
	case "progress":
		return b.buildProgress(node)
	}

	return nil
}

// buildInput converts an <input> element based on its type attribute.
func (b *builder) buildInput(node *dom.Node, style css.StyleDeclaration) ui.Element {
	inputType := strings.ToLower(node.Attr("type"))
	if inputType == "" {
		inputType = "text"
	}
	value := node.Attr("value")
	placeholder := node.Attr("placeholder")
	disabled := node.Attr("disabled") != ""

	switch inputType {
	case "text", "email", "url", "tel", "search":
		var opts []form.TextFieldOption
		if disabled {
			opts = append(opts, form.WithDisabled())
		}
		return form.NewTextField(value, placeholder, opts...)

	case "password":
		var opts []form.PasswordFieldOption
		if disabled {
			opts = append(opts, form.PasswordDisabled())
		}
		return form.NewPasswordField(value, placeholder, opts...)

	case "checkbox":
		checked := node.Attr("checked") != ""
		if disabled {
			return form.CheckboxDisabled("", checked)
		}
		return form.NewCheckbox("", checked, nil)

	case "radio":
		selected := node.Attr("checked") != ""
		if disabled {
			return form.RadioDisabled("", selected)
		}
		return form.NewRadio("", selected, nil)

	case "number":
		var opts []form.NumericInputOption
		if disabled {
			opts = append(opts, form.WithNumericDisabled())
		}
		return form.NewNumericInput(0, opts...)

	case "range":
		if disabled {
			return form.SliderDisabled(0.5)
		}
		return form.NewSlider(0.5, nil)

	case "date":
		var opts []form.DatePickerOption
		if disabled {
			opts = append(opts, form.WithDatePickerDisabled())
		}
		return form.NewDatePicker(time.Time{}, opts...)

	case "time":
		var opts []form.TimePickerOption
		if disabled {
			opts = append(opts, form.WithTimePickerDisabled())
		}
		return form.NewTimePicker(0, 0, opts...)

	case "color":
		var opts []form.ColorPickerOption
		if disabled {
			opts = append(opts, form.WithColorPickerDisabled())
		}
		return form.NewColorPicker(draw.Color{}, opts...)

	case "file":
		var opts []form.FilePickerOption
		if disabled {
			opts = append(opts, form.WithFilePickerDisabled())
		}
		return form.NewFilePicker("", opts...)

	case "submit", "reset", "button":
		label := value
		if label == "" {
			label = inputType
		}
		if disabled {
			return button.TextDisabled(label)
		}
		return button.Text(label, nil)

	case "hidden":
		return nil

	default:
		// Unknown input type — render as text field.
		var opts []form.TextFieldOption
		if disabled {
			opts = append(opts, form.WithDisabled())
		}
		return form.NewTextField(value, placeholder, opts...)
	}
}

// buildSelect converts a <select> element.
func (b *builder) buildSelect(node *dom.Node) ui.Element {
	var options []string
	var selected string

	for child := node.FirstChild; child != nil; child = child.NextSib {
		if child.Type == dom.ElementNode && strings.ToLower(child.Tag) == "option" {
			text := child.TextContent()
			options = append(options, text)
			if child.Attr("selected") != "" {
				selected = text
			}
		}
	}

	if selected == "" && len(options) > 0 {
		selected = options[0]
	}

	disabled := node.Attr("disabled") != ""
	var opts []form.SelectOption
	if disabled {
		opts = append(opts, form.WithSelectDisabled())
	}

	return form.NewSelect(selected, options, opts...)
}

// buildTextArea converts a <textarea> element.
func (b *builder) buildTextArea(node *dom.Node, style css.StyleDeclaration) ui.Element {
	value := node.TextContent()
	placeholder := node.Attr("placeholder")
	disabled := node.Attr("disabled") != ""

	var opts []form.TextAreaOption
	if disabled {
		opts = append(opts, form.TextAreaDisabled())
	}

	return form.NewTextArea(value, placeholder, opts...)
}

// buildButton converts a <button> element.
func (b *builder) buildButton(node *dom.Node, style css.StyleDeclaration) ui.Element {
	label := node.TextContent()
	if label == "" {
		label = "Button"
	}
	disabled := node.Attr("disabled") != ""

	if disabled {
		return button.TextDisabled(label)
	}
	return button.Text(label, nil)
}

// buildProgress converts a <progress> element.
func (b *builder) buildProgress(node *dom.Node) ui.Element {
	value := node.Attr("value")
	if value == "" {
		return form.Indeterminate()
	}
	if v, ok := css.ParseDimension(value); ok {
		maxVal := float32(1.0)
		if m := node.Attr("max"); m != "" {
			if mv, ok := css.ParseDimension(m); ok {
				maxVal = mv
			}
		}
		if maxVal > 0 {
			return form.NewProgressBar(v / maxVal)
		}
	}
	return form.NewProgressBar(0)
}
