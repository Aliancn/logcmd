package migration

import (
	"database/sql"
	"fmt"
)

// Migration 数据库迁移管理
type Migration struct {
	db *sql.DB
}

// NewMigration 创建迁移管理器
func NewMigration(db *sql.DB) *Migration {
	return &Migration{db: db}
}

// Migrate 执行数据库迁移
func (m *Migration) Migrate() error {
	return m.createNewTables()
}

// createNewTables 创建新版本的所有表
func (m *Migration) createNewTables() error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 创建 projects 表
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			-- 基本信息
			path TEXT NOT NULL UNIQUE,
			name TEXT DEFAULT '',
			description TEXT DEFAULT '',

			-- 分类和标签
			category TEXT DEFAULT '',
			tags TEXT DEFAULT '',

			-- 统计信息
			total_commands INTEGER DEFAULT 0,
			success_commands INTEGER DEFAULT 0,
			failed_commands INTEGER DEFAULT 0,
			total_duration_ms INTEGER DEFAULT 0,

			-- 最后执行信息
			last_command TEXT DEFAULT '',
			last_command_status TEXT DEFAULT '',
			last_command_time TIMESTAMP,

			-- 时间戳
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			last_checked TIMESTAMP NOT NULL,

			-- 配置信息
			template_config TEXT DEFAULT '',
			custom_config TEXT DEFAULT ''
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 projects 表失败: %w", err)
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_projects_path ON projects(path)",
		"CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name)",
		"CREATE INDEX IF NOT EXISTS idx_projects_category ON projects(category)",
		"CREATE INDEX IF NOT EXISTS idx_projects_updated_at ON projects(updated_at)",
		"CREATE INDEX IF NOT EXISTS idx_projects_last_command_time ON projects(last_command_time)",
	}

	for _, idx := range indexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("创建索引失败: %w", err)
		}
	}

	// 创建 command_history 表
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,

			-- 命令信息
			command TEXT NOT NULL,
			command_name TEXT NOT NULL,
			command_args TEXT,

			-- 执行信息
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP NOT NULL,
			duration_ms INTEGER NOT NULL,
			exit_code INTEGER NOT NULL,
			status TEXT NOT NULL,

			-- 日志文件信息
			log_file_path TEXT NOT NULL,
			log_date TEXT NOT NULL,

			-- 输出预览
			stdout_preview TEXT,
			stderr_preview TEXT,
			has_error BOOLEAN DEFAULT 0,

			-- 元数据
			working_directory TEXT,
			environment_info TEXT,

			-- 时间戳
			created_at TIMESTAMP NOT NULL,

			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 command_history 表失败: %w", err)
	}

	// 创建 command_history 索引
	cmdIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_command_history_project_id ON command_history(project_id)",
		"CREATE INDEX IF NOT EXISTS idx_command_history_command_name ON command_history(command_name)",
		"CREATE INDEX IF NOT EXISTS idx_command_history_start_time ON command_history(start_time)",
		"CREATE INDEX IF NOT EXISTS idx_command_history_status ON command_history(status)",
		"CREATE INDEX IF NOT EXISTS idx_command_history_log_date ON command_history(log_date)",
		"CREATE INDEX IF NOT EXISTS idx_command_history_exit_code ON command_history(exit_code)",
		"CREATE INDEX IF NOT EXISTS idx_command_history_project_time ON command_history(project_id, start_time DESC)",
		"CREATE INDEX IF NOT EXISTS idx_command_history_project_status ON command_history(project_id, status)",
	}

	for _, idx := range cmdIndexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("创建命令历史索引失败: %w", err)
		}
	}

	// 创建 project_stats_cache 表
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS project_stats_cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			stat_date TEXT NOT NULL,

			-- 每日统计
			total_commands INTEGER DEFAULT 0,
			success_commands INTEGER DEFAULT 0,
			failed_commands INTEGER DEFAULT 0,
			total_duration_ms INTEGER DEFAULT 0,
			avg_duration_ms INTEGER DEFAULT 0,
			max_duration_ms INTEGER DEFAULT 0,
			min_duration_ms INTEGER DEFAULT 0,

			-- 分布统计
			command_distribution TEXT,
			exit_code_distribution TEXT,

			-- 时间戳
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
			UNIQUE(project_id, stat_date)
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 project_stats_cache 表失败: %w", err)
	}

	// 创建统计缓存索引
	statsIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_project_stats_project_id ON project_stats_cache(project_id)",
		"CREATE INDEX IF NOT EXISTS idx_project_stats_stat_date ON project_stats_cache(stat_date)",
		"CREATE INDEX IF NOT EXISTS idx_project_stats_project_date ON project_stats_cache(project_id, stat_date DESC)",
	}

	for _, idx := range statsIndexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("创建统计缓存索引失败: %w", err)
		}
	}

	// 创建系统配置表
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS system_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			description TEXT,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 system_config 表失败: %w", err)
	}

	// 创建后台任务表
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT NOT NULL,
			command_args TEXT,
			working_dir TEXT NOT NULL,
			log_dir TEXT NOT NULL,
			status TEXT NOT NULL,
			pid INTEGER,
			log_file_path TEXT,
			exit_code INTEGER,
			error_message TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			started_at TIMESTAMP,
			completed_at TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 tasks 表失败: %w", err)
	}

	taskIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at)",
	}

	for _, idx := range taskIndexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("创建 tasks 索引失败: %w", err)
		}
	}

	// 插入默认配置
	_, err = tx.Exec(`
		INSERT OR IGNORE INTO system_config (key, value, description, updated_at) VALUES
		('version', '2', '数据库版本', CURRENT_TIMESTAMP),
		('auto_cleanup_days', '365', '自动清理日志的天数', CURRENT_TIMESTAMP),
		('enable_stdout_preview', 'true', '是否启用输出预览功能', CURRENT_TIMESTAMP),
		('max_preview_length', '500', '输出预览最大长度', CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return fmt.Errorf("插入默认配置失败: %w", err)
	}

	return tx.Commit()
}
