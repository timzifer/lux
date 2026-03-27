package uitest

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/timzifer/lux/draw"
)

var update = flag.Bool("update", false, "update golden files")

// AssertScene serializes the scene and compares it against the golden file at
// goldenPath. If the -update flag is set, the golden file is written/overwritten
// instead of compared.
//
// goldenPath is relative to the test's working directory (typically the package
// directory). Convention: "testdata/<name>.golden".
func AssertScene(t *testing.T, scene draw.Scene, goldenPath string) {
	t.Helper()

	got := SerializeScene(scene)

	if *update {
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create testdata dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
		t.Logf("updated golden file: %s", goldenPath)
		return
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("golden file %s does not exist — run with -update to create it", goldenPath)
		}
		t.Fatalf("read golden file: %v", err)
	}
	want := string(wantBytes)

	if got == want {
		return
	}

	diff := DiffScenes(got, want)
	t.Errorf("scene does not match golden file %s:\n%s", goldenPath, diff)
}
