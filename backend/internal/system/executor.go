package system

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// AllowedCommands acts as a strict whitelist of binaries that CommandExecutor is authorized to execute.
var AllowedCommands = map[string]bool{
	"systemctl":         true,
	"journalctl":        true,
	"apt-get":           true,
	"apt":               true,
	"dpkg":              true,
	"dnf":               true,
	"yum":               true,
	"rpm":               true,
	"clickhouse-server": true,
	"clickhouse-client": true,
	"clickhouse":        true,
	"curl":              true,
	"wget":              true,
	"gpg":               true,
	"install":           true,
	"mkdir":             true,
	"chown":             true,
	"chmod":             true,
	"tar":               true,
	"cat":               true,
	"uname":             true,
	"df":                true,
	"ls":                true,
	"free":              true,
	"sh":                true,
	"bash":              true,
}

type CommandExecutor struct {
	mu sync.Mutex
}

var DefaultExecutor = &CommandExecutor{}

// OutputCallback receives streamed output line by line
type OutputCallback func(line string)

func (e *CommandExecutor) isAllowed(bin string) bool {
	// Extract base binary name
	parts := strings.Split(bin, "/")
	base := parts[len(parts)-1]
	partsWin := strings.Split(base, "\\")
	base = partsWin[len(partsWin)-1]
	base = strings.TrimSuffix(base, ".exe")
	return AllowedCommands[base]
}

// Execute runs a command with a timeout and returns standard combined output
func (e *CommandExecutor) Execute(ctx context.Context, name string, args ...string) (string, error) {
	if !e.isAllowed(name) {
		return "", fmt.Errorf("command '%s' is not in the security whitelist", name)
	}

	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ExecuteStream runs a command and streams output lines via callback in real-time
func (e *CommandExecutor) ExecuteStream(ctx context.Context, callback OutputCallback, name string, args ...string) error {
	if !e.isAllowed(name) {
		err := fmt.Errorf("command '%s' is not in the security whitelist", name)
		if callback != nil {
			callback(fmt.Sprintf("[Security Alert] %v", err))
		}
		return err
	}

	cmd := exec.CommandContext(ctx, name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	scan := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if callback != nil {
				callback(line)
			}
		}
	}

	wg.Add(2)
	go scan(stdout)
	go scan(stderr)

	wg.Wait()
	return cmd.Wait()
}

// CheckBinaryInstalled checks if a binary is in PATH
func CheckBinaryInstalled(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// IsLinux checks if current platform is Linux
func IsLinux() bool {
	return runtime.GOOS == "linux"
}
