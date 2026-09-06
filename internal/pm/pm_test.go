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
	// git/gh/node must install everywhere; the rest may degrade to manual.
	for _, p := range platforms {
		for _, d := range Deps() {
			r := d.Recipe(p)
			if (d.Bin == "git" || d.Bin == "gh" || d.Bin == "node") && len(r) == 0 {
				t.Errorf("dep %q has no recipe for %+v", d.Name, p)
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
