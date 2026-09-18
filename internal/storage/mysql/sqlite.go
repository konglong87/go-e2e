package mysql

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// OpenSQLiteGormRepository opens the desktop persistence database while
// retaining the existing tenant repository contract. The desktop schema is
// intentionally limited to the session-control runtime surface; optional
// MySQL-only channel and image tables remain server concerns.
func OpenSQLiteGormRepository(ctx context.Context, path string, logger *slog.Logger) (*GormRepository, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite database path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := gorm.Open(gormsqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	repo := &GormRepository{db: db, logger: logger}
	if err := migrateSQLiteDesktopSchema(db); err != nil {
		_ = repo.Close()
		return nil, err
	}
	return repo, nil
}

func migrateSQLiteDesktopSchema(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&gormTenant{}, &gormTenantUser{}, &gormMemory{}, &gormSkill{}, &gormSkillOverride{}, &gormDocument{},
		&gormSession{}, &gormMessage{},
		&gormAgentTask{}, &gormAgentTaskEvent{}, &gormAuditLog{}, &gormTelemetryEvent{},
		&gormPendingInput{}, &gormPendingInputSetting{},
		&gormAgentProfile{}, &gormAgentProfileAssignment{}, &gormAgentProfileChannelBinding{},
		&gormAgentTeam{}, &gormAgentTeamMember{}, &gormAgentTeamBinding{},
		&gormAgentTeamRun{}, &gormAgentTeamMailbox{},
		&gormPromptTemplate{},
	); err != nil {
		return err
	}
	// These timestamps are part of the existing MySQL contract but are not
	// represented by the write models. Keep them explicit for SQLite reads.
	for _, statement := range []string{
		`ALTER TABLE tenants ADD COLUMN created_at DATETIME`,
		`ALTER TABLE tenants ADD COLUMN updated_at DATETIME`,
		`ALTER TABLE tenant_users ADD COLUMN created_at DATETIME`,
		`ALTER TABLE tenant_users ADD COLUMN updated_at DATETIME`,
		`ALTER TABLE tenant_sessions ADD COLUMN created_at DATETIME`,
		`ALTER TABLE tenant_sessions ADD COLUMN started_at DATETIME`,
		`ALTER TABLE tenant_sessions ADD COLUMN archived_at DATETIME`,
		`ALTER TABLE tenant_session_messages ADD COLUMN created_at DATETIME`,
		`ALTER TABLE tenant_agent_tasks ADD COLUMN started_at DATETIME`,
		`ALTER TABLE tenant_agent_tasks ADD COLUMN finished_at DATETIME`,
		`ALTER TABLE tenant_agent_task_events ADD COLUMN created_at DATETIME`,
		`ALTER TABLE tenant_audit_logs ADD COLUMN created_at DATETIME`,
	} {
		if err := addSQLiteColumnIfMissing(db, statement); err != nil {
			return err
		}
	}
	for _, statement := range []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_tenants_tenant_key ON tenants (tenant_key)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_tenant_users_user_key ON tenant_users (tenant_id, user_key)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_tenant_skills_key_version ON tenant_skills (tenant_id, skill_key, version)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_user_skill_override ON tenant_user_skill_overrides (tenant_id, user_id, skill_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_agent_profiles_version_owner ON agent_profiles (tenant_id, profile_key, profile_version, owner_key)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_agent_profile_assignments_surface ON agent_profile_assignments (tenant_id, user_id, surface)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_agent_profile_channel_bindings_profile ON agent_profile_channel_bindings (tenant_id, profile_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_tenant_sessions_key ON tenant_sessions (tenant_id, user_id, session_key)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_session_messages_turn ON tenant_session_messages (session_id, turn_index)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_tasks_parent_session ON tenant_agent_tasks (tenant_id, user_id, parent_session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_task_events_task ON tenant_agent_task_events (task_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_tenant_audit_recent ON tenant_audit_logs (tenant_id, created_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_prompt_templates_owner_title ON tenant_prompt_templates (tenant_id, user_id, title)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS tenant_quota_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER NOT NULL,
			quota_enabled INTEGER NOT NULL DEFAULT 0,
			qps_limit INTEGER NULL,
			daily_token_limit INTEGER NULL,
			daily_message_limit INTEGER NULL,
			max_concurrent_requests INTEGER NULL,
			timezone TEXT NOT NULL DEFAULT 'UTC',
			reserve_output_tokens INTEGER NOT NULL DEFAULT 4096,
			status TEXT NOT NULL DEFAULT 'active',
			updated_by_user_id INTEGER NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (tenant_id)
		)`,
		`CREATE TABLE IF NOT EXISTS tenant_usage_ledger (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id TEXT NOT NULL UNIQUE,
			tenant_id INTEGER NOT NULL,
			user_id INTEGER NULL,
			session_id INTEGER NULL,
			trace_id TEXT NULL,
			source TEXT NOT NULL,
			route TEXT NULL,
			model TEXT NULL,
			provider TEXT NULL,
			turn_index INTEGER NOT NULL DEFAULT 0,
			usage_source TEXT NULL,
			status TEXT NOT NULL,
			estimated INTEGER NOT NULL DEFAULT 0,
			reserved_input_tokens INTEGER NOT NULL DEFAULT 0,
			reserved_output_tokens INTEGER NOT NULL DEFAULT 0,
			input_tokens INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			cache_read_input_tokens INTEGER NOT NULL DEFAULT 0,
			cache_creation_input_tokens INTEGER NOT NULL DEFAULT 0,
			cache_creation_ephemeral_1h_input_tokens INTEGER NOT NULL DEFAULT 0,
			cache_creation_ephemeral_5m_input_tokens INTEGER NOT NULL DEFAULT 0,
			total_tokens INTEGER NOT NULL DEFAULT 0,
			error_code TEXT NULL,
			error_message TEXT NULL,
			started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			finished_at DATETIME NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS tenant_usage_daily (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER NOT NULL,
			usage_date DATE NOT NULL,
			source TEXT NOT NULL DEFAULT '*',
			model TEXT NOT NULL DEFAULT '*',
			request_count INTEGER NOT NULL DEFAULT 0,
			message_count INTEGER NOT NULL DEFAULT 0,
			input_tokens INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			cache_read_input_tokens INTEGER NOT NULL DEFAULT 0,
			cache_creation_input_tokens INTEGER NOT NULL DEFAULT 0,
			total_tokens INTEGER NOT NULL DEFAULT 0,
			rejected_count INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (tenant_id, usage_date, source, model)
		)`,
		`CREATE TABLE IF NOT EXISTS tenant_quota_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER NOT NULL,
			user_id INTEGER NULL,
			request_id TEXT NULL,
			event_type TEXT NOT NULL,
			limit_type TEXT NULL,
			limit_value INTEGER NULL,
			current_value INTEGER NULL,
			source TEXT NULL,
			route TEXT NULL,
			model TEXT NULL,
			trace_id TEXT NULL,
			metadata_json TEXT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_ledger_tenant_started ON tenant_usage_ledger (tenant_id, started_at, id)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_daily_date ON tenant_usage_daily (usage_date, tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_quota_events_tenant_created ON tenant_quota_events (tenant_id, created_at, id)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	for _, statement := range []string{
		`UPDATE tenants SET created_at = COALESCE(created_at, CURRENT_TIMESTAMP), updated_at = COALESCE(updated_at, CURRENT_TIMESTAMP)`,
		`UPDATE tenant_users SET created_at = COALESCE(created_at, CURRENT_TIMESTAMP), updated_at = COALESCE(updated_at, CURRENT_TIMESTAMP)`,
		`UPDATE tenant_sessions SET created_at = COALESCE(created_at, CURRENT_TIMESTAMP), started_at = COALESCE(started_at, CURRENT_TIMESTAMP)`,
		`UPDATE tenant_session_messages SET created_at = COALESCE(created_at, CURRENT_TIMESTAMP)`,
		`UPDATE tenant_agent_tasks SET started_at = COALESCE(started_at, CURRENT_TIMESTAMP)`,
		`UPDATE tenant_agent_task_events SET created_at = COALESCE(created_at, CURRENT_TIMESTAMP)`,
		`UPDATE tenant_audit_logs SET created_at = COALESCE(created_at, CURRENT_TIMESTAMP)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func addSQLiteColumnIfMissing(db *gorm.DB, statement string) error {
	var table, column string
	parts := strings.Fields(statement)
	if len(parts) < 6 {
		return fmt.Errorf("invalid sqlite migration statement %q", statement)
	}
	table, column = parts[2], parts[5]
	var count int64
	if err := db.Raw(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if err := db.Exec(statement).Error; err != nil {
		return fmt.Errorf("sqlite migration failed: %s: %w", statement, err)
	}
	return nil
}
