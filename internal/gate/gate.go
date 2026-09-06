// Package gate runs deterministic verification and emits VERIFIED / BLOCKED /
// INCOMPLETE with evidence. SKIP on a required check is INCOMPLETE, never a
// pass. LLM review is advisory only and can never mark VERIFIED.
package gate

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/devbase/devbase/internal/detect"
)

const timeout = 120 * time.Second
const e2eTimeout = 300 * time.Second
const testTimeout = 300 * time.Second
const maxOut = 3500

// EvidenceSchema versions the machine-readable gate output.
const EvidenceSchema = "devbase.evidence.v1"

// Status is the outcome of a single check.
type Status string

const (
	Pass       Status = "PASS"
	Fail       Status = "FAIL"
	Skip       Status = "SKIP"
	Incomplete Status = "INCOMPLETE"
)

// Result is one check outcome with evidence.
type Result struct {
	Name      string `json:"name"`
	Status    Status `json:"status"`
	Detail    string `json:"detail"`
	Required  bool   `json:"required"`
	Command   string `json:"command"`
	ExitCode  int    `json:"exit_code"`
	DurationM int64  `json:"duration_ms"`
}

// Report is the full gate outcome for one snapshot.
type Report struct {
	SHA         string            `json:"commit"`
	WorkingTree string            `json:"working_tree"`
	Dirty       bool              `json:"dirty"`
	Stacks      []string          `json:"stacks"`
	Toolchain   map[string]string `json:"toolchain"`
	Results     []Result          `json:"checks"`
}

// Evidence is the machine-readable envelope for --json and artifacts.
type Evidence struct {
	Schema  string `json:"schema"`
	Verdict Status `json:"verdict"`
	Report
}

// Verdict is BLOCKED on any FAIL, INCOMPLETE on any required SKIP,
// VERIFIED only when every required check passed.
func (r Report) Verdict() Status {
	for _, res := range r.Results {
		if res.Status == Fail {
			return Fail
		}
	}
	for _, res := range r.Results {
		if res.Status == Skip && res.Required {
			return Incomplete
		}
	}
	return Pass
}

// Run executes the deterministic checks for dir.
func Run(dir string) Report {
	sha := gitSHA(dir)
	tree, dirty := workingTree(dir)
	rep := Report{
		SHA:         sha,
		WorkingTree: tree,
		Dirty:       dirty,
		Stacks:      detect.Detect(dir),
		Toolchain:   toolchain(),
	}
	rep.Results = append(rep.Results, buildChecks(dir)...)
	rep.Results = append(rep.Results, testChecks(dir)...)
	rep.Results = append(rep.Results, playwrightCheck(dir))
	rep.Results = append(rep.Results, semgrepCheck(dir))
	return rep
}

// outcome carries everything the evidence needs about one command run.
type outcome struct {
	code int
	out  string
	ms   int64
	cmd  string
}

func (o outcome) result(name string, required bool) Result {
	return Result{Name: name, Command: o.cmd, ExitCode: o.code, DurationM: o.ms, Required: required}
}

// Typed wrappers keep the executed binary a string literal at every call
// site (auditable, allowlisted) instead of threading it as a variable.
func gitOut(dir string, args ...string) outcome {
	return doRun(dir, "git", exec.Command("git", args...), args)
}

func goOut(dir string, args ...string) outcome {
	return doRun(dir, "go", exec.Command("go", args...), args)
}

func npmOut(dir string, args ...string) outcome {
	return doRun(dir, "npm", exec.Command("npm", args...), args)
}

// npmOutCI forces single-run mode so watch-mode runners (jest, vitest,
// react-scripts) execute once instead of hanging until the timeout.
func npmOutCI(dir string, args ...string) outcome {
	cmd := exec.Command("npm", args...)
	cmd.Env = append(os.Environ(), "CI=true")
	return doRunTimeout(testTimeout, dir, "npm", cmd, args)
}

func nodeOut(dir string, args ...string) outcome {
	return doRun(dir, "node", exec.Command("node", args...), args)
}

func pythonOut(dir string, args ...string) outcome {
	return doRun(dir, "python3", exec.Command("python3", safePaths(args)...), args)
}

func semgrepOut(dir string, args ...string) outcome {
	return doRun(dir, "semgrep", exec.Command("semgrep", args...), args)
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

func doRun(dir, bin string, cmd *exec.Cmd, args []string) outcome {
	return doRunTimeout(timeout, dir, bin, cmd, args)
}

func doRunTimeout(d time.Duration, dir, bin string, cmd *exec.Cmd, args []string) outcome {
	o := outcome{cmd: bin + " " + strings.Join(args, " "), code: -1}
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	start := time.Now()
	// fail-open on missing binary or spawn error: caller decides via code.
	if err := cmd.Start(); err != nil {
		o.out = err.Error()
		return o
	}
	timer := time.AfterFunc(d, func() { _ = cmd.Process.Kill() })
	err := cmd.Wait()
	o.ms = time.Since(start).Milliseconds()
	timer.Stop()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			o.code = ee.ExitCode()
		}
	} else {
		o.code = 0
	}
	s := out.String()
	if len(s) > maxOut {
		s = s[len(s)-maxOut:]
	}
	o.out = strings.TrimSpace(s)
	return o
}

func gitSHA(dir string) string {
	o := gitOut(dir, "rev-parse", "--short", "HEAD")
	if o.code != 0 {
		return "nogit"
	}
	return o.out
}

// workingTree returns "clean" or a short hash of the uncommitted changes
// (tracked diff plus untracked listing), so evidence never claims a commit
// SHA for code that isn't in it.
func workingTree(dir string) (string, bool) {
	porcelain := gitOut(dir, "status", "--porcelain")
	if porcelain.code != 0 {
		return "nogit", false
	}
	if strings.TrimSpace(porcelain.out) == "" {
		return "clean", false
	}
	diff := gitOut(dir, "diff", "HEAD")
	sum := sha256.Sum256([]byte(porcelain.out + "\n" + diff.out))
	return fmt.Sprintf("%.12x", sum), true
}

func has(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// toolchain records the exact tool versions behind a verdict, since the same
// PASS under different toolchains is not the same evidence.
func toolchain() map[string]string {
	tools := map[string][]string{
		"go":      {"version"},
		"node":    {"--version"},
		"npm":     {"--version"},
		"semgrep": {"--version"},
		"python3": {"--version"},
	}
	out := map[string]string{}
	for bin, args := range tools {
		if _, err := exec.LookPath(bin); err != nil {
			out[bin] = "missing"
			continue
		}
		// Literal dispatch keeps each binary auditable (see typed wrappers).
		var o outcome
		switch bin {
		case "go":
			o = goOut(".", args...)
		case "node":
			o = nodeOut(".", args...)
		case "npm":
			o = npmOut(".", args...)
		case "semgrep":
			o = semgrepOut(".", args...)
		case "python3":
			o = pythonOut(".", args...)
		}
		out[bin] = firstLine(o.out)
		if out[bin] == "" {
			out[bin] = "unknown"
		}
	}
	return out
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// buildChecks returns one result per build step so no evidence is lost.
func buildChecks(dir string) []Result {
	switch {
	case has(dir, "go.mod"):
		var out []Result
		o := goOut(dir, "build", "./...")
		rb := o.result("go build", true)
		if o.code != 0 {
			rb.Status, rb.Detail = Fail, o.out
			return []Result{rb}
		}
		rb.Status, rb.Detail = Pass, "clean"
		out = append(out, rb)
		o = goOut(dir, "vet", "./...")
		rv := o.result("go vet", true)
		if o.code != 0 {
			rv.Status, rv.Detail = Fail, o.out
		} else {
			rv.Status, rv.Detail = Pass, "clean"
		}
		return append(out, rv)
	case has(dir, "package.json"):
		data, err := os.ReadFile(filepath.Join(dir, "package.json"))
		if err != nil || !strings.Contains(string(data), `"build"`) {
			return []Result{{Name: "npm build", Status: Skip, Detail: "no build script", Required: true}}
		}
		o := npmOut(dir, "run", "-s", "build")
		r := o.result("npm build", true)
		if o.code != 0 {
			r.Status, r.Detail = Fail, o.out
			return []Result{r}
		}
		r.Status, r.Detail = Pass, "clean"
		return []Result{r}
	case has(dir, "composer.json"), has(dir, "requirements.txt"), has(dir, "pyproject.toml"):
		files := changedPy(dir)
		if len(files) == 0 {
			return []Result{{Name: "py_compile", Status: Skip, Detail: "no python files changed", Required: true}}
		}
		args := append([]string{"-m", "py_compile"}, files...)
		o := pythonOut(dir, args...)
		r := o.result("py_compile", true)
		if o.code != 0 {
			r.Status, r.Detail = Fail, o.out
			return []Result{r}
		}
		r.Status, r.Detail = Pass, "clean"
		return []Result{r}
	default:
		return []Result{{Name: "build", Status: Skip, Detail: "unknown stack", Required: true}}
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
	if o := gitOut(dir, "diff", "--name-only", "HEAD"); o.code == 0 {
		add(o.out)
	}
	if o := gitOut(dir, "ls-files", "--others", "--exclude-standard"); o.code == 0 {
		add(o.out)
	}
	return files
}

func npxOutTimeout(d time.Duration, dir string, args ...string) outcome {
	return doRunTimeout(d, dir, "npx", exec.Command("npx", args...), args)
}

// playwrightCheck runs E2E specs only when the project has them: a
// playwright.config in the root or a conventional subdir. No config means
// SKIP on an optional check — Playwright is conditional, never required.
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
		return Result{Name: "playwright", Status: Skip, Detail: "no playwright.config found", Required: false}
	}
	o := npxOutTimeout(e2eTimeout, e2eDir, "playwright", "test", "--reporter=line")
	r := o.result("playwright", false)
	switch {
	case o.code == 0:
		r.Status, r.Detail = Pass, "e2e green"
	case strings.Contains(o.out, "Executable doesn't exist") || strings.Contains(o.out, "Host system is missing dependencies"):
		r.Status, r.Detail = Fail, "browsers not installed — run: npx playwright install --with-deps"
	default:
		r.Status, r.Detail = Fail, o.out
	}
	return r
}

func hasConfig(base string) bool {
	m, _ := filepath.Glob(filepath.Join(base, "playwright.config.*"))
	return len(m) > 0
}

// testChecks runs the project's own test suite when one exists. A detected
// suite is required: missing runner or red tests block or incompletely verify.
// No suite at all is an optional SKIP — the gate cannot invent tests.
func testChecks(dir string) []Result {
	switch {
	case has(dir, "go.mod"):
		o := doRunTimeout(testTimeout, dir, "go", exec.Command("go", "test", "./..."), []string{"test", "./..."})
		return []Result{finalize(o, "go test", true)}
	case has(dir, "package.json"):
		if !npmHasScript(dir, "test") {
			return []Result{{Name: "npm test", Status: Skip, Detail: "no test script", Required: false}}
		}
		o := npmOutCI(dir, "run", "-s", "test")
		return []Result{finalize(o, "npm test", true)}
	case has(dir, "composer.json"):
		bin := phpTestBinary(dir)
		if bin == "" {
			if phpSuiteFiles(dir) {
				return []Result{{Name: "phpunit", Status: Skip, Detail: "suite found but no phpunit/pest binary — run composer install", Required: true}}
			}
			return []Result{{Name: "phpunit", Status: Skip, Detail: "no test suite detected", Required: false}}
		}
		o := doRunTimeout(testTimeout, dir, "php", exec.Command("php", bin), []string{bin})
		return []Result{finalize(o, "phpunit", true)}
	case has(dir, "requirements.txt"), has(dir, "pyproject.toml"):
		if !pySuite(dir) {
			return []Result{{Name: "pytest", Status: Skip, Detail: "no test suite detected", Required: false}}
		}
		if o := pythonOut(dir, "-c", "import pytest"); o.code != 0 {
			return []Result{{Name: "pytest", Status: Skip, Detail: "pytest not installed — pip install pytest", Required: true}}
		}
		o := doRunTimeout(testTimeout, dir, "python3", exec.Command("python3", "-m", "pytest", "-q"), []string{"-m", "pytest", "-q"})
		return []Result{finalize(o, "pytest", true)}
	case has(dir, "Cargo.toml"):
		o := doRunTimeout(testTimeout, dir, "cargo", exec.Command("cargo", "test", "--quiet"), []string{"test", "--quiet"})
		return []Result{finalize(o, "cargo test", true)}
	case has(dir, "pom.xml"), has(dir, "build.gradle"), has(dir, "build.gradle.kts"):
		bin, args := javaTestRunner(dir)
		var o outcome
		switch bin {
		case "./mvnw":
			o = doRunTimeout(testTimeout, dir, "./mvnw", exec.Command("./mvnw", args...), args)
		case "./gradlew":
			o = doRunTimeout(testTimeout, dir, "./gradlew", exec.Command("./gradlew", args...), args)
		case "mvn":
			o = doRunTimeout(testTimeout, dir, "mvn", exec.Command("mvn", args...), args)
		case "gradle":
			o = doRunTimeout(testTimeout, dir, "gradle", exec.Command("gradle", args...), args)
		default:
			return []Result{{Name: "java test", Status: Skip, Detail: "no maven/gradle runner found", Required: true}}
		}
		return []Result{finalize(o, "java test", true)}
	case has(dir, "Gemfile"):
		if !has(dir, "spec") {
			return []Result{{Name: "rspec", Status: Skip, Detail: "no test suite detected", Required: false}}
		}
		o := doRunTimeout(testTimeout, dir, "bundle", exec.Command("bundle", "exec", "rspec"), []string{"exec", "rspec"})
		return []Result{finalize(o, "rspec", true)}
	case globCSProj(dir):
		o := doRunTimeout(testTimeout, dir, "dotnet", exec.Command("dotnet", "test", "--nologo", "-v", "q"), []string{"test", "--nologo", "-v", "q"})
		return []Result{finalize(o, "dotnet test", true)}
	default:
		return []Result{{Name: "tests", Status: Skip, Detail: "no test suite detected", Required: false}}
	}
}

// finalize turns a command outcome into a PASS/FAIL result.
func finalize(o outcome, name string, required bool) Result {
	r := o.result(name, required)
	if o.code != 0 {
		r.Status, r.Detail = Fail, o.out
		return r
	}
	r.Status, r.Detail = Pass, "suite green"
	return r
}

func npmHasScript(dir, script string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return false
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return false
	}
	_, ok := pkg.Scripts[script]
	return ok
}

// phpTestBinary returns the vendored test runner when present.
func phpTestBinary(dir string) string {
	for _, b := range []string{"vendor/bin/pest", "vendor/bin/phpunit"} {
		if has(dir, b) {
			return b
		}
	}
	return ""
}

func phpSuiteFiles(dir string) bool {
	if globAny(dir, []string{"phpunit.xml", "phpunit.xml.dist", "phpunit.dist.xml"}) {
		return true
	}
	m, _ := filepath.Glob(filepath.Join(dir, "tests", "*Test.php"))
	return len(m) > 0
}

func pySuite(dir string) bool {
	if globAny(dir, []string{"pytest.ini", "tox.ini", "conftest.py", "test_*.py"}) {
		return true
	}
	for _, sub := range []string{"tests", "test"} {
		m, _ := filepath.Glob(filepath.Join(dir, sub, "*.py"))
		if len(m) > 0 {
			return true
		}
	}
	return false
}

// javaTestRunner prefers wrappers over system installs.
func javaTestRunner(dir string) (string, []string) {
	switch {
	case has(dir, "mvnw"):
		return "./mvnw", []string{"-q", "test"}
	case has(dir, "gradlew"):
		return "./gradlew", []string{"test", "--quiet"}
	case lookPath("mvn"):
		return "mvn", []string{"-q", "test"}
	case lookPath("gradle"):
		return "gradle", []string{"test", "--quiet"}
	}
	return "", nil
}

func globCSProj(dir string) bool {
	if m, _ := filepath.Glob(filepath.Join(dir, "*.csproj")); len(m) > 0 {
		return true
	}
	if m, _ := filepath.Glob(filepath.Join(dir, "*.sln")); len(m) > 0 {
		return true
	}
	return false
}

func globAny(dir string, patterns []string) bool {
	for _, p := range patterns {
		if m, _ := filepath.Glob(filepath.Join(dir, p)); len(m) > 0 {
			return true
		}
	}
	return false
}

func lookPath(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

func semgrepCheck(dir string) Result {
	if _, err := exec.LookPath("semgrep"); err != nil {
		return Result{Name: "semgrep", Status: Skip, Detail: "not installed", Required: true}
	}
	o := semgrepOut(dir, "scan", "--config", "auto", "--severity", "ERROR", "--error")
	r := o.result("semgrep", true)
	switch o.code {
	case 0:
		r.Status, r.Detail = Pass, "no ERROR findings"
	case 1:
		r.Status, r.Detail = Fail, o.out
	default:
		r.Status, r.Detail = Skip, o.out
	}
	return r
}
