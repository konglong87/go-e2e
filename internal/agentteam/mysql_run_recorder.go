package agentteam

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	mysqlstore "github.com/konglong87/go-e2e/internal/storage/mysql"
)

type TeamRunStore interface {
	CreateAgentTeamRun(context.Context, mysqlstore.AgentTeamRunInput) (mysqlstore.AgentTeamRun, error)
	UpdateAgentTeamRun(context.Context, mysqlstore.AgentTeamRunUpdate) error
}

type MySQLRunRecorder struct {
	store TeamRunStore
}

func NewMySQLRunRecorder(store TeamRunStore) *MySQLRunRecorder {
	return &MySQLRunRecorder{store: store}
}

func (r *MySQLRunRecorder) Start(ctx context.Context, record RunRecord) error {
	if r == nil || r.store == nil || record.TenantID == 0 || record.TeamID == 0 || record.InboxEventID == 0 || record.SourceAccountID == 0 || strings.TrimSpace(record.RunID) == "" {
		return mysqlstore.ErrInvalidInput
	}
	var startedAt *time.Time
	if !record.StartedAt.IsZero() {
		value := record.StartedAt
		startedAt = &value
	}
	_, err := r.store.CreateAgentTeamRun(ctx, mysqlstore.AgentTeamRunInput{ID: record.RunID, TenantID: record.TenantID, TeamID: record.TeamID, InboxEventID: record.InboxEventID, SourceAccountID: record.SourceAccountID, ConversationID: record.ConversationID, CoordinatorMemberKey: record.CoordinatorMember, Status: string(record.Status), TeamEffectiveHash: record.TeamEffectiveHash, MemberCount: record.MemberCount, MaxRounds: record.MaxRounds, MaxParallelMembers: record.MaxParallelMembers, MaxTotalTokens: record.MaxTotalTokens, StartedAt: startedAt})
	return err
}

func (r *MySQLRunRecorder) Finish(ctx context.Context, record RunRecord) error {
	if r == nil || r.store == nil || record.TenantID == 0 || strings.TrimSpace(record.RunID) == "" || record.Status == "" {
		return mysqlstore.ErrInvalidInput
	}
	var finishedAt *time.Time
	if !record.FinishedAt.IsZero() {
		value := record.FinishedAt
		finishedAt = &value
	}
	resultJSON, err := json.Marshal(map[string]string{"content": record.Result})
	if err != nil {
		return err
	}
	return r.store.UpdateAgentTeamRun(ctx, mysqlstore.AgentTeamRunUpdate{TenantID: record.TenantID, RunID: record.RunID, Status: string(record.Status), UsedTokens: record.UsedTokens, UsedTurns: record.UsedTurns, FinishedAt: finishedAt, ResultJSON: string(resultJSON), ErrorMessage: record.Error})
}
