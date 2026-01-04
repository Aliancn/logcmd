package registry

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/aliancn/logcmd/internal/migration"
	"github.com/aliancn/logcmd/internal/model"
)

// Registry 管理所有项目的注册信息，并负责数据库迁移
type Registry struct {
	db *sql.DB
}

// New 创建一个带自动迁移的Registry实例
func New() (*Registry, error) {
	dbPath, err := getDBPath()
	if err != nil {
		return nil, fmt.Errorf("获取数据库路径失败: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	r := &Registry{db: db}

	// 执行数据库迁移
	migrator := migration.NewMigration(db)
	if err := migrator.Migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return r, nil
}

// getDBPath 获取数据库文件路径
func getDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	logcmdDir := filepath.Join(home, ".logcmd")
	dataDir := filepath.Join(logcmdDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("创建数据目录失败: %w", err)
	}

	return filepath.Join(dataDir, "registry.db"), nil
}

// Register 注册一个项目
func (r *Registry) Register(path string) (*model.Project, error) {
	// 规范化路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 检查目录是否存在
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("目录不存在: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("路径不是目录: %s", absPath)
	}

	// 从路径提取项目名称
	projectName := extractProjectName(absPath)

	now := time.Now()
	query := `
		INSERT INTO projects (path, name, created_at, updated_at, last_checked)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			updated_at = ?,
			last_checked = ?
		RETURNING id, path, name, description, category, tags,
				  total_commands, success_commands, failed_commands, total_duration_ms,
				  last_command, last_command_status, last_command_time,
				  created_at, updated_at, last_checked,
				  template_config, custom_config
	`

	var project model.Project
	err = r.db.QueryRow(query, absPath, projectName, now, now, now, now, now).Scan(
		&project.ID,
		&project.Path,
		&project.Name,
		&project.Description,
		&project.Category,
		&project.TagsJSON,
		&project.TotalCommands,
		&project.SuccessCommands,
		&project.FailedCommands,
		&project.TotalDurationMs,
		&project.LastCommand,
		&project.LastCommandStatus,
		&project.LastCommandTime,
		&project.CreatedAt,
		&project.UpdatedAt,
		&project.LastChecked,
		&project.TemplateJSON,
		&project.CustomJSON,
	)

	if err != nil {
		return nil, fmt.Errorf("注册项目失败: %w", err)
	}

	if err := project.AfterLoad(); err != nil {
		return nil, fmt.Errorf("加载项目数据失败: %w", err)
	}

	return &project, nil
}

// List 列出所有已注册的项目
func (r *Registry) List() ([]*model.Project, error) {
	query := `
		SELECT id, path, name, description, category, tags,
			   total_commands, success_commands, failed_commands, total_duration_ms,
			   last_command, last_command_status, last_command_time,
			   created_at, updated_at, last_checked,
			   template_config, custom_config
		FROM projects
		ORDER BY updated_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	var projects []*model.Project
	for rows.Next() {
		var project model.Project
		err := rows.Scan(
			&project.ID,
			&project.Path,
			&project.Name,
			&project.Description,
			&project.Category,
			&project.TagsJSON,
			&project.TotalCommands,
			&project.SuccessCommands,
			&project.FailedCommands,
			&project.TotalDurationMs,
			&project.LastCommand,
			&project.LastCommandStatus,
			&project.LastCommandTime,
			&project.CreatedAt,
			&project.UpdatedAt,
			&project.LastChecked,
			&project.TemplateJSON,
			&project.CustomJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("读取数据失败: %w", err)
		}

		if err := project.AfterLoad(); err != nil {
			return nil, fmt.Errorf("加载项目数据失败: %w", err)
		}

		projects = append(projects, &project)
	}

	return projects, nil
}

// Get 根据ID或路径获取项目
func (r *Registry) Get(idOrPath string) (*model.Project, error) {
	var query string
	var args []interface{}

	// 尝试解析为ID
	id, err := strconv.Atoi(idOrPath)
	if err == nil {
		// 按ID查询
		query = `
			SELECT id, path, name, description, category, tags,
				   total_commands, success_commands, failed_commands, total_duration_ms,
				   last_command, last_command_status, last_command_time,
				   created_at, updated_at, last_checked,
				   template_config, custom_config
			FROM projects WHERE id = ?
		`
		args = []interface{}{id}
	} else {
		// 按路径查询
		absPath, err := filepath.Abs(idOrPath)
		if err != nil {
			return nil, fmt.Errorf("获取绝对路径失败: %w", err)
		}
		query = `
			SELECT id, path, name, description, category, tags,
				   total_commands, success_commands, failed_commands, total_duration_ms,
				   last_command, last_command_status, last_command_time,
				   created_at, updated_at, last_checked,
				   template_config, custom_config
			FROM projects WHERE path = ?
		`
		args = []interface{}{absPath}
	}

	var project model.Project
	err = r.db.QueryRow(query, args...).Scan(
		&project.ID,
		&project.Path,
		&project.Name,
		&project.Description,
		&project.Category,
		&project.TagsJSON,
		&project.TotalCommands,
		&project.SuccessCommands,
		&project.FailedCommands,
		&project.TotalDurationMs,
		&project.LastCommand,
		&project.LastCommandStatus,
		&project.LastCommandTime,
		&project.CreatedAt,
		&project.UpdatedAt,
		&project.LastChecked,
		&project.TemplateJSON,
		&project.CustomJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("未找到项目: %s", idOrPath)
	}
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}

	if err := project.AfterLoad(); err != nil {
		return nil, fmt.Errorf("加载项目数据失败: %w", err)
	}

	return &project, nil
}

// Update 更新项目信息
func (r *Registry) Update(project *model.Project) error {
	if err := project.BeforeSave(); err != nil {
		return fmt.Errorf("准备保存数据失败: %w", err)
	}

	project.UpdatedAt = time.Now()

	query := `
		UPDATE projects SET
			name = ?,
			description = ?,
			category = ?,
			tags = ?,
			total_commands = ?,
			success_commands = ?,
			failed_commands = ?,
			total_duration_ms = ?,
			last_command = ?,
			last_command_status = ?,
			last_command_time = ?,
			updated_at = ?,
			template_config = ?,
			custom_config = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query,
		project.Name,
		project.Description,
		project.Category,
		project.TagsJSON,
		project.TotalCommands,
		project.SuccessCommands,
		project.FailedCommands,
		project.TotalDurationMs,
		project.LastCommand,
		project.LastCommandStatus,
		project.LastCommandTime,
		project.UpdatedAt,
		project.TemplateJSON,
		project.CustomJSON,
		project.ID,
	)

	if err != nil {
		return fmt.Errorf("更新项目失败: %w", err)
	}

	return nil
}

// UpdateStats 更新项目统计信息（在命令执行后调用）
func (r *Registry) UpdateStats(projectID int, command string, success bool, duration time.Duration) error {
	status := "success"
	if !success {
		status = "failed"
	}

	now := time.Now()

	query := `
		UPDATE projects SET
			total_commands = total_commands + 1,
			success_commands = success_commands + CASE WHEN ? THEN 1 ELSE 0 END,
			failed_commands = failed_commands + CASE WHEN ? THEN 0 ELSE 1 END,
			total_duration_ms = total_duration_ms + ?,
			last_command = ?,
			last_command_status = ?,
			last_command_time = ?,
			updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query,
		success,
		success,
		duration.Milliseconds(),
		command,
		status,
		now,
		now,
		projectID,
	)

	if err != nil {
		return fmt.Errorf("更新统计信息失败: %w", err)
	}

	return nil
}

// Delete 删除指定的项目，并清理其日志目录
func (r *Registry) Delete(idOrPath string) error {
	project, err := r.Get(idOrPath)
	if err != nil {
		return err
	}

	result, err := r.db.Exec(`DELETE FROM projects WHERE id = ?`, project.ID)
	if err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("未找到项目: %s", idOrPath)
	}

	return nil
}

// CheckAndCleanup 检查所有目录是否仍然存在，删除不存在的条目
func (r *Registry) CheckAndCleanup() error {
	projects, err := r.List()
	if err != nil {
		return err
	}

	for _, project := range projects {
		// 检查目录是否存在
		if _, err := os.Stat(project.Path); os.IsNotExist(err) {
			// 目录不存在，删除条目
			if err := r.Delete(fmt.Sprintf("%d", project.ID)); err != nil {
				return fmt.Errorf("删除无效项目失败 [%d: %s]: %w", project.ID, project.Path, err)
			}
		} else {
			// 更新检查时间
			project.LastChecked = time.Now()
			query := `UPDATE projects SET last_checked = ? WHERE id = ?`
			if _, err := r.db.Exec(query, project.LastChecked, project.ID); err != nil {
				return fmt.Errorf("更新检查时间失败: %w", err)
			}
		}
	}

	return nil
}

// Close 关闭数据库连接
func (r *Registry) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// UpdateLastChecked 更新项目的最后检查时间
func (r *Registry) UpdateLastChecked(idOrPath string) error {
	var query string
	var args []interface{}

	now := time.Now()

	if id, err := strconv.Atoi(idOrPath); err == nil {
		query = `UPDATE projects SET last_checked = ? WHERE id = ?`
		args = []interface{}{now, id}
	} else {
		absPath, err := filepath.Abs(idOrPath)
		if err != nil {
			return fmt.Errorf("获取绝对路径失败: %w", err)
		}
		query = `UPDATE projects SET last_checked = ? WHERE path = ?`
		args = []interface{}{now, absPath}
	}

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("更新检查时间失败: %w", err)
	}

	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return fmt.Errorf("未找到项目: %s", idOrPath)
	}

	return nil
}

// extractProjectName 从路径中提取项目名称
func extractProjectName(path string) string {
	sep := string(os.PathSeparator)
	suffix := sep + ".logcmd"
	if strings.HasSuffix(path, suffix) {
		path = strings.TrimSuffix(path, suffix)
	}
	return filepath.Base(path)
}

// GetDB 获取数据库连接（供其他模块使用）
func (r *Registry) GetDB() *sql.DB {
	return r.db
}
