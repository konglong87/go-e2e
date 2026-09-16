package agentteam

import (
	"context"
	"encoding/json"
	"testing"

	mysqlstore "github.com/konglong87/go-e2e/internal/storage/mysql"
)

type fakeTeamRunStore struct {
	updated mysqlstore.AgentTeamRunUpdate
}

func (f *fakeTeamRunStore) CreateAgentTeamRun(_ context.Context, input mysqlstore.AgentTeamRunInput) (mysqlstore.AgentTeamRun, error) {
	return mysqlstore.AgentTeamRun{ID: input.ID}, nil
}

func (f *fakeTeamRunStore) UpdateAgentTeamRun(_ context.Context, input mysqlstore.AgentTeamRunUpdate) error {
	f.updated = input
	return nil
}

func TestMySQLRunRecorderEncodesMarkdownResultAsJSON(t *testing.T) {
	store := &fakeTeamRunStore{}
	recorder := NewMySQLRunRecorder(store)
	const result = "# 标题\n\n正文 **加粗**"
	if err := recorder.Finish(context.Background(), RunRecord{TenantID: 1, RunID: "team-run-1", Status: StatusCompleted, Result: result}); err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(store.updated.ResultJSON)) {
		t.Fatalf("result_json=%q is not valid JSON", store.updated.ResultJSON)
	}
	var decoded map[string]string
	if err := json.Unmarshal([]byte(store.updated.ResultJSON), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["content"] != result {
		t.Fatalf("decoded result=%q, want %q", decoded["content"], result)
	}
}
