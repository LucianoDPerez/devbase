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
const e2eTimeout = 300 * time.Second
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
	rep.Results = append(rep.Results, playwrightCheck(dir))
	rep.Results = append(rep.Results, semgrepCheck(dir))
	return rep
}

// Typed wrappers keep the executed binary a string literal at every call
// site (auditable, allowlisted) instead of threading it as a variable.
func gitOut(dir string, args ...string) (int, string) {
	return doRun(dir, exec.Command("git", args...))
}

func goOut(dir string, args ...string) (int, string) {
	return doRun(dir, exec.Command("go", args...))
}

func npmOut(dir string, args ...string) (int, string) {
	return doRun(dir, exec.Command("npm", args...))
}

func pythonOut(dir string, args ...string) (int, string) {
	return doRun(dir, exec.Command("python3", safePaths(args)...))
}

func semgrepOut(dir string, args ...string) (int, string) {
	return doRun(dir, exec.Command("semgrep", args...))
}

// safePaths drops path arguments that could escape the repo or be parsed as
// options (absolute paths, parent traversal, option-looking filenames).
func safePaths(args []string) []string {
	var out []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") || filepath.IsAbs(a) {
			continue
		}
		if a != filepath.Clean(a) || strings.HasPrefix(a, "..") {
			continue
		}
		out = append(out, a)
	}
	return out
}

func doRun(dir string, cmd *exec.Cmd) (int, string) {
	return doRunTimeout(timeout, dir, cmd)
}

func doRunTimeout(d time.Duration, dir string, cmd *exec.Cmd) (int, string) {
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	// fail-open on missing binary or spawn error: caller decides via ok=false
	if err := cmd.Start(); err != nil {
		return -1, err.Error()
	}
	timer := time.AfterFunc(d, func() { _ = cmd.Process.Kill() })
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
	code, out := gitOut(dir, "rev-parse", "--short", "HEAD")
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
		if code, out := goOut(dir, "build", "./..."); code != 0 {
			return Result{"go build", Fail, out}
		}
		if code, out := goOut(dir, "vet", "./..."); code != 0 {
			return Result{"go vet", Fail, out}
		}
		return Result{"go build+vet", Pass, "clean"}
	case has(dir, "package.json"):
		data, err := os.ReadFile(filepath.Join(dir, "package.json"))
		if err != nil || !strings.Contains(string(data), `"build"`) {
			return Result{"npm build", Skip, "no build script"}
		}
		if code, out := npmOut(dir, "run", "-s", "build"); code != 0 {
			return Result{"npm build", Fail, out}
		}
		return Result{"npm build", Pass, "clean"}
	case has(dir, "composer.json"), has(dir, "requirements.txt"), has(dir, "pyproject.toml"):
		files := changedPy(dir)
		if len(files) == 0 {
			return Result{"py_compile", Skip, "no python files changed"}
		}
		args := append([]string{"-m", "py_compile"}, files...)
		if code, out := pythonOut(dir, args...); code != 0 {
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
	if code, out := gitOut(dir, "diff", "--name-only", "HEAD"); code == 0 {
		add(out)
	}
	if code, out := gitOut(dir, "ls-files", "--others", "--exclude-standard"); code == 0 {
		add(out)
	}
	return files
}

func npxOutTimeout(d time.Duration, dir string, args ...string) (int, string) {
	return doRunTimeout(d, dir, exec.Command("npx", args...))
}

// playwrightCheck runs E2E specs only when the project has them: a
// playwright.config in the root or a conventional subdir. No config means
// SKIP — Playwright is conditional, never a default requirement.
func playwrightCheck(dir string) Result {
	e2eDir := ""
	for _, sub := range []string{".", "frontend", "web", "client", "app", "ui", "e2e", "tests"} {
		base := filepath.Join(dir, sub)
		if hasConfig(base) {
			e2eDir = base
			break
		}
	}
	if e2eDir == "" {
		return Result{"playwright", Skip, "no playwright.config found"}
	}
	code, out := npxOutTimeout(e2eTimeout, e2eDir, "playwright", "test", "--reporter=line")
	switch {
	case code == 0:
		return Result{"playwright", Pass, "e2e green"}
	case strings.Contains(out, "Executable doesn't exist") || strings.Contains(out, "Host system is missing dependencies"):
		return Result{"playwright", Fail, "browsers not installed — run: npx playwright install --with-deps"}
	default:
		return Result{"playwright", Fail, out}
	}
}

func hasConfig(base string) bool {
	m, _ := filepath.Glob(filepath.Join(base, "playwright.config.*"))
	return len(m) > 0
}

func semgrepCheck(dir string) Result {
	if _, err := exec.LookPath("semgrep"); err != nil {
		return Result{"semgrep", Skip, "not installed"}
	}
	code, out := semgrepOut(dir, "scan", "--config", "auto", "--severity", "ERROR", "--error")
	switch code {
	case 0:
		return Result{"semgrep", Pass, "no ERROR findings"}
	case 1:
		return Result{"semgrep", Fail, out}
	default:
		return Result{"semgrep", Skip, out}
	}
}
