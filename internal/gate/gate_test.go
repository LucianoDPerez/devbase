package gate

import (
	"testing"
)

// Any single FAIL must block the verdict; SKIP must never block.
func TestVerdict(t *testing.T) {
	blocked := Report{SHA: "abc", Results: []Result{
		{"go build+vet", Pass, "clean"},
		{"semgrep", Fail, "finding"},
	}}
	if blocked.Verdict() != Fail {
		t.Error("expected BLOCKED with one FAIL")
	}
	verified := Report{SHA: "abc", Results: []Result{
		{"go build+vet", Pass, "clean"},
		{"semgrep", Skip, "not installed"},
	}}
	if verified.Verdict() != Pass {
		t.Error("SKIP must not block VERIFIED")
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
	r := buildCheck(t.TempDir())
	if r.Status != Skip {
		t.Errorf("buildCheck(unknown) = %v, want SKIP", r.Status)
	}
}
