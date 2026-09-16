package provisioning

import (
	"context"
	"testing"
)

type memoryRepo struct{ item Record }

func (m *memoryRepo) UpsertAgentProvisioning(_ context.Context, input RecordInput) (Record, error) {
	if m.item.ID == 0 {
		m.item.ID = 1
	}
	m.item = Record{ID: m.item.ID, TenantID: input.TenantID, ProfileKey: input.ProfileKey, AccountKey: input.AccountKey, CredentialRef: input.CredentialRef, Supervisor: input.Supervisor, Status: Status(input.Status), WorkerSpec: input.WorkerSpec, WorkerStatus: input.WorkerStatus, Checks: input.Checks, LastError: input.LastError}
	return m.item, nil
}
func (m *memoryRepo) GetAgentProvisioning(_ context.Context, tenantID, _ uint64, _ string) (Record, error) {
	if m.item.TenantID != tenantID {
		return Record{}, NewError(ErrNotFound, "not found")
	}
	return m.item, nil
}
func (m *memoryRepo) ListAgentProvisionings(context.Context, uint64, int) ([]Record, error) {
	return []Record{m.item}, nil
}

type fakeCreds struct{}

func (fakeCreds) Put(context.Context, CredentialRef) (CredentialRef, error) {
	return CredentialRef{}, nil
}
func (fakeCreds) Get(context.Context, string) (CredentialRef, error) {
	return CredentialRef{ID: "cred", AppID: "app", SecretValue: "secret"}, nil
}
func (fakeCreds) Delete(context.Context, string) error { return nil }

type fakeFeishu struct{}

func (fakeFeishu) Preflight(context.Context, CredentialRef, WorkerSpec) ([]HealthCheck, error) {
	return []HealthCheck{{Name: "token", Status: "passed"}}, nil
}
func TestServicePreflightPersistsChecks(t *testing.T) {
	repo := &memoryRepo{}
	service := &Service{Repo: repo, Credentials: fakeCreds{}, Feishu: fakeFeishu{}}
	item, err := service.Create(context.Background(), 9, 2, CreateRequest{ProfileKey: "writer", AccountKey: "writer-feishu", CredentialRef: CredentialRef{ID: "cred"}})
	if err != nil {
		t.Fatal(err)
	}
	item, err = service.Preflight(context.Background(), 9, item.ID, 2)
	if err != nil || item.Status != StatusPreflight || len(item.Checks) != 1 {
		t.Fatalf("item=%#v err=%v", item, err)
	}
}

type fakeInventory struct{}

func (fakeInventory) List(context.Context) ([]WorkerStatus, error) {
	return []WorkerStatus{{TenantID: 9, AccountID: 2, State: WorkerStateRunning, PID: 22}}, nil
}
func TestServiceOverviewIncludesUnmanagedWorkers(t *testing.T) {
	repo := &memoryRepo{}
	service := &Service{Repo: repo, Inventory: fakeInventory{}}
	result, err := service.Overview(context.Background(), 9, 100)
	if err != nil || len(result.Workers) != 1 || result.Workers[0].AccountID != 2 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}
