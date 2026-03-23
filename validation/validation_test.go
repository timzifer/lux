package validation

import (
	"regexp"
	"testing"
)

func TestRequired(t *testing.T) {
	tests := []struct {
		value string
		valid bool
	}{
		{"hello", true},
		{"", false},
		{"   ", false},
		{"  x ", true},
	}
	for _, tt := range tests {
		r := Validate(tt.value, Required)
		if r.Valid() != tt.valid {
			t.Errorf("Required(%q): got valid=%v, want %v", tt.value, r.Valid(), tt.valid)
		}
	}
}

func TestMinLen(t *testing.T) {
	v := MinLen(3)
	if msg := v("ab"); msg == "" {
		t.Error("MinLen(3) should reject 'ab'")
	}
	if msg := v("abc"); msg != "" {
		t.Errorf("MinLen(3) should accept 'abc', got %q", msg)
	}
	if msg := v("über"); msg != "" {
		t.Errorf("MinLen(3) should accept 'über' (4 runes), got %q", msg)
	}
}

func TestMaxLen(t *testing.T) {
	v := MaxLen(5)
	if msg := v("hello"); msg != "" {
		t.Errorf("MaxLen(5) should accept 'hello', got %q", msg)
	}
	if msg := v("toolong"); msg == "" {
		t.Error("MaxLen(5) should reject 'toolong'")
	}
}

func TestPattern(t *testing.T) {
	re := regexp.MustCompile(`^\d{5}$`)
	v := Pattern(re, "Must be 5 digits")
	if msg := v("12345"); msg != "" {
		t.Errorf("Pattern should accept '12345', got %q", msg)
	}
	if msg := v("abcde"); msg == "" {
		t.Error("Pattern should reject 'abcde'")
	}
	// Empty values pass (use Required for mandatory check).
	if msg := v(""); msg != "" {
		t.Errorf("Pattern should accept empty, got %q", msg)
	}
}

func TestEmail(t *testing.T) {
	valid := []string{"user@example.com", "a@b.c", ""}
	for _, v := range valid {
		if msg := Email(v); msg != "" {
			t.Errorf("Email(%q) should be valid, got %q", v, msg)
		}
	}
	invalid := []string{"notanemail", "@no.com", "user@", "user @host.com"}
	for _, v := range invalid {
		if msg := Email(v); msg == "" {
			t.Errorf("Email(%q) should be invalid", v)
		}
	}
}

func TestMinMaxVal(t *testing.T) {
	min := MinVal(10)
	if msg := min("5"); msg == "" {
		t.Error("MinVal(10) should reject '5'")
	}
	if msg := min("15"); msg != "" {
		t.Errorf("MinVal(10) should accept '15', got %q", msg)
	}
	if msg := min(""); msg != "" {
		t.Errorf("MinVal should accept empty, got %q", msg)
	}
	if msg := min("abc"); msg == "" {
		t.Error("MinVal should reject non-numeric")
	}

	max := MaxVal(100)
	if msg := max("50"); msg != "" {
		t.Errorf("MaxVal(100) should accept '50', got %q", msg)
	}
	if msg := max("200"); msg == "" {
		t.Error("MaxVal(100) should reject '200'")
	}
}

func TestCustom(t *testing.T) {
	v := Custom(func(s string) bool { return s == "ok" }, "must be ok")
	if msg := v("ok"); msg != "" {
		t.Errorf("Custom should accept 'ok', got %q", msg)
	}
	if msg := v("nope"); msg == "" {
		t.Error("Custom should reject 'nope'")
	}
}

func TestValidateChaining(t *testing.T) {
	r := Validate("ab", Required, MinLen(3))
	if r.Valid() {
		t.Error("Should fail MinLen(3)")
	}
	if r.Error != "Must be at least 3 characters" {
		t.Errorf("Unexpected error: %q", r.Error)
	}

	// First failing validator wins.
	r = Validate("", Required, MinLen(3))
	if r.Error != "This field is required" {
		t.Errorf("Expected Required error, got %q", r.Error)
	}
}

func TestFieldResult(t *testing.T) {
	valid := FieldResult{}
	if !valid.Valid() {
		t.Error("Zero FieldResult should be valid")
	}
	invalid := FieldResult{Error: "oops"}
	if invalid.Valid() {
		t.Error("FieldResult with error should be invalid")
	}
}

func TestEqualField(t *testing.T) {
	cv := EqualField("password")
	all := map[string]string{"password": "secret", "confirm": "secret"}
	if msg := cv("secret", all); msg != "" {
		t.Errorf("EqualField should pass when values match, got %q", msg)
	}
	if msg := cv("other", all); msg == "" {
		t.Error("EqualField should fail when values differ")
	}
}

func TestRequiredIf(t *testing.T) {
	cv := RequiredIf("email")
	all := map[string]string{"email": "a@b.com", "name": ""}
	if msg := cv("", all); msg == "" {
		t.Error("RequiredIf should fail when other field is non-empty and this is empty")
	}
	all["email"] = ""
	if msg := cv("", all); msg != "" {
		t.Errorf("RequiredIf should pass when other field is empty, got %q", msg)
	}
}

func TestSchema(t *testing.T) {
	schema := Schema{
		"email":    Rules(Required, Email),
		"password": Rules(Required, MinLen(8)),
		"confirm":  RulesCross([]Validator{Required}, EqualField("password")),
	}

	values := map[string]string{
		"email":    "user@example.com",
		"password": "12345678",
		"confirm":  "12345678",
	}
	results := schema.ValidateMap(values)
	if !results.Valid() {
		t.Error("Valid form should pass")
	}

	// Invalid email.
	values["email"] = "bad"
	results = schema.ValidateMap(values)
	if results.Get("email").Valid() {
		t.Error("Bad email should fail")
	}
	if !results.Get("password").Valid() {
		t.Error("Password should still be valid")
	}

	// Mismatched confirm.
	values["email"] = "user@example.com"
	values["confirm"] = "wrong"
	results = schema.ValidateMap(values)
	if results.Get("confirm").Valid() {
		t.Error("Mismatched confirm should fail")
	}
}

func TestFormResultGet(t *testing.T) {
	var nilResult FormResult
	r := nilResult.Get("missing")
	if !r.Valid() {
		t.Error("Get on nil FormResult should return valid result")
	}

	fr := FormResult{"x": FieldResult{Error: "err"}}
	if fr.Get("x").Valid() {
		t.Error("Get('x') should return invalid result")
	}
	if !fr.Get("y").Valid() {
		t.Error("Get('y') on missing key should return valid result")
	}
}
