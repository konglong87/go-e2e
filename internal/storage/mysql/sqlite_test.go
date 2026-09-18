package mysql

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/konglong87/go-e2e/internal/agenttasks"
)

func TestSQLiteDesktopRepositorySupportsSessionControlPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "desktop.sqlite")
	ctx := context.Background()
	repo, err := OpenSQLiteGormRepository(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	tenantID, err := repo.UpsertTenant(ctx, TenantInput{TenantKey: "webui-local", Name: "Local Desktop"})
	if err != nil {
		t.Fatal(err)
	}
	userID, err := repo.EnsureUser(ctx, tenantID, "webui-local-user")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SetUserRole(ctx, tenantID, userID, DesktopLocalUserRole); err != nil {
		t.Fatal(err)
	}
	user, err := repo.GetUser(ctx, tenantID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if user.Role != DesktopLocalUserRole {
		t.Fatalf("desktop user role = %q, want %q", user.Role, DesktopLocalUserRole)
	}
	if _, err := repo.GetTenantQuotaConfig(ctx, tenantID); err != nil {
		t.Fatalf("get default quota config: %v", err)
	}
	if _, err := repo.ListTenantUsageDaily(ctx, tenantID, ListOptions{Limit: 10}); err != nil {
		t.Fatalf("list usage daily: %v", err)
	}
	if _, err := repo.ListTenantUsageLedger(ctx, tenantID, ListOptions{Limit: 10}); err != nil {
		t.Fatalf("list usage ledger: %v", err)
	}
	if _, err := repo.ListQuotaEvents(ctx, tenantID, ListOptions{Limit: 10}); err != nil {
		t.Fatalf("list quota events: %v", err)
	}
	sessionID, err := repo.UpsertSession(ctx, SessionInput{
		TenantID: tenantID, UserID: userID, SessionKey: "session-1",
		Title: "SQLite test", Status: "idle", MetadataJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpsertMessage(ctx, MessageInput{
		TenantID: tenantID, UserID: userID, SessionID: sessionID,
		TurnIndex: 1, Role: "user", Content: "hello",
	}); err != nil {
		t.Fatal(err)
	}
	taskID, err := repo.CreateAgentTask(ctx, agenttasks.TaskInput{
		TenantID: tenantID, UserID: userID, ParentSessionID: sessionID,
		AgentName: agenttasks.AgentNameWeb, Status: agenttasks.StatusReady,
		IdempotencyKey: "request-1", MetadataJSON: `{"model":"test"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	tasks, err := repo.ListWebAgentConversationTasks(ctx, tenantID, userID, sessionID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].ID != taskID || tasks[0].StartedAt.IsZero() {
		t.Fatalf("new task = %+v", tasks)
	}
	_, err = repo.CreateAgentTask(ctx, agenttasks.TaskInput{
		TenantID: tenantID, UserID: userID, AgentName: agenttasks.AgentNameWeb,
		Status: agenttasks.StatusCompleted, MetadataJSON: `{"web_agent_session_id":1}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.CreateAgentTask(ctx, agenttasks.TaskInput{
		TenantID: tenantID, UserID: userID, AgentName: agenttasks.AgentNameWeb,
		Status: agenttasks.StatusCompleted, MetadataJSON: `{"web_agent_session_id":"1"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	legacyTasks, err := repo.ListWebAgentConversationTasks(ctx, tenantID, userID, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(legacyTasks) != 3 {
		t.Fatalf("legacy tasks = %+v", legacyTasks)
	}
	if _, err := repo.AppendAgentTaskEvent(ctx, agenttasks.EventInput{
		TenantID: tenantID, UserID: userID, TaskID: taskID,
		EventType: agenttasks.EventMessage, PayloadJSON: `{"content":"hello"}`,
	}); err != nil {
		t.Fatal(err)
	}
	sessions, err := repo.ListSessionControlSessions(ctx, tenantID, userID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ID != sessionID {
		t.Fatalf("sessions = %+v", sessions)
	}

	// Reopening the same database must be safe and preserve the rows.
	if err := repo.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenSQLiteGormRepository(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	messages, err := reopened.ListMessages(ctx, tenantID, userID, sessionID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Content != "hello" {
		t.Fatalf("messages = %+v", messages)
	}
}

func TestSQLiteDesktopRepositorySupportsProfilesAndSkills(t *testing.T) {
	path := filepath.Join(t.TempDir(), "desktop.sqlite")
	ctx := context.Background()
	repo, err := OpenSQLiteGormRepository(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	tenantID, err := repo.UpsertTenant(ctx, TenantInput{TenantKey: "desktop", Name: "Desktop"})
	if err != nil {
		t.Fatal(err)
	}
	userID, err := repo.EnsureUser(ctx, tenantID, "desktop-user")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := repo.CreateAgentProfile(ctx, AgentProfileInput{
		TenantID: tenantID, OwnerKey: "desktop-user", ProfileKey: "default",
		Scope: "user_private", DisplayName: "Default", ProfileVersion: 1,
		ConfigJSON:      `{ "prompt": { "mode": "chat" } }`,
		CreatedByUserID: userID, UpdatedByUserID: userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if profile.ID == 0 {
		t.Fatal("profile id was not assigned")
	}
	if _, err := repo.UpsertAgentProfileAssignment(ctx, AgentProfileAssignmentInput{
		TenantID: tenantID, UserID: userID, Surface: "web_chat",
		ProfileID: profile.ID, AssignedByUserID: userID,
	}); err != nil {
		t.Fatal(err)
	}
	assignment, err := repo.GetAgentProfileAssignment(ctx, tenantID, userID, "web_chat")
	if err != nil {
		t.Fatal(err)
	}
	if assignment.ProfileID != profile.ID {
		t.Fatalf("assignment profile id = %d, want %d", assignment.ProfileID, profile.ID)
	}
	if _, err := repo.UpsertSkill(ctx, SkillInput{
		TenantID: tenantID, SkillKey: "welcome", Name: "Welcome",
		ContentMD: "Say hello.", Version: 1, Enabled: true,
		CreatedByUserID: userID,
	}); err != nil {
		t.Fatal(err)
	}
	skills, err := repo.ListEffectiveSkills(ctx, tenantID, userID, true, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].SkillKey != "welcome" {
		t.Fatalf("effective skills = %+v", skills)
	}
}

// The desktop stop button failed on SQLite because the stop idempotency
// recovery query used the MySQL-only JSON_UNQUOTE function. This round trip
// exercises cancel + recovery + audit lookup against a real SQLite database.
func TestSQLiteSessionControlStopRecoveryRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "desktop.sqlite")
	ctx := context.Background()
	repo, err := OpenSQLiteGormRepository(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	tenantID, err := repo.UpsertTenant(ctx, TenantInput{TenantKey: "webui-local", Name: "Local Desktop"})
	if err != nil {
		t.Fatal(err)
	}
	userID, err := repo.EnsureUser(ctx, tenantID, "webui-local-user")
	if err != nil {
		t.Fatal(err)
	}
	sessionID, err := repo.UpsertSession(ctx, SessionInput{
		TenantID: tenantID, UserID: userID, SessionKey: "stop-session",
		Title: "stop", Status: "running", MetadataJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	taskID, err := repo.CreateAgentTask(ctx, agenttasks.TaskInput{
		TenantID: tenantID, UserID: userID, ParentSessionID: sessionID,
		AgentName: agenttasks.AgentNameWeb, Status: agenttasks.StatusRunning, MetadataJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	keyHash := strings.Repeat("ab", 32)
	stopPayload := `{"schema":"golang-cc.session-control-stop.v1","operation":{"key_hash":"` + keyHash + `"}}`
	cancelled, err := repo.CancelAgentTaskForSessionControl(ctx, SessionControlStopInput{
		TenantID: tenantID, UserID: userID, TaskID: taskID,
		ResultJSON: `{"status":"cancelled"}`, EventPayloadJSON: stopPayload, TraceID: "trace-stop",
	})
	if err != nil || !cancelled.Cancelled {
		t.Fatalf("CancelAgentTaskForSessionControl() = %+v, err = %v", cancelled, err)
	}

	recovered, err := repo.RecoverSessionControlStop(ctx, tenantID, userID, sessionID, keyHash)
	if err != nil {
		t.Fatalf("RecoverSessionControlStop() err = %v", err)
	}
	if !recovered.Found || recovered.TaskID != taskID || recovered.EventID != cancelled.EventID || recovered.EventPayloadJSON != stopPayload {
		t.Fatalf("recovered = %+v, want task %d event %d", recovered, taskID, cancelled.EventID)
	}
	missing, err := repo.RecoverSessionControlStop(ctx, tenantID, userID, sessionID, strings.Repeat("cd", 32))
	if err != nil || missing.Found {
		t.Fatalf("missing recovery = %+v, err = %v", missing, err)
	}

	auditKeyHash := strings.Repeat("ef", 32)
	if _, err := repo.InsertSessionControlAudit(ctx, SessionControlAuditInput{AuditLogInput: AuditLogInput{
		TenantID: tenantID, ActorUserID: userID, Action: "session_control.stop",
		ResourceType: "session", ResourceID: "1",
		MetadataJSON: `{"session_control_operation":{"key_hash":"` + auditKeyHash + `"}}`,
		TraceID:      "trace-audit",
	}}); err != nil {
		t.Fatal(err)
	}
	audit, err := repo.GetSessionControlAuditByKeyHash(ctx, tenantID, userID, "session_control.stop", auditKeyHash)
	if err != nil || audit.Action != "session_control.stop" {
		t.Fatalf("audit = %+v, err = %v", audit, err)
	}
}

// Handoff replay/recovery predicates share the same JSON extraction helper.
// Links and events are staged directly so this test only covers the JSON
// predicate dialect, not the link upsert path.
func TestSQLiteHandoffRecoveryJSONPredicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "desktop.sqlite")
	ctx := context.Background()
	repo, err := OpenSQLiteGormRepository(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	tenantID, err := repo.UpsertTenant(ctx, TenantInput{TenantKey: "webui-local", Name: "Local Desktop"})
	if err != nil {
		t.Fatal(err)
	}
	userID, err := repo.EnsureUser(ctx, tenantID, "webui-local-user")
	if err != nil {
		t.Fatal(err)
	}
	sessionID, err := repo.UpsertSession(ctx, SessionInput{
		TenantID: tenantID, UserID: userID, SessionKey: "handoff-target",
		Title: "handoff", Status: "running", MetadataJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	taskID, err := repo.CreateAgentTask(ctx, agenttasks.TaskInput{
		TenantID: tenantID, UserID: userID, ParentSessionID: sessionID,
		AgentName: agenttasks.AgentNameWeb, Status: agenttasks.StatusRunning, MetadataJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	// The desktop SQLite schema has no tenant_session_links table (handoff is
	// a managed-MySQL feature); create the minimal shape so the recovery query
	// can join links while we validate the JSON predicate dialect on SQLite.
	if err := repo.db.Exec(`CREATE TABLE tenant_session_links (id integer PRIMARY KEY AUTOINCREMENT, tenant_id integer, user_id integer, target_session_id integer, source_kind text, source_session_key text, source_session_id integer, relation_type text, status text, metadata_json text, created_by_user_id integer, created_at datetime, updated_at datetime)`).Error; err != nil {
		t.Fatal(err)
	}

	operationIdentity := strings.Repeat("01", 32)
	sources := []string{"tenant:source-a", "tenant:source-b"}
	for index, sourceRef := range sources {
		payload := testSessionHandoffEventForOperation(t, taskID, sourceRef, fmt.Sprintf("task_event:%d", 21+index), operationIdentity)
		if _, err := repo.AppendAgentTaskEvent(ctx, agenttasks.EventInput{
			TenantID: tenantID, UserID: userID, TaskID: taskID,
			EventType: agenttasks.EventSessionHandoff, PayloadJSON: payload,
		}); err != nil {
			t.Fatal(err)
		}
		kind, key, ok := strings.Cut(sourceRef, ":")
		if !ok {
			t.Fatalf("source ref %q", sourceRef)
		}
		if err := repo.db.Create(&gormSessionLink{
			TenantID: tenantID, UserID: userID, TargetSessionID: sessionID,
			SourceKind: kind, SourceSessionKey: key, RelationType: agenttasks.SessionHandoffRelationType,
			Status: agenttasks.SessionHandoffLinkStatus, CreatedByUserID: userID,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}

	recovered, err := repo.RecoverHandoffBatch(ctx, agenttasks.HandoffRecoveryInput{
		TenantID: tenantID, UserID: userID, TargetSessionID: sessionID,
		TargetTaskID: taskID, OperationIdentity: operationIdentity, ExpectedSources: sources,
	})
	if err != nil {
		t.Fatalf("RecoverHandoffBatch() err = %v", err)
	}
	if !recovered.Found || len(recovered.Items) != len(sources) {
		t.Fatalf("recovered = %+v", recovered)
	}
	for index, item := range recovered.Items {
		if item.SourceRef != sources[index] || item.LinkID == 0 || item.EventID == 0 || item.PackageSHA256 == "" {
			t.Fatalf("recovered item %d = %+v", index, item)
		}
	}
}
