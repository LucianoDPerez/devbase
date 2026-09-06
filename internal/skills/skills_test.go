package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every embedded skill must install non-empty SKILL.md files with a
// name/description frontmatter the agents can route on.
func TestInstall(t *testing.T) {
	if len(Names()) == 0 {
		t.Fatal("no skills embedded")
	}
	dir := t.TempDir()
	installed, err := Install(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 2*len(Names()) {
		t.Errorf("installed %d files, want 2 per skill", len(installed))
	}
	for _, p := range installed {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("unreadable %s: %v", p, err)
			continue
		}
		body := string(data)
		if !strings.Contains(body, "name:") || !strings.Contains(body, "description:") {
			t.Errorf("%s missing routing frontmatter", p)
		}
		if filepath.Base(p) != "SKILL.md" {
			t.Errorf("unexpected file %s", p)
		}
	}
}
