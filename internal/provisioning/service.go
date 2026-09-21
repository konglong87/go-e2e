package provisioning

import (
	"context"
	"fmt"
	"strings"
)

type RecordInput struct {
	ID                                                        uint64
	TenantID                                                  uint64
	ProfileKey, AccountKey, CredentialRef, Supervisor, Status string
	WorkerSpec                                                WorkerSpec
	WorkerStatus                                              WorkerStatus
	Checks                                                    []HealthCheck
	LastError                                                 *ProvisionError
	UserID                                                    uint64
}
type Repository interface {
	UpsertAgentProvisioning(context.Context, RecordInput) (Record, error)
	GetAgentProvisioning(context.Context, uint64, uint64, string) (Record, error)
	ListAgentProvisionings(context.Context, uint64, int) ([]Record, error)
}
type Service struct {
	Repo        Repository
	Credentials CredentialStore
	Channels    ChannelAccountStore
	Feishu      FeishuProvisioner
	Supervisor  WorkerSupervisor
	Inventory   WorkerInventory
	Providers   ProviderCatalog
}
type CreateRequest struct {
	ProfileKey    string        `json:"profile_key"`
	AccountKey    string        `json:"account_key"`
	CredentialRef CredentialRef `json:"credential"`
	SecretValue   string        `json:"secret_value,omitempty"`
	Worker        WorkerSpec    `json:"worker"`
}

func (s *Service) Create(ctx context.Context, tenantID, userID uint64, req CreateRequest) (Record, error) {
	if tenantID == 0 || userID == 0 || strings.TrimSpace(req.ProfileKey) == "" || strings.TrimSpace(req.AccountKey) == "" {
		return Record{}, NewError(ErrInvalidInput, "profile_key, account_key and tenant context are required")
	}
	if req.Worker.Supervisor == "" {
		req.Worker.Supervisor = "screen"
	}
	if strings.TrimSpace(req.Worker.WorkerName) == "" {
		req.Worker.WorkerName = strings.TrimSpace(req.ProfileKey)
	}
	if req.SecretValue == "" {
		req.SecretValue = req.CredentialRef.SecretValue
	}
	if req.SecretValue != "" {
		if s.Credentials == nil {
			return Record{}, NewError(ErrInvalidInput, "credential store is not configured")
		}
		req.CredentialRef.SecretValue = req.SecretValue
		stored, err := s.Credentials.Put(ctx, req.CredentialRef)
		if err != nil {
			return Record{}, NewError(ErrInvalidInput, err.Error())
		}
		req.CredentialRef = stored
		if pathProvider, ok := s.Credentials.(CredentialPathProvider); ok {
			if req.Worker.Environment == nil {
				req.Worker.Environment = map[string]string{}
			}
			req.Worker.Environment["GOLANG_CC_FEISHU_CREDENTIAL_FILE"] = pathProvider.Path(stored.ID)
		}
	}
	if req.Worker.Environment == nil {
		req.Worker.Environment = map[string]string{}
	}
	req.Worker.Environment["GOLANG_CC_CHANNEL_TENANT_ID"] = fmt.Sprint(tenantID)
	req.Worker.Environment["GOLANG_CC_CHANNEL_USER_ID"] = fmt.Sprint(userID)
	if s.Channels != nil {
		accountID, err := s.Channels.EnsureChannelAccount(ctx, tenantID, "feishu", req.AccountKey, req.CredentialRef.AppID, req.CredentialRef.ID)
		if err != nil {
			return Record{}, NewError(ErrInvalidInput, err.Error())
		}
		req.Worker.Environment["GOLANG_CC_CHANNEL_ACCOUNT_ID"] = fmt.Sprint(accountID)
	}
	return s.Repo.UpsertAgentProvisioning(ctx, RecordInput{TenantID: tenantID, ProfileKey: req.ProfileKey, AccountKey: req.AccountKey, CredentialRef: req.CredentialRef.ID, Supervisor: req.Worker.Supervisor, Status: string(StatusDraft), WorkerSpec: req.Worker, UserID: userID})
}
func (s *Service) List(ctx context.Context, tenantID uint64, limit int) ([]Record, error) {
	return s.Repo.ListAgentProvisionings(ctx, tenantID, limit)
}
func (s *Service) Overview(ctx context.Context, tenantID uint64, limit int) (Overview, error) {
	records, err := s.List(ctx, tenantID, limit)
	if err != nil {
		return Overview{}, err
	}
	workers := []WorkerStatus{}
	if s.Inventory != nil {
		all, inventoryErr := s.Inventory.List(ctx)
		if inventoryErr != nil {
			return Overview{}, inventoryErr
		}
		for _, worker := range all {
			if worker.TenantID == tenantID {
				workers = append(workers, worker)
			}
		}
	}
	return Overview{Records: records, Workers: workers}, nil
}
func (s *Service) Get(ctx context.Context, tenantID, id uint64, key string) (Record, error) {
	return s.Repo.GetAgentProvisioning(ctx, tenantID, id, key)
}
func (s *Service) Preflight(ctx context.Context, tenantID, id, userID uint64) (Record, error) {
	item, err := s.Get(ctx, tenantID, id, "")
	if err != nil {
		return item, err
	}
	if s.Feishu == nil || s.Credentials == nil {
		return item, NewError(ErrPreflight, "Feishu preflight is not configured")
	}
	credential, err := s.Credentials.Get(ctx, item.CredentialRef)
	if err != nil {
		return item, err
	}
	checks, err := s.Feishu.Preflight(ctx, credential, item.WorkerSpec)
	item.Checks = checks
	if err != nil {
		item.LastError = NewError(ErrPreflight, err.Error())
		item.Status = StatusFailed
	} else {
		item.Status = StatusPreflight
	}
	return s.save(ctx, item, userID)
}
func (s *Service) WorkerAction(ctx context.Context, tenantID, id, userID uint64, action string) (Record, error) {
	item, err := s.Get(ctx, tenantID, id, "")
	if err != nil {
		return item, err
	}
	if s.Supervisor == nil {
		return item, NewError(ErrSupervisor, "worker supervisor is not configured")
	}
	var status WorkerStatus
	switch action {
	case "start":
		status, err = s.Supervisor.Start(ctx, item.WorkerSpec)
	case "restart":
		status, err = s.Supervisor.Restart(ctx, item.WorkerSpec)
	case "stop":
		status, err = s.Supervisor.Stop(ctx, item.WorkerSpec)
	case "status":
		status, err = s.Supervisor.Status(ctx, item.WorkerSpec)
	default:
		return item, NewError(ErrInvalidInput, "unknown worker action")
	}
	item.WorkerStatus = status
	if err != nil {
		item.Status = StatusFailed
		item.LastError = NewError(ErrSupervisor, err.Error())
	} else if status.State == WorkerStateRunning {
		item.Status = StatusRunning
		item.LastError = nil
	} else {
		item.Status = Status(status.State)
	}
	return s.save(ctx, item, userID)
}

func (s *Service) Logs(ctx context.Context, tenantID, id uint64, tail int) (string, error) {
	item, err := s.Get(ctx, tenantID, id, "")
	if err != nil {
		return "", err
	}
	if s.Supervisor == nil {
		return "", NewError(ErrSupervisor, "worker supervisor is not configured")
	}
	return s.Supervisor.Logs(ctx, item.WorkerSpec, tail)
}
func (s *Service) save(ctx context.Context, item Record, userID uint64) (Record, error) {
	return s.Repo.UpsertAgentProvisioning(ctx, RecordInput{ID: item.ID, TenantID: item.TenantID, ProfileKey: item.ProfileKey, AccountKey: item.AccountKey, CredentialRef: item.CredentialRef, Supervisor: item.Supervisor, Status: string(item.Status), WorkerSpec: item.WorkerSpec, WorkerStatus: item.WorkerStatus, Checks: item.Checks, LastError: item.LastError, UserID: userID})
}
