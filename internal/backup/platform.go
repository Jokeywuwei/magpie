// Platform state in a backup: the accounts, the sessions, the open-model
// policy and the registration switch. These are the files the user portal
// and the gateway's owner check read; without them a restored caller-keys
// file has keys whose owners no longer exist.
package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/filelock"
	"github.com/yetone/magpie/internal/identity"
	"github.com/yetone/magpie/internal/modelaccess"
)

// Platform is the portal's own state as it was on disk. The bytes are the
// files verbatim: their schemas stay the packages' own, and a restore puts
// back exactly what a running magpie wrote.
type Platform struct {
	Users        []byte `json:"users,omitempty"`
	Sessions     []byte `json:"sessions,omitempty"`
	ModelAccess  []byte `json:"modelAccess,omitempty"`
	Registration []byte `json:"registration,omitempty"`
}

// maxPlatformFile keeps a damaged or hostile local file from becoming an
// enormous backup in memory. The real files are kilobytes.
const maxPlatformFile = 64 << 20

// collectPlatform reads the four files, missing ones left nil. A file that
// exists but can't be read or isn't the expected JSON stops the backup: a
// backup that carries "no accounts" would take them all away on restore.
func collectPlatform() (*Platform, error) {
	p := &Platform{}
	one := false
	for _, f := range []struct {
		path string
		to   *[]byte
	}{
		{identity.Path(), &p.Users},
		{identity.SessionsPath(), &p.Sessions},
		{modelaccess.Path(), &p.ModelAccess},
		{identity.RegistrationPath(), &p.Registration},
	} {
		data, err := readPlatformFile(f.path)
		if err != nil {
			return nil, err
		}
		if data == nil {
			continue
		}
		*f.to = data
		one = true
	}
	if !one {
		return nil, nil
	}
	if err := validatePlatform(p); err != nil {
		return nil, err
	}
	return p, nil
}

func readPlatformFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if info.Size() > maxPlatformFile {
		return nil, fmt.Errorf("%s is over %d bytes", filepath.Base(path), maxPlatformFile)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// validatePlatform checks the shapes the four readers expect, before
// anything is written. It never uses a name from the backup: the target
// paths below are the platform's own.
func validatePlatform(p *Platform) error {
	if len(p.Users) > 0 {
		var users []json.RawMessage
		if err := json.Unmarshal(p.Users, &users); err != nil {
			return fmt.Errorf("the backup's accounts don't read: %w", err)
		}
	}
	if len(p.Sessions) > 0 {
		var sessions map[string]json.RawMessage
		if err := json.Unmarshal(p.Sessions, &sessions); err != nil {
			return fmt.Errorf("the backup's sessions don't read: %w", err)
		}
	}
	if len(p.ModelAccess) > 0 {
		var policy struct {
			Models map[string]json.RawMessage `json:"models"`
		}
		if err := json.Unmarshal(p.ModelAccess, &policy); err != nil || policy.Models == nil {
			return errors.New("the backup's model access doesn't read")
		}
	}
	if len(p.Registration) > 0 {
		var reg struct {
			Open *bool `json:"open"`
		}
		if err := json.Unmarshal(p.Registration, &reg); err != nil || reg.Open == nil {
			return errors.New("the backup's registration switch doesn't read")
		}
	}
	return nil
}

type platformFile struct {
	path string
	data []byte
}

type platformSnapshot struct {
	path   string
	data   []byte
	mode   fs.FileMode
	exists bool
}

// writePlatform is writePlatformFile, a variable so a test can stop a
// restore in the middle and see the earlier files put back.
var writePlatform = writePlatformFile

// restorePlatform writes the platform files as one change: all four are
// validated first, and if any write fails the ones already written are put
// back as they were. The locks are the same files the identity and model
// access packages use, so a running page doesn't write between them.
func restorePlatform(p *Platform) error {
	if p == nil {
		return nil
	}
	files := []platformFile{
		{identity.Path(), p.Users},
		{identity.SessionsPath(), p.Sessions},
		{modelaccess.Path(), p.ModelAccess},
		{identity.RegistrationPath(), p.Registration},
	}
	any := false
	for _, f := range files {
		if len(f.data) > 0 {
			any = true
		}
	}
	if !any {
		return nil
	}
	if err := validatePlatform(p); err != nil {
		return err
	}

	var locks []*filelock.Mutex
	for _, path := range []string{identity.Path() + ".lock", modelaccess.Path() + ".lock", identity.RegistrationPath() + ".lock"} {
		path := path
		m := &filelock.Mutex{Path: func() string { return path }}
		m.Lock()
		locks = append(locks, m)
	}
	defer func() {
		for i := len(locks) - 1; i >= 0; i-- {
			locks[i].Unlock()
		}
	}()

	snaps := make([]platformSnapshot, 0, len(files))
	for _, f := range files {
		s, err := snapshotFile(f.path)
		if err != nil {
			return err
		}
		snaps = append(snaps, s)
	}
	for i, f := range files {
		if len(f.data) == 0 {
			continue
		}
		if err := writePlatform(f.path, f.data); err != nil {
			if rbErr := rollbackPlatform(snaps[:i+1]); rbErr != nil {
				return fmt.Errorf("restoring platform state: %w; rolling back: %v", err, rbErr)
			}
			return fmt.Errorf("restoring platform state: %w", err)
		}
	}
	return nil
}

func snapshotFile(path string) (platformSnapshot, error) {
	st, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return platformSnapshot{path: path}, nil
	}
	if err != nil {
		return platformSnapshot{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return platformSnapshot{}, err
	}
	return platformSnapshot{path: path, data: b, mode: st.Mode().Perm(), exists: true}, nil
}

func rollbackPlatform(snaps []platformSnapshot) error {
	var first error
	for i := len(snaps) - 1; i >= 0; i-- {
		s := snaps[i]
		if !s.exists {
			if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) && first == nil {
				first = err
			}
			continue
		}
		if err := writePlatformFile(s.path, s.data); err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		if s.mode != 0 {
			if err := os.Chmod(s.path, s.mode); err != nil && first == nil {
				first = err
			}
		}
	}
	return first
}

// writePlatformFile writes one private file atomically. The pre-create is
// what gives a new file 0600; edit.WriteAtomic keeps the mode afterwards.
func writePlatformFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}
	return edit.WriteAtomic(path, data)
}
