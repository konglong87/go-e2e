package agentprofile

import (
	"context"
	"testing"

	mysqlstore "github.com/konglong87/go-e2e/internal/storage/mysql"
)

type fakeMySQLProfileStore struct {
	profiles   []mysqlstore.AgentProfile
	assignment mysqlstore.AgentProfileAssignment
}

func (f fakeMySQLProfileStore) ListAgentProfiles(context.Context, uint64, uint64, string, int) ([]mysqlstore.AgentProfile, error) {
	return append([]mysqlstore.AgentProfile(nil), f.profiles...), nil
}

func (f fakeMySQLProfileStore) GetAgentProfile(context.Context, uint64, uint64) (mysqlstore.AgentProfile, error) {
	for _, profile := range f.profiles {
		if profile.ID == f.assignment.ProfileID {
			return profile, nil
		}
	}
	return mysqlstore.AgentProfile{}, mysqlstore.ErrNotFound
}

func (f fakeMySQLProfileStore) GetAgentProfileAssignment(context.Context, uint64, uint64, string) (mysqlstore.AgentProfileAssignment, error) {
	return f.assignment, nil
}

func TestMySQLSourceMapsPublishedProfileAndAssignment(t *testing.T) {
	copywriter := builtinCopywriter()
	configJSON, err := copywriter.Config.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	source := NewMySQLSource(fakeMySQLProfileStore{
		profiles: []mysqlstore.AgentProfile{{
			ID: 42, TenantID: 7, OwnerKey: "shared", ProfileKey: copywriter.Key, Scope: string(copywriter.Scope),
			DisplayName: copywriter.DisplayName, ProfileVersion: copywriter.Version, Status: string(StatusPublished), ConfigJSON: string(configJSON),
		}},
		assignment: mysqlstore.AgentProfileAssignment{TenantID: 7, UserID: 11, Surface: SurfaceWebChat, ProfileID: 42},
	})

	profile, err := source.FindPublishedProfile(context.Background(), 7, 11, ProfileCopywriter, 1)
	if err != nil || profile.Key != ProfileCopywriter || profile.Version != 1 || profile.Config.Prompt.Mode != PromptModeChat {
		t.Fatalf("profile=%+v err=%v", profile, err)
	}
	assignment, err := source.FindAssignment(context.Background(), 7, 11, SurfaceWebChat)
	if err != nil || assignment.ProfileKey != ProfileCopywriter || assignment.ProfileVersion != 1 {
		t.Fatalf("assignment=%+v err=%v", assignment, err)
	}
}
