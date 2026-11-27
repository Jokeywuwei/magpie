// Registration is the platform's own switch, not a setting of the window's:
// whether a stranger may make an account in the user portal once an
// administrator exists. It is off until an administrator turns it on, so a
// deployment never opens sign-up by having been left as it was. The first
// administrator is still made with `magpie user add --role admin`; that
// path is not registration.
package identity

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/filelock"
	"github.com/yetone/magpie/internal/settings"
)

// ErrRegistrationClosed is the user portal's sign-up when the administrator
// has not opened it.
var ErrRegistrationClosed = errors.New("registration is closed")

// RegistrationPath is the switch's file, beside the accounts and settings.
func RegistrationPath() string { return filepath.Join(settings.Dir(), "registration.json") }

var registrationMu = filelock.Mutex{Path: func() string { return RegistrationPath() + ".lock" }}

type registrationFile struct {
	Open bool `json:"open"`
}

// RegistrationOpen reports whether new accounts may register in the user
// portal. A missing file, and a file that can't be read, is closed: sign-up
// is never opened by an answer magpie can't read.
func RegistrationOpen() (bool, error) {
	b, err := os.ReadFile(RegistrationPath())
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var f registrationFile
	if err := json.Unmarshal(b, &f); err != nil {
		return false, err
	}
	return f.Open, nil
}

// SetRegistrationOpen opens or closes sign-up. The file is readable by its
// owner alone, as it names a choice about who may reach the platform.
func SetRegistrationOpen(open bool) error {
	registrationMu.Lock()
	defer registrationMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(RegistrationPath()), 0o700); err != nil {
		return err
	}
	if f, err := os.OpenFile(RegistrationPath(), os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
		_ = f.Close()
	}
	b, err := json.MarshalIndent(registrationFile{Open: open}, "", "  ")
	if err != nil {
		return err
	}
	if err := edit.WriteAtomic(RegistrationPath(), append(b, '\n')); err != nil {
		return err
	}
	return os.Chmod(RegistrationPath(), 0o600)
}
