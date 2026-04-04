package form

import "testing"

func TestIPInput_SegmentCount(t *testing.T) {
	ip4 := IPInput{Version: IPVersion4}
	if got := ip4.SegmentCount(); got != 4 {
		t.Errorf("IPv4 SegmentCount() = %d, want 4", got)
	}

	ip6 := IPInput{Version: IPVersion6}
	if got := ip6.SegmentCount(); got != 8 {
		t.Errorf("IPv6 SegmentCount() = %d, want 8", got)
	}
}

func TestIPInput_Segments(t *testing.T) {
	ip := IPInput{Value: "192.168.1.1", Version: IPVersion4}
	segs := ip.Segments()
	if len(segs) != 4 {
		t.Fatalf("Segments() len = %d, want 4", len(segs))
	}
	want := []string{"192", "168", "1", "1"}
	for i, s := range segs {
		if s != want[i] {
			t.Errorf("Segments()[%d] = %q, want %q", i, s, want[i])
		}
	}
}

func TestIPInput_Segments_Partial(t *testing.T) {
	ip := IPInput{Value: "10.0", Version: IPVersion4}
	segs := ip.Segments()
	if len(segs) != 4 {
		t.Fatalf("partial Segments() len = %d, want 4", len(segs))
	}
	if segs[0] != "10" || segs[1] != "0" || segs[2] != "" || segs[3] != "" {
		t.Errorf("partial Segments() = %v", segs)
	}
}

func TestIPInput_ValidateSegment_IPv4(t *testing.T) {
	ip := IPInput{Version: IPVersion4}

	tests := []struct {
		seg  string
		ok   bool
	}{
		{"0", true},
		{"255", true},
		{"256", false},
		{"", true},
		{"abc", false},
		{"-1", false},
	}
	for _, tt := range tests {
		if got := ip.ValidateSegment(tt.seg, IPVersion4); got != tt.ok {
			t.Errorf("ValidateSegment(%q, IPv4) = %v, want %v", tt.seg, got, tt.ok)
		}
	}
}

func TestIPInput_ValidateSegment_IPv6(t *testing.T) {
	ip := IPInput{Version: IPVersion6}

	tests := []struct {
		seg  string
		ok   bool
	}{
		{"0", true},
		{"ffff", true},
		{"FFFF", true},
		{"1a2b", true},
		{"12345", false}, // too long
		{"gggg", false},
		{"", true},
	}
	for _, tt := range tests {
		if got := ip.ValidateSegment(tt.seg, IPVersion6); got != tt.ok {
			t.Errorf("ValidateSegment(%q, IPv6) = %v, want %v", tt.seg, got, tt.ok)
		}
	}
}

func TestFilterIPChars(t *testing.T) {
	tests := []struct {
		in      string
		version IPVersion
		want    string
	}{
		{"192.168.1.1", IPVersion4, "192.168.1.1"},
		{"192abc168", IPVersion4, "192168"},       // letters filtered
		{"abc:def:123", IPVersion6, "abc:def:123"}, // hex + colon
	}
	for _, tt := range tests {
		got := filterIPChars(tt.in, tt.version)
		if got != tt.want {
			t.Errorf("filterIPChars(%q, %d) = %q, want %q", tt.in, tt.version, got, tt.want)
		}
	}
}

func TestIsCompleteIP(t *testing.T) {
	if !isCompleteIP("192.168.1.1", IPVersion4) {
		t.Error("192.168.1.1 should be complete IPv4")
	}
	if isCompleteIP("192.168.1.", IPVersion4) {
		t.Error("trailing dot should not be complete")
	}
	if isCompleteIP("192.168.1.999", IPVersion4) {
		t.Error("999 segment should not be complete")
	}
}

func TestIPInput_AutoDetect(t *testing.T) {
	// Contains colon → IPv6.
	ip := IPInput{Value: "::1", Version: IPVersionAuto}
	if got := ip.effectiveVersion(); got != IPVersion6 {
		t.Errorf("auto-detect '::1' = %d, want IPv6", got)
	}

	// No colon → IPv4.
	ip.Value = "10.0.0.1"
	if got := ip.effectiveVersion(); got != IPVersion4 {
		t.Errorf("auto-detect '10.0.0.1' = %d, want IPv4", got)
	}
}
