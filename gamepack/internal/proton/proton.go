package proton

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gamepack/internal/cfg"
)

// LaunchGame asks the running Steam instance to launch a (non-Steam) shortcut.
// It uses steam://rungameid which Steam forwards to the master instance via
// ~/.steam/steam.pipe, so it works from an SSH session even in game mode.
func LaunchGame(ctx context.Context, c *cfg.Config, appid uint32) error {
	cmd := exec.CommandContext(ctx, c.SteamBin, "steam://rungameid/"+fmt.Sprintf("%d", appid))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("steam launch failed (%v): %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// SteamRunning reports whether a Steam master instance is alive.
func SteamRunning(c *cfg.Config) bool {
	pidData, err := os.ReadFile("/var/home/jevonx/.steam/steam.pid")
	if err != nil {
		return false
	}
	pid := strings.TrimSpace(string(pidData))
	if pid == "" {
		return false
	}
	if _, err := os.Stat("/proc/" + pid); err != nil {
		return false
	}
	if _, err := os.Stat("/var/home/jevonx/.steam/steam.pipe"); err != nil {
		return false
	}
	return true
}

// WaitPrefixBuilt polls until compatdata/<appid>/pfx/creation_sync_guard exists.
func WaitPrefixBuilt(ctx context.Context, c *cfg.Config, appid uint32, timeout time.Duration, onStatus func(string)) error {
	guard := filepath.Join(c.PrefixDir(appid), "pfx", "creation_sync_guard")
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(guard); err == nil {
			return nil
		}
		if onStatus != nil {
			onStatus("waiting for prefix build…")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for prefix %d", appid)
		}
		if err := sleepCtx(ctx, 2*time.Second); err != nil {
			return err
		}
	}
}

// IsRunning reports whether the game appears to be running. Primary signal is
// a live (non-zombie) process matching the game exe name (Proton/Wine names
// its process after the exe); fallback is the flock state of pfx.lock, which
// some games hold while running (and which is NOT reliable for others).
func IsRunning(c *cfg.Config, appid uint32, exeName string) bool {
	if exeName != "" {
		out, err := exec.Command(c.PgrepBin, "-f", exeName).Output()
		if err == nil {
			for _, pid := range strings.Fields(string(out)) {
				if pidAlive(pid) {
					return true
				}
			}
		}
	}
	lock := filepath.Join(c.PrefixDir(appid), "pfx.lock")
	if _, err := os.Stat(lock); err != nil {
		return false
	}
	cmd := exec.Command(c.FlockBin, "-n", lock, "-c", "true")
	if err := cmd.Run(); err != nil {
		// exit 1 = lock held = running
		return true
	}
	return false
}

// pidAlive reports whether pid is a live (non-zombie) process.
func pidAlive(pid string) bool {
	data, err := os.ReadFile("/proc/" + pid + "/stat")
	if err != nil {
		return false
	}
	state := statState(string(data))
	return state != 'Z' && state != 'X'
}

// statState extracts the process state character from /proc/<pid>/stat
// (the field after the comm field, which may itself contain spaces/parens).
func statState(stat string) byte {
	i := strings.LastIndex(stat, ")")
	if i < 0 || i+3 > len(stat) {
		return 0
	}
	return stat[i+2]
}

// WaitGameExit polls until the game is no longer running. Requires two
// consecutive negative polls so transient gaps (launcher restarts) don't
// report an early exit.
func WaitGameExit(ctx context.Context, c *cfg.Config, appid uint32, exeName string, timeout time.Duration, onStatus func(string)) error {
	deadline := time.Now().Add(timeout)
	quiet := 0
	for {
		if !IsRunning(c, appid, exeName) {
			quiet++
			if quiet >= 2 {
				return nil
			}
		} else {
			quiet = 0
		}
		if onStatus != nil {
			onStatus("game running — close it when you reach the main menu")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for game %d to close", appid)
		}
		if err := sleepCtx(ctx, 2*time.Second); err != nil {
			return err
		}
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
