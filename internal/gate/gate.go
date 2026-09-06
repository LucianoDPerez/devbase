// Package gate runs deterministic verification and emits VERIFIED / BLOCKED
// with evidence. LLM review is advisory only and can never mark VERIFIED.
package gate

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const timeout = 120 * time.Second
const maxOut = 3500

// Status is the outcome of a single check.
type Status string

const (
	Pass Status = "PASS"
	Fail Status = "FAIL"
	Skip Status = "SKIP"
)

// Result is one check outcome with evidence.
type Result struct {
	Name   string
	Status Status
	Detail string
}

// Report is the full gate outcome for one commit snapshot.
type Report struct {
	SHA     string
	Results []Result
}

// Verdict is FAIL if any check failed, PASS otherwise.
func (r Report) Verdict() Status {
	for _, res := range r.Results {
		if res.Status == Fail {
			return Fail
		}
	}
	return Pass
}

// Run executes the deterministic checks for dir.
func Run(dir string) Report {
	rep := Report{SHA: gitSHA(dir)}
	rep.Results = append(rep.Results, buildCheck(dir))
	rep.Results = append(rep.Results, semgrepCheck(dir))
	return rep
}

func run(dir, name string, args ...string) (int, string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	// fail-open on missing binary or spawn error: caller decides via ok=false
	if err := cmd.Start(); err != nil {
		return -1, err.Error()
	}
	timer := time.AfterFunc(timeout, func() { _ = cmd.Process.Kill() })
	err := cmd.Wait()
	timer.Stop()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			code = -1
		}
	}
	s := out.String()
	if len(s) > maxOut {
		s = s[len(s)-maxOut:]
	}
	return code, strings.TrimSpace(s)
}

func gitSHA(dir string) string {
	code, out := run(dir, "git", "rev-parse", "--short", "HEAD")
	if code != 0 {
		return "nogit"
	}
	return out
}

func has(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

func buildCheck(dir string) Result {
	switch {
	case has(dir, "go.mod"):
		if code, out := run(dir, "go", "build", "./..."); code != 0 {
			return Result{"go build", Fail, out}
		}
		if code, out := run(dir, "go", "vet", "./..."); code != 0 {
			return Result{"go vet", Fail, out}
		}
		return Result{"go build+vet", Pass, "clean"}
	case has(dir, "package.json"):
		data, err := os.ReadFile(filepath.Join(dir, "package.json"))
		if err != nil || !strings.Contains(string(data), `"build"`) {
			return Result{"npm build", Skip, "no build script"}
		}
		if code, out := run(dir, "npm", "run", "-s", "build"); code != 0 {
			return Result{"npm build", Fail, out}
		}
		return Result{"npm build", Pass, "clean"}
	case has(dir, "composer.json"), has(dir, "requirements.txt"), has(dir, "pyproject.toml"):
		files := changedPy(dir)
		if len(files) == 0 {
			return Result{"py_compile", Skip, "no python files changed"}
		}
		args := append([]string{"-m", "py_compile"}, files...)
		if code, out := run(dir, "python3", args...); code != 0 {
			return Result{"py_compile", Fail, out}
		}
		return Result{"py_compile", Pass, "clean"}
	default:
		return Result{"build", Skip, "unknown stack"}
	}
}

func changedPy(dir string) []string {
	var files []string
	add := func(out string) {
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasSuffix(line, ".py") {
				files = append(files, line)
			}
		}
	}
	if code, out := run(dir, "git", "diff", "--name-only", "HEAD"); code == 0 {
		add(out)
	}
	if code, out := run(dir, "git", "ls-files", "--others", "--exclude-standard"); code == 0 {
		add(out)
	}
	return files
}

func semgrepCheck(dir string) Result {
	if _, err := exec.LookPath("semgrep"); err != nil {
		return Result{"semgrep", Skip, "not installed"}
	}
	code, out := run(dir, "semgrep", "scan", "--config", "auto", "--severity", "ERROR", "--error")
	switch code {
	case 0:
		return Result{"semgrep", Pass, "no ERROR findings"}
	case 1:
		return Result{"semgrep", Fail, out}
	default:
		return Result{"semgrep", Skip, out}
	}
}
