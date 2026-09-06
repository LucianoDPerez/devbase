// Package pm detects the OS package manager and knows how to install every
// DevBase dependency on each platform. Anything without a reliable recipe
// degrades to a manual instruction instead of failing.
package pm

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Info describes the install environment.
type Info struct {
	OS string // darwin, linux, windows
	PM string // brew, apt, dnf, pacman, winget, none
}

// Detect returns the OS and the first available package manager.
func Detect() Info {
	info := Info{OS: runtime.GOOS}
	switch info.OS {
	case "darwin":
		if look("brew") {
			info.PM = "brew"
		}
	case "windows":
		if look("winget") {
			info.PM = "winget"
		}
	case "linux":
		id := osReleaseID()
		switch {
		case look("apt") && (id == "" || id == "debian" || id == "ubuntu" || id == "linuxmint" || id == "pop"):
			info.PM = "apt"
		case look("dnf"):
			info.PM = "dnf"
		case look("pacman"):
			info.PM = "pacman"
		case look("apt"):
			info.PM = "apt"
		}
	}
	if info.PM == "" {
		info.PM = "none"
	}
	return info
}

// Dep is one external binary DevBase can install or point to.
type Dep struct {
	Name   string
	Bin    string
	Manual string
	// Recipe returns the install argv for the detected environment,
	// or nil when there is no reliable automated recipe.
	Recipe func(info Info) []string
}

// Deps lists every external dependency in install order.
func Deps() []Dep {
	return []Dep{
		{Name: "git", Bin: "git", Manual: "https://git-scm.com/downloads", Recipe: func(info Info) []string {
			switch info.PM {
			case "brew":
				return []string{"brew", "install", "git"}
			case "apt":
				return []string{"sudo", "apt", "install", "-y", "git"}
			case "dnf":
				return []string{"sudo", "dnf", "install", "-y", "git"}
			case "pacman":
				return []string{"sudo", "pacman", "-S", "--noconfirm", "git"}
			case "winget":
				return []string{"winget", "install", "--id", "Git.Git", "-e", "--silent"}
			}
			return nil
		}},
		{Name: "GitHub CLI", Bin: "gh", Manual: "https://cli.github.com", Recipe: func(info Info) []string {
			switch info.PM {
			case "brew":
				return []string{"brew", "install", "gh"}
			case "apt":
				return []string{"sudo", "apt", "install", "-y", "gh"}
			case "dnf":
				return []string{"sudo", "dnf", "install", "-y", "gh"}
			case "pacman":
				return []string{"sudo", "pacman", "-S", "--noconfirm", "github-cli"}
			case "winget":
				return []string{"winget", "install", "--id", "GitHub.cli", "-e", "--silent"}
			}
			return nil
		}},
		{Name: "Node.js (for npx-based MCP servers)", Bin: "node", Manual: "https://nodejs.org", Recipe: func(info Info) []string {
			switch info.PM {
			case "brew":
				return []string{"brew", "install", "node"}
			case "apt":
				return []string{"sudo", "apt", "install", "-y", "nodejs", "npm"}
			case "dnf":
				return []string{"sudo", "dnf", "install", "-y", "nodejs", "npm"}
			case "pacman":
				return []string{"sudo", "pacman", "-S", "--noconfirm", "nodejs", "npm"}
			case "winget":
				return []string{"winget", "install", "--id", "OpenJS.NodeJS.LTS", "-e", "--silent"}
			}
			return nil
		}},
		{Name: "Semgrep", Bin: "semgrep", Manual: "pipx install semgrep", Recipe: func(info Info) []string {
			if info.PM == "brew" {
				return []string{"brew", "install", "semgrep"}
			}
			if look("pipx") {
				return []string{"pipx", "install", "semgrep"}
			}
			return nil
		}},
		{Name: "Engram", Bin: "engram", Manual: "https://github.com/Gentleman-Programming/engram/blob/main/docs/INSTALLATION.md", Recipe: func(info Info) []string {
			if info.PM == "brew" {
				return []string{"brew", "install", "gentleman-programming/tap/engram"}
			}
			return nil
		}},
		{Name: "codebase-memory-mcp", Bin: "codebase-memory-mcp", Manual: "https://github.com/DeusData/codebase-memory-mcp#quick-start", Recipe: func(info Info) []string {
			switch info.OS {
			case "darwin", "linux":
				return []string{"bash", "-c", "curl -fsSL https://raw.githubusercontent.com/DeusData/codebase-memory-mcp/main/install.sh | bash"}
			case "windows":
				return []string{"powershell", "-ExecutionPolicy", "Bypass", "-Command", "irm https://raw.githubusercontent.com/DeusData/codebase-memory-mcp/main/install.ps1 | iex"}
			}
			return nil
		}},
	}
}

// Missing returns the deps whose binaries are not on PATH.
func Missing() []Dep {
	var out []Dep
	for _, d := range Deps() {
		if !look(d.Bin) {
			out = append(out, d)
		}
	}
	return out
}

func look(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

func osReleaseID() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(line, "ID="); ok {
			return strings.Trim(strings.ToLower(v), `"`)
		}
	}
	return ""
}
