//go:build !windows

package config

import (
	"fmt"
	"os"
	"syscall"
)

// OwnerGuard trusts files owned by TrustedUID (root in production) that no
// one else can write, and treats Privileged callers as allowed to change them.
type OwnerGuard struct {
	TrustedUID uint32
	Privileged func() bool
}

func defaultGuard() Guard {
	return OwnerGuard{TrustedUID: 0, Privileged: func() bool { return os.Geteuid() == 0 }}
}

func systemBoundaryDir() string {
	return "/etc/jira-cli/boundaries"
}

func (g OwnerGuard) IsPrivileged() bool {
	return g.Privileged != nil && g.Privileged()
}

func (g OwnerGuard) CheckTrusted(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("boundary path %s is a symlink", path)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot read owner of %s", path)
	}
	if stat.Uid != g.TrustedUID {
		return fmt.Errorf("boundary path %s is owned by uid %d, not uid %d", path, stat.Uid, g.TrustedUID)
	}
	if info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("boundary path %s is group- or world-writable", path)
	}
	return nil
}

// Protect is a no-op: ownership by root and 0644/0755 modes already stop
// unprivileged writes.
func (g OwnerGuard) Protect(string) error {
	return nil
}
