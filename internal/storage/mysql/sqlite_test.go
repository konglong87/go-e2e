package mysql

import (
	"context"
	"path/filepath"
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
