package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// IsRunning reports whether a process with EXACT comm name is alive.
// FIX vs Copilot version: uses exact match (==), not strings.Contains,
// so binary "torstack" no longer false-positives as "tor".
// Strategy: /proc scan -> pgrep -x -> ps exact match.
func IsRunning(name string) (bool, error) {
	if pid, err := PidForName(name); err == nil && pid > 0 {
		return true, nil
	} else if err != nil && !os.IsNotExist(err) {
		// PidForName returns "not found" as plain error; fall through
		// only hard IO errors propagate, but keep it simple:
		if !strings.Contains(err.Error(), "not found") {
			return false, err
		}
	}
	return false, nil
}

// PidForName returns the first PID whose comm equals name exactly.
func PidForName(name string) (int, error) {
	// 1) /proc scan (Linux + Termux, no external binary needed)
	if pid, err := pidViaProc(name); err == nil && pid > 0 {
		return pid, nil
	}
	// 2) pgrep -x (exact)
	if path, err := exec.LookPath("pgrep"); err == nil {
		out, err := exec.Command(path, "-x", name).Output()
		if err == nil {
			for _, f := range strings.Fields(string(out)) {
				if pid, err := strconv.Atoi(f); err == nil {
					return pid, nil
				}
			}
		}
	}
	// 3) ps exact match on comm column
	out, err := exec.Command("ps", "-eo", "pid,comm").Output()
	if err != nil {
		return 0, fmt.Errorf("process lookup failed for %s: %w", name, err)
	}
	self := os.Getpid()
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		if pid == self {
			continue // never match ourselves
		}
		comm := filepath.Base(fields[1])
		if comm == name {
			return pid, nil
		}
	}
	return 0, fmt.Errorf("process %s not found", name)
}

func pidViaProc(name string) (int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	self := os.Getpid()
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if pid == self {
			continue
		}
		commBytes, err := os.ReadFile(filepath.Join("/proc", e.Name(), "comm"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(commBytes)) == name {
			return pid, nil
		}
	}
	return 0, fmt.Errorf("process %s not found", name)
}

// StopProcess sends SIGTERM. FIX: returns error when not found
// instead of silently returning nil.
func StopProcess(name string) error {
	pid, err := PidForName(name)
	if err != nil {
		return err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}

func SignalProcess(name string, sig syscall.Signal) error {
	pid, err := PidForName(name)
	if err != nil {
		return err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(sig)
}

// StartDaemon launches binary detached (setsid) with output to logDir/<name>.log.
// FIX vs Copilot cmd.Start(): child survives parent exit, survives Termux session.
func StartDaemon(logDir, name, binary string, args ...string) error {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}
	logPath := filepath.Join(logDir, name+".log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	cmd := exec.Command(binary, args...)
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}
