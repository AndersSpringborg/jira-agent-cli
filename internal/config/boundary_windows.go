//go:build windows

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// elevationGuard trusts files owned by Administrators or SYSTEM and lets
// only elevated processes change them.
type elevationGuard struct{}

func defaultGuard() Guard {
	return elevationGuard{}
}

func systemBoundaryDir() string {
	programData, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0)
	if err != nil {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "jira-cli", "boundaries")
}

func (elevationGuard) IsPrivileged() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

func (elevationGuard) CheckTrusted(path string) error {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := sd.Owner()
	if err != nil {
		return err
	}
	if !owner.IsWellKnown(windows.WinBuiltinAdministratorsSid) && !owner.IsWellKnown(windows.WinLocalSystemSid) {
		return fmt.Errorf("boundary path %s is owned by %s, not Administrators or SYSTEM", path, owner.String())
	}
	return nil
}

// Protect makes Administrators the owner and replaces inherited permissions
// so Administrators and SYSTEM have full control and users can only read.
func (elevationGuard) Protect(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	sddl := "O:BAD:P(A;;FA;;;BA)(A;;FA;;;SY)(A;;FR;;;BU)"
	if info.IsDir() {
		sddl = "O:BAD:P(A;OICI;FA;;;BA)(A;OICI;FA;;;SY)(A;OICI;FR;;;BU)"
	}
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return err
	}
	owner, _, err := sd.Owner()
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		owner, nil, dacl, nil)
}
