package pm

import "testing"

// Every dep must document a manual fallback; recipes are validated per-OS below.
func TestDepsHaveManual(t *testing.T) {
	for _, d := range Deps() {
		if d.Manual == "" || d.Bin == "" {
			t.Errorf("dep %q missing bin/manual", d.Name)
		}
		if d.Recipe == nil {
			t.Errorf("dep %q missing recipe func", d.Name)
		}
	}
}

func TestRecipesKnownPlatforms(t *testing.T) {
	platforms := []Info{
		{OS: "darwin", PM: "brew"},
		{OS: "linux", PM: "apt"},
		{OS: "windows", PM: "winget"},
	}
	// git/gh must install everywhere; node intentionally has no apt recipe
	// (distro nodejs is too old for npx MCP servers — honest manual step).
	for _, p := range platforms {
		for _, d := range Deps() {
			r := d.Recipe(p)
			mustHave := d.Bin == "git" || d.Bin == "gh" ||
				(d.Bin == "node" && p.PM != "apt")
			if mustHave && len(r) == 0 {
				t.Errorf("dep %q has no recipe for %+v", d.Name, p)
			}
		}
	}
	for _, d := range Deps() {
		if d.Bin == "node" {
			if r := d.Recipe(Info{OS: "linux", PM: "apt"}); r != nil {
				t.Errorf("apt node recipe must be nil, got %v", r)
			}
		}
	}
}

func TestDetectLocal(t *testing.T) {
	info := Detect()
	if info.OS == "" || info.PM == "" {
		t.Errorf("Detect() = %+v, want OS and PM set", info)
	}
}
