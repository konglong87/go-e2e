package credentials

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/konglong87/go-e2e/internal/provisioning"
)

var ErrInvalidCredential = errors.New("credentials: invalid credential")

type FileStore struct{ Dir string }

func (s FileStore) Put(ctx context.Context, ref provisioning.CredentialRef) (provisioning.CredentialRef, error) {
	if err := ctx.Err(); err != nil {
		return provisioning.CredentialRef{}, err
	}
	if strings.TrimSpace(ref.Provider) == "" || strings.TrimSpace(ref.AppID) == "" || strings.TrimSpace(ref.SecretValue) == "" {
		return provisioning.CredentialRef{}, ErrInvalidCredential
	}
	if s.Dir == "" {
		return provisioning.CredentialRef{}, ErrInvalidCredential
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return provisioning.CredentialRef{}, err
	}
	path := s.path(ref.ID)
	if ref.ID == "" {
		return provisioning.CredentialRef{}, ErrInvalidCredential
	}
	contents := []byte(ref.AppID + "\n" + ref.SecretValue + "\n")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, contents, 0o600); err != nil {
		return provisioning.CredentialRef{}, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return provisioning.CredentialRef{}, err
	}
	ref.SecretPresent = true
	ref.SecretValue = ""
	return ref, nil
}

func (s FileStore) Get(ctx context.Context, id string) (provisioning.CredentialRef, error) {
	if err := ctx.Err(); err != nil {
		return provisioning.CredentialRef{}, err
	}
	if strings.TrimSpace(id) == "" {
		return provisioning.CredentialRef{}, ErrInvalidCredential
	}
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return provisioning.CredentialRef{}, err
	}
	parts := strings.SplitN(string(data), "\n", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return provisioning.CredentialRef{}, ErrInvalidCredential
	}
	return provisioning.CredentialRef{ID: id, Provider: "feishu", AppID: parts[0], SecretValue: parts[1], SecretPresent: true}, nil
}

func (s FileStore) Delete(_ context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidCredential
	}
	return os.Remove(s.path(id))
}

func (s FileStore) Path(id string) string { return s.path(id) }
func (s FileStore) path(id string) string { return s.Dir + "/" + id + ".credential" }

var _ provisioning.CredentialStore = FileStore{}
