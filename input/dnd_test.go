package input

import "testing"

func TestDragOperationHas(t *testing.T) {
	op := DragOperationMove | DragOperationCopy
	if !op.Has(DragOperationMove) {
		t.Error("expected Move to be set")
	}
	if !op.Has(DragOperationCopy) {
		t.Error("expected Copy to be set")
	}
	if op.Has(DragOperationLink) {
		t.Error("expected Link to NOT be set")
	}
	if DragOperationNone.Has(DragOperationMove) {
		t.Error("expected None to not have Move")
	}
}

func TestDropEffectString(t *testing.T) {
	tests := []struct {
		e    DropEffect
		want string
	}{
		{DropEffectNone, "none"},
		{DropEffectMove, "move"},
		{DropEffectCopy, "copy"},
		{DropEffectLink, "link"},
	}
	for _, tt := range tests {
		if got := tt.e.String(); got != tt.want {
			t.Errorf("DropEffect(%d).String() = %q, want %q", tt.e, got, tt.want)
		}
	}
}

func TestDragDataHasType(t *testing.T) {
	d := &DragData{
		Items: []DragItem{
			{MIMEType: MIMEText, Data: "hello"},
			{MIMEType: MIMEJSON, Data: `{"k":"v"}`},
		},
	}
	if !d.HasType(MIMEText) {
		t.Error("expected HasType(text/plain) = true")
	}
	if !d.HasType(MIMEJSON) {
		t.Error("expected HasType(application/json) = true")
	}
	if d.HasType(MIMEURIList) {
		t.Error("expected HasType(text/uri-list) = false")
	}
}

func TestDragDataHasTypeNil(t *testing.T) {
	var d *DragData
	if d.HasType(MIMEText) {
		t.Error("nil DragData should not have any type")
	}
}

func TestDragDataGet(t *testing.T) {
	d := &DragData{
		Items: []DragItem{
			{MIMEType: MIMEText, Data: "hello"},
			{MIMEType: MIMEJSON, Data: 42},
		},
	}

	v, ok := d.Get(MIMEText)
	if !ok || v != "hello" {
		t.Errorf("Get(text/plain) = (%v, %v), want (hello, true)", v, ok)
	}

	v, ok = d.Get(MIMEJSON)
	if !ok || v != 42 {
		t.Errorf("Get(application/json) = (%v, %v), want (42, true)", v, ok)
	}

	v, ok = d.Get(MIMEURIList)
	if ok {
		t.Errorf("Get(text/uri-list) = (%v, true), want (nil, false)", v)
	}
}

func TestDragDataGetNil(t *testing.T) {
	var d *DragData
	v, ok := d.Get(MIMEText)
	if ok || v != nil {
		t.Errorf("nil DragData.Get() = (%v, %v), want (nil, false)", v, ok)
	}
}

func TestDragDataSetText(t *testing.T) {
	d := &DragData{}
	d.SetText("first")
	if got := d.Text(); got != "first" {
		t.Errorf("Text() = %q after SetText, want %q", got, "first")
	}

	// Overwrite existing text/plain.
	d.SetText("second")
	if got := d.Text(); got != "second" {
		t.Errorf("Text() = %q after second SetText, want %q", got, "second")
	}
	if len(d.Items) != 1 {
		t.Errorf("expected 1 item after overwrite, got %d", len(d.Items))
	}
}

func TestDragDataTextEmpty(t *testing.T) {
	d := &DragData{
		Items: []DragItem{{MIMEType: MIMEJSON, Data: "{}"}},
	}
	if got := d.Text(); got != "" {
		t.Errorf("Text() = %q, want empty string", got)
	}
}

func TestDragDataTypes(t *testing.T) {
	d := &DragData{
		Items: []DragItem{
			{MIMEType: MIMEText, Data: "a"},
			{MIMEType: MIMEJSON, Data: "b"},
			{MIMEType: MIMESortableKey, Data: "c"},
		},
	}
	types := d.Types()
	if len(types) != 3 {
		t.Fatalf("Types() length = %d, want 3", len(types))
	}
	if types[0] != MIMEText || types[1] != MIMEJSON || types[2] != MIMESortableKey {
		t.Errorf("Types() = %v, want [text/plain, application/json, application/x-lux-sortable-key]", types)
	}
}

func TestDragDataTypesNil(t *testing.T) {
	var d *DragData
	if types := d.Types(); types != nil {
		t.Errorf("nil DragData.Types() = %v, want nil", types)
	}
}

func TestNewDragData(t *testing.T) {
	d := NewDragData(MIMEJSON, `{"key":"val"}`)
	if len(d.Items) != 1 {
		t.Fatalf("NewDragData items = %d, want 1", len(d.Items))
	}
	if d.Items[0].MIMEType != MIMEJSON {
		t.Errorf("MIMEType = %q, want %q", d.Items[0].MIMEType, MIMEJSON)
	}
	if d.AllowedOps != DragOperationMove {
		t.Errorf("AllowedOps = %d, want DragOperationMove", d.AllowedOps)
	}
}

func TestNewTextDragData(t *testing.T) {
	d := NewTextDragData("hello world")
	if d.Text() != "hello world" {
		t.Errorf("Text() = %q, want %q", d.Text(), "hello world")
	}
	if !d.AllowedOps.Has(DragOperationMove) {
		t.Error("expected Move to be allowed")
	}
	if !d.AllowedOps.Has(DragOperationCopy) {
		t.Error("expected Copy to be allowed")
	}
}

func TestResolveOperation(t *testing.T) {
	all := DragOperationMove | DragOperationCopy | DragOperationLink

	tests := []struct {
		name    string
		allowed DragOperation
		mods    ModifierSet
		want    DragOperation
	}{
		{"no mods, all allowed → Move", all, 0, DragOperationMove},
		{"Ctrl → Copy", all, ModCtrl, DragOperationCopy},
		{"Shift → Move", all, ModShift, DragOperationMove},
		{"Ctrl+Shift → Link", all, ModCtrl | ModShift, DragOperationLink},
		{"Ctrl but only Move allowed → Move", DragOperationMove, ModCtrl, DragOperationMove},
		{"no mods, only Copy → Copy", DragOperationCopy, 0, DragOperationCopy},
		{"no mods, only Link → Link", DragOperationLink, 0, DragOperationLink},
		{"none allowed → None", DragOperationNone, 0, DragOperationNone},
		{"Ctrl+Shift, no Link → fallback", DragOperationMove | DragOperationCopy, ModCtrl | ModShift, DragOperationMove},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveOperation(tt.allowed, tt.mods)
			if got != tt.want {
				t.Errorf("ResolveOperation(%d, %d) = %d, want %d", tt.allowed, tt.mods, got, tt.want)
			}
		})
	}
}

func TestDragDataSetTextNil(t *testing.T) {
	var d *DragData
	d.SetText("should not panic") // should be a no-op
}
