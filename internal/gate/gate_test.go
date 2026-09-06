package gate

import (
	"os"
	"path/filepath"
	"testing"
)

// Any single FAIL blocks; a required SKIP is INCOMPLETE, never a pass.
func TestVerdict(t *testing.T) {
	blocked := Report{Results: []Result{
		{Name: "go build+vet", Status: Pass, Required: true},
		{Name: "semgrep", Status: Fail, Detail: "finding", Required: true},
	}}
	if blocked.Verdict() != Fail {
		t.Error("expected BLOCKED with one FAIL")
	}
	incomplete := Report{Results: []Result{
		{Name: "go build+vet", Status: Pass, Required: true},
		{Name: "semgrep", Status: Skip, Detail: "not installed", Required: true},
	}}
	if incomplete.Verdict() != Incomplete {
		t.Error("expected INCOMPLETE with a required SKIP")
	}
	verified := Report{Results: []Result{
		{Name: "go build+vet", Status: Pass, Required: true},
		{Name: "playwright", Status: Skip, Detail: "no config", Required: false},
	}}
	if verified.Verdict() != Pass {
		t.Error("optional SKIP must not block VERIFIED")
	}
}

// Outside a git repo the SHA degrades gracefully instead of failing.
func TestGitSHANoRepo(t *testing.T) {
	if sha := gitSHA(t.TempDir()); sha != "nogit" {
		t.Errorf("gitSHA outside repo = %q, want nogit", sha)
	}
}

// Unknown stacks skip the build check instead of inventing one.
func TestBuildCheckUnknownStack(t *testing.T) {
	rs := buildChecks(t.TempDir())
	if len(rs) != 1 {
		t.Fatalf("buildChecks(unknown) = %v, want one result", rs)
	}
	if rs[0].Status != Skip || !rs[0].Required {
		t.Errorf("buildChecks(unknown) = %+v, want required SKIP", rs[0])
	}
}

// Playwright is conditional: no config means optional SKIP.
func TestPlaywrightCheckSkip(t *testing.T) {
	r := playwrightCheck(t.TempDir())
	if r.Status != Skip || r.Required {
		t.Errorf("playwrightCheck(no config) = %+v, want optional SKIP", r)
	}
}

// A missing toolchain is INCOMPLETE (cannot verify), never a false FAIL.
func TestMissingToolchainIncomplete(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // hide every binary: no go, git, semgrep…
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module x\n\ngo 1.25\n")
	rep := Run(dir)
	if rep.Verdict() != Incomplete {
		t.Errorf("no-toolchain verdict = %v, want INCOMPLETE", rep.Verdict())
	}
	for _, r := range rep.Results {
		if r.Status == Fail {
			t.Errorf("no-toolchain produced FAIL: %+v", r)
		}
	}
}

// A PHP project must not fall through to Python checks: it gets its own
// composer validation (or a required SKIP when composer is absent).
func TestComposerSplit(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "composer.json"), `{"name":"acme/app"}`+"\n")
	rs := buildChecks(dir)
	if len(rs) != 1 || rs[0].Name != "composer validate" || !rs[0].Required {
		t.Fatalf("composer project = %+v, want required composer validate", rs)
	}
	if rs[0].Status == Fail {
		t.Errorf("valid manifest must not FAIL: %+v", rs[0])
	}
}

// A dirty tree hashes differently from clean and flags dirty.
func TestWorkingTreeDirty(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, dir, "git", "init", "-q")
	mustRun(t, dir, "git", "config", "user.email", "t@t")
	mustRun(t, dir, "git", "config", "user.name", "t")
	mustWrite(t, filepath.Join(dir, "a.txt"), "v1")
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-qm", "one")
	if tree, dirty := workingTree(dir); tree != "clean" || dirty {
		t.Errorf("clean tree = %q dirty=%v, want clean/false", tree, dirty)
	}
	mustWrite(t, filepath.Join(dir, "a.txt"), "v2")
	tree, dirty := workingTree(dir)
	if tree == "clean" || !dirty {
		t.Errorf("dirty tree = %q dirty=%v, want hash/true", tree, dirty)
	}
}

// A green Go suite must PASS with the suite evidence attached.
func TestRunGreenGo(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module green\n\ngo 1.25\n")
	mustWrite(t, filepath.Join(dir, "add.go"), "package main\n\nfunc Add(a, b int) int { return a + b }\nfunc main() {}\n")
	mustWrite(t, filepath.Join(dir, "add_test.go"),
		"package main\nimport \"testing\"\nfunc TestAdd(t *testing.T) { if 1+1 != 2 { t.Fail() } }\n")
	rep := Run(dir)
	if rep.Verdict() != Pass {
		t.Errorf("green go suite verdict = %v, want PASS", rep.Verdict())
	}
	found := false
	for _, r := range rep.Results {
		if r.Name == "go test" && r.Status == Pass && r.Required {
			found = true
		}
	}
	if !found {
		t.Errorf("missing required go test PASS in %+v", rep.Results)
	}
}

// A red Go suite must BLOCK.
func TestRunRedGo(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module red\n\ngo 1.25\n")
	mustWrite(t, filepath.Join(dir, "add.go"), "package main\n\nfunc Add(a, b int) int { return a + b }\nfunc main() {}\n")
	mustWrite(t, filepath.Join(dir, "add_test.go"),
		"package main\nimport \"testing\"\nfunc TestAdd(t *testing.T) { t.Fatal(\"broken\") }\n")
	rep := Run(dir)
	if rep.Verdict() != Fail {
		t.Errorf("red go suite verdict = %v, want FAIL", rep.Verdict())
	}
}

// A broken Go module must BLOCK with evidence attached.
func TestRunBrokenGo(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module broken\n\ngo 1.25\n")
	mustWrite(t, filepath.Join(dir, "main.go"), "package main\nfunc main( {\n")
	rep := Run(dir)
	if rep.Verdict() != Fail {
		t.Errorf("broken go module verdict = %v, want FAIL", rep.Verdict())
	}
	if rep.WorkingTree == "" || len(rep.Toolchain) == 0 {
		t.Error("report missing tree hash or toolchain")
	}
}

func mustRun(t *testing.T, dir, bin string, args ...string) {
	t.Helper()
	var o outcome
	switch bin {
	case "git":
		o = gitOut(dir, args...)
	default:
		t.Fatalf("mustRun: unsupported %s", bin)
	}
	if o.code != 0 {
		t.Fatalf("%s %v: %s", bin, args, o.out)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
