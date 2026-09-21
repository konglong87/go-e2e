package mysql

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	channelcontract "github.com/konglong87/go-e2e/internal/channel"
)

func TestSQLiteDesktopRepositorySupportsChannelRuntimePersistence(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenSQLiteGormRepository(ctx, filepath.Join(t.TempDir(), "channel.sqlite"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	tenantID, err := repo.UpsertTenant(ctx, TenantInput{TenantKey: "desktop-channel", Name: "Desktop Channel"})
	if err != nil {
		t.Fatal(err)
	}
	userID, err := repo.EnsureUser(ctx, tenantID, "desktop-user")
	if err != nil {
		t.Fatal(err)
	}
	accountID, err := repo.EnsureChannelAccount(ctx, tenantID, ChannelProviderFeishu, "code", "cli_test", "code")
	if err != nil {
		t.Fatal(err)
	}
	if accountID == 0 {
		t.Fatal("channel account id is zero")
	}

	scope, err := channelcontract.NewScope(channelcontract.ProviderFeishu, "code", "chat-1", "")
	if err != nil {
		t.Fatal(err)
	}
	event, created, err := repo.ClaimInboxEvent(ctx, ChannelInboxEventInput{
		TenantID: tenantID, AccountID: accountID, ProviderEventID: "event-1",
		ProviderMessageID: "message-1", ScopeHash: scope.Hash[:],
		PayloadSHA256: []byte("hash"), PayloadCiphertext: []byte("{}"),
		Authorized: true, Status: ChannelInboxStatusReceived, ReceivedAt: ptrTimeForSQLiteTest(time.Now().UTC()),
	})
	if err != nil || !created {
		t.Fatalf("claim inbox event = %+v, created=%v, err=%v", event, created, err)
	}
	if err := repo.MarkInboxQueued(ctx, tenantID, event.ID, 1, scope.Hash[:]); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkInboxProcessed(ctx, tenantID, event.ID); err != nil {
		t.Fatal(err)
	}

	conversation, err := repo.GetOrCreateChannelConversation(ctx, ChannelConversationInput{
		TenantID: tenantID, AccountID: accountID, ExternalChatID: "chat-1",
		ScopeKey: scope.Key, ScopeHash: scope.Hash[:], ExternalUserID: "user-1",
		ChatType: ChannelChatTypeP2P, SessionID: userID,
		RuntimeFingerprint:        channelruntimeFingerprintForSQLiteTest,
		RuntimeFingerprintVersion: 1, Status: ChannelConversationStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	run, err := repo.CreateChannelRun(ctx, ChannelRunInput{
		ID: "run-1", TenantID: tenantID, AccountID: accountID,
		ConversationID: conversation.ID, ScopeHash: scope.Hash[:],
		SessionID: conversation.SessionID, RuntimeFingerprint: channelruntimeFingerprintForSQLiteTest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AttachRunInputs(ctx, tenantID, run.ID, []ChannelRunInputEvent{{TenantID: tenantID, InboxEventID: event.ID, SequenceNo: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := repo.RequeueStrandedChannelRuns(ctx, tenantID, accountID); err != nil {
		t.Fatal(err)
	}
	if err := repo.FinishChannelRun(ctx, tenantID, accountID, run.ID, ChannelRunStatusCompleted, "", ""); err != nil {
		t.Fatal(err)
	}
}

const channelruntimeFingerprintForSQLiteTest = "sqlite-test-runtime"

func ptrTimeForSQLiteTest(value time.Time) *time.Time { return &value }
