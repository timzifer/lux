// Package validation provides a pluggable form field validation framework.
//
// Validators are pure functions that inspect a value and return a result.
// They run in the Elm Update cycle — the application stores FieldResult
// values in its Model and passes them to FormField elements in the View.
//
// Single-field validation:
//
//	result := validation.Validate(email, validation.Required, validation.Email)
//
// Cross-field validation:
//
//	schema := validation.Schema{
//	    "password": {validation.Required, validation.MinLen(8)},
//	    "confirm":  {validation.Required, validation.EqualField("password")},
//	}
//	results := schema.ValidateMap(map[string]string{
//	    "password": pw,
//	    "confirm":  confirmPw,
//	})
package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ── Core Types ──────────────────────────────────────────────────

// Validator inspects a string value and returns an error message,
// or "" if the value is valid.
type Validator func(value string) string

// CrossValidator inspects a field value in the context of all form
// field values and returns an error message, or "" if valid.
type CrossValidator func(value string, allValues map[string]string) string

// FieldResult holds the outcome of validating a single field.
type FieldResult struct {
	Error string // non-empty means invalid
}

// Valid reports whether the field passed validation.
func (r FieldResult) Valid() bool { return r.Error == "" }

// FormResult maps field names to their validation results.
type FormResult map[string]FieldResult

// Valid reports whether all fields passed validation.
func (r FormResult) Valid() bool {
	for _, fr := range r {
		if !fr.Valid() {
			return false
		}
	}
	return true
}

// Get returns the FieldResult for name, or a valid (zero) result if absent.
func (r FormResult) Get(name string) FieldResult {
	if r == nil {
		return FieldResult{}
	}
	return r[name]
}

// ── Validate (single field) ─────────────────────────────────────

// Validate runs validators in order and returns the first error.
func Validate(value string, validators ...Validator) FieldResult {
	for _, v := range validators {
		if msg := v(value); msg != "" {
			return FieldResult{Error: msg}
		}
	}
	return FieldResult{}
}

// ── Schema (multi-field / cross-field) ──────────────────────────

// FieldRules describes the validators for a single field inside a Schema.
// It supports both single-field Validators and CrossValidators.
type FieldRules struct {
	Validators      []Validator
	CrossValidators []CrossValidator
}

// Rules is a convenience constructor for field rules with only single-field validators.
func Rules(vs ...Validator) FieldRules {
	return FieldRules{Validators: vs}
}

// RulesCross creates field rules that include cross-field validators.
func RulesCross(vs []Validator, cvs ...CrossValidator) FieldRules {
	return FieldRules{Validators: vs, CrossValidators: cvs}
}

// Schema maps field names to their validation rules.
type Schema map[string]FieldRules

// ValidateMap validates all fields in values according to the schema.
// Fields present in the schema but absent from values are treated as "".
func (s Schema) ValidateMap(values map[string]string) FormResult {
	res := make(FormResult, len(s))
	for name, rules := range s {
		val := values[name]
		// Run single-field validators first.
		fr := Validate(val, rules.Validators...)
		if fr.Valid() {
			// Run cross-field validators.
			for _, cv := range rules.CrossValidators {
				if msg := cv(val, values); msg != "" {
					fr = FieldResult{Error: msg}
					break
				}
			}
		}
		res[name] = fr
	}
	return res
}

// ── Built-in Validators ─────────────────────────────────────────

// Required rejects empty (whitespace-only) values.
func Required(value string) string {
	if strings.TrimSpace(value) == "" {
		return "This field is required"
	}
	return ""
}

// MinLen rejects values shorter than n runes.
func MinLen(n int) Validator {
	return func(value string) string {
		if utf8.RuneCountInString(value) < n {
			return fmt.Sprintf("Must be at least %d characters", n)
		}
		return ""
	}
}

// MaxLen rejects values longer than n runes.
func MaxLen(n int) Validator {
	return func(value string) string {
		if utf8.RuneCountInString(value) > n {
			return fmt.Sprintf("Must be at most %d characters", n)
		}
		return ""
	}
}

// Pattern rejects values that do not match the regular expression.
func Pattern(re *regexp.Regexp, msg string) Validator {
	return func(value string) string {
		if value != "" && !re.Match([]byte(value)) {
			return msg
		}
		return ""
	}
}

// Email is a simple e-mail format validator.
var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// Email rejects values that are not valid e-mail addresses.
func Email(value string) string {
	if value != "" && !emailRe.MatchString(value) {
		return "Invalid email address"
	}
	return ""
}

// MinVal rejects numeric values below min.
func MinVal(min float64) Validator {
	return func(value string) string {
		if value == "" {
			return ""
		}
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return "Must be a number"
		}
		if v < min {
			return fmt.Sprintf("Must be at least %g", min)
		}
		return ""
	}
}

// MaxVal rejects numeric values above max.
func MaxVal(max float64) Validator {
	return func(value string) string {
		if value == "" {
			return ""
		}
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return "Must be a number"
		}
		if v > max {
			return fmt.Sprintf("Must be at most %g", max)
		}
		return ""
	}
}

// Custom creates a validator from an arbitrary predicate.
func Custom(pred func(string) bool, msg string) Validator {
	return func(value string) string {
		if !pred(value) {
			return msg
		}
		return ""
	}
}

// ── Built-in Cross Validators ───────────────────────────────────

// EqualField rejects values that differ from the named sibling field.
func EqualField(otherField string) CrossValidator {
	return func(value string, all map[string]string) string {
		if value != all[otherField] {
			return fmt.Sprintf("Must match %s", otherField)
		}
		return ""
	}
}

// RequiredIf rejects empty values when the named sibling field is non-empty.
func RequiredIf(otherField string) CrossValidator {
	return func(value string, all map[string]string) string {
		if strings.TrimSpace(all[otherField]) != "" && strings.TrimSpace(value) == "" {
			return "This field is required"
		}
		return ""
	}
}
