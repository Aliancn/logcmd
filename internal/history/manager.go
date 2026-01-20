package history

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aliancn/logcmd/internal/model"
)

// RetentionPolicy 描述日志保留策略。
type RetentionPolicy struct {
	MaxAgeDays   int // 删除多少天之前的记录
	MaxKeepCount int // 仅保留最近多少条
}

// IsZero 判断策略是否为空。
func (p RetentionPolicy) IsZero() bool {
	return p.MaxAgeDays <= 0 && p.MaxKeepCount <= 0
}

const (
	sqlRecordHistory = `
		INSERT INTO command_history (
			project_id, command, command_name, command_args,
			start_time, end_time, duration_ms, exit_code, status,
			log_file_path, log_date,
			stdout_preview, stderr_preview, has_error,
			working_directory, environment_info,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	sqlSelectHistoryBase = `
		SELECT id, project_id, command, command_name, command_args,
			   start_time, end_time, duration_ms, exit_code, status,
			   log_file_path, log_date,
			   stdout_preview, stderr_preview, has_error,
			   working_directory, environment_info,
			   created_at
		FROM command_history
	`

	sqlDeleteHistory          = "DELETE FROM command_history WHERE id = ?"
	sqlDeleteHistoryByProject = "DELETE FROM command_history WHERE project_id = ?"
	sqlDeleteOldHistory       = "DELETE FROM command_history WHERE start_time < ?"
	sqlCountHistory           = "SELECT COUNT(*) FROM command_history"

	// Cleanup Queries
	sqlSelectOldLogPaths = "SELECT log_file_path FROM command_history WHERE start_time < ?"

	sqlSelectExcessLogPaths = `
		SELECT id, log_file_path FROM command_history 
		WHERE id NOT IN (
			SELECT id FROM command_history ORDER BY start_time DESC LIMIT ?
		)
	`

	sqlDeleteExcessHistory = `
		DELETE FROM command_history 
		WHERE id NOT IN (
			SELECT id FROM command_history ORDER BY start_time DESC LIMIT ?
		)
	`
)

// Manager 命令历史管理器
type Manager struct {
	db *sql.DB
}

// NewManager 创建命令历史管理器
func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// Record 记录一条命令执行历史
func (m *Manager) Record(cmd *model.CommandHistory) error {
	if err := cmd.BeforeSave(); err != nil {
		return fmt.Errorf("准备保存数据失败: %w", err)
	}

	_, err := m.db.Exec(sqlRecordHistory,
		cmd.ProjectID,
		cmd.Command,
		cmd.CommandName,
		cmd.ArgsJSON,
		cmd.StartTime,
		cmd.EndTime,
		cmd.DurationMs,
		cmd.ExitCode,
		cmd.Status,
		cmd.LogFilePath,
		cmd.LogDate,
		cmd.StdoutPreview,
		cmd.StderrPreview,
		cmd.HasError,
		cmd.WorkingDirectory,
		cmd.EnvironmentJSON,
		cmd.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("记录命令历史失败: %w", err)
	}

	return nil
}

// QueryOptions 查询选项
type QueryOptions struct {
	ProjectID   int       // 项目ID（0表示所有项目）
	CommandName string    // 命令名称（空表示所有命令）
	Status      string    // 状态（success/failed，空表示所有）
	StartDate   time.Time // 开始日期
	EndDate     time.Time // 结束日期
	Limit       int       // 限制返回数量（0表示不限制）
	Offset      int       // 偏移量
	OrderBy     string    // 排序字段（默认：start_time DESC）
}

// Query 查询命令历史
func (m *Manager) Query(opts QueryOptions) ([]*model.CommandHistory, error) {
	var conditions []string
	var args []interface{}

	// 构建查询条件
	if opts.ProjectID > 0 {
		conditions = append(conditions, "project_id = ?")
		args = append(args, opts.ProjectID)
	}

	if opts.CommandName != "" {
		conditions = append(conditions, "command_name = ?")
		args = append(args, opts.CommandName)
	}

	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}

	if !opts.StartDate.IsZero() {
		conditions = append(conditions, "start_time >= ?")
		args = append(args, opts.StartDate)
	}

	if !opts.EndDate.IsZero() {
		conditions = append(conditions, "start_time <= ?")
		args = append(args, opts.EndDate)
	}

	// 构建SQL查询
	query := sqlSelectHistoryBase

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// 排序
	orderBy := opts.OrderBy
	if orderBy == "" {
		orderBy = "start_time DESC"
	}
	query += " ORDER BY " + orderBy

	// 限制和偏移
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
		if opts.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", opts.Offset)
		}
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	var results []*model.CommandHistory
	for rows.Next() {
		var cmd model.CommandHistory
		err := rows.Scan(
			&cmd.ID,
			&cmd.ProjectID,
			&cmd.Command,
			&cmd.CommandName,
			&cmd.ArgsJSON,
			&cmd.StartTime,
			&cmd.EndTime,
			&cmd.DurationMs,
			&cmd.ExitCode,
			&cmd.Status,
			&cmd.LogFilePath,
			&cmd.LogDate,
			&cmd.StdoutPreview,
			&cmd.StderrPreview,
			&cmd.HasError,
			&cmd.WorkingDirectory,
			&cmd.EnvironmentJSON,
			&cmd.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("读取数据失败: %w", err)
		}

		if err := cmd.AfterLoad(); err != nil {
			return nil, fmt.Errorf("加载数据失败: %w", err)
		}

		results = append(results, &cmd)
	}

	return results, nil
}

// GetByID 根据ID获取命令历史
func (m *Manager) GetByID(id int) (*model.CommandHistory, error) {
	query := sqlSelectHistoryBase + " WHERE id = ?"

	var cmd model.CommandHistory
	err := m.db.QueryRow(query, id).Scan(
		&cmd.ID,
		&cmd.ProjectID,
		&cmd.Command,
		&cmd.CommandName,
		&cmd.ArgsJSON,
		&cmd.StartTime,
		&cmd.EndTime,
		&cmd.DurationMs,
		&cmd.ExitCode,
		&cmd.Status,
		&cmd.LogFilePath,
		&cmd.LogDate,
		&cmd.StdoutPreview,
		&cmd.StderrPreview,
		&cmd.HasError,
		&cmd.WorkingDirectory,
		&cmd.EnvironmentJSON,
		&cmd.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("未找到命令历史: %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}

	if err := cmd.AfterLoad(); err != nil {
		return nil, fmt.Errorf("加载数据失败: %w", err)
	}

	return &cmd, nil
}

// GetRecent 获取最近的命令历史
func (m *Manager) GetRecent(projectID int, limit int) ([]*model.CommandHistory, error) {
	return m.Query(QueryOptions{
		ProjectID: projectID,
		Limit:     limit,
		OrderBy:   "start_time DESC",
	})
}

// GetFailed 获取失败的命令历史
func (m *Manager) GetFailed(projectID int, limit int) ([]*model.CommandHistory, error) {
	return m.Query(QueryOptions{
		ProjectID: projectID,
		Status:    "failed",
		Limit:     limit,
		OrderBy:   "start_time DESC",
	})
}

// GetByDate 获取指定日期的命令历史
func (m *Manager) GetByDate(projectID int, date string) ([]*model.CommandHistory, error) {
	return m.Query(QueryOptions{
		ProjectID: projectID,
		StartDate: parseDate(date),
		EndDate:   parseDate(date).Add(24 * time.Hour),
		OrderBy:   "start_time ASC",
	})
}

// GetCommandStats 获取命令统计信息
func (m *Manager) GetCommandStats(projectID int, startDate, endDate time.Time) (map[string]int, error) {
	query := `
		SELECT command_name, COUNT(*) as count
		FROM command_history
		WHERE project_id = ?
	`
	args := []interface{}{projectID}

	if !startDate.IsZero() {
		query += " AND start_time >= ?"
		args = append(args, startDate)
	}

	if !endDate.IsZero() {
		query += " AND start_time <= ?"
		args = append(args, endDate)
	}

	query += " GROUP BY command_name ORDER BY count DESC"

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询统计失败: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var cmdName string
		var count int
		if err := rows.Scan(&cmdName, &count); err != nil {
			return nil, fmt.Errorf("读取统计数据失败: %w", err)
		}
		stats[cmdName] = count
	}

	return stats, nil
}

// Delete 删除命令历史
func (m *Manager) Delete(id int) error {
	result, err := m.db.Exec(sqlDeleteHistory, id)
	if err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("未找到命令历史: %d", id)
	}

	return nil
}

// DeleteByProject 删除项目的所有命令历史
func (m *Manager) DeleteByProject(projectID int) error {
	_, err := m.db.Exec(sqlDeleteHistoryByProject, projectID)
	if err != nil {
		return fmt.Errorf("删除项目命令历史失败: %w", err)
	}
	return nil
}

// DeleteOldRecords 删除指定天数之前的记录并清理文件
func (m *Manager) DeleteOldRecords(days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	// 1. 获取要删除的日志文件路径
	rows, err := m.db.Query(sqlSelectOldLogPaths, cutoffDate)
	if err != nil {
		return fmt.Errorf("查询旧记录失败: %w", err)
	}

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err == nil && path != "" {
			paths = append(paths, path)
		}
	}
	rows.Close()

	// 2. 删除物理文件
	deletedFiles := 0
	for _, path := range paths {
		if err := os.Remove(path); err == nil {
			deletedFiles++
		}
	}

	// 3. 删除数据库记录
	result, err := m.db.Exec(sqlDeleteOldHistory, cutoffDate)
	if err != nil {
		return fmt.Errorf("删除旧记录失败: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("已清理 %d 条过期记录，删除 %d 个日志文件\n", rowsAffected, deletedFiles)

	return nil
}

// DeleteExcessRecords 保留最近的 N 条记录，删除多余的
func (m *Manager) DeleteExcessRecords(limit int) error {
	if limit <= 0 {
		return nil
	}

	// 1. 获取要删除的日志文件路径
	rows, err := m.db.Query(sqlSelectExcessLogPaths, limit)
	if err != nil {
		return fmt.Errorf("查询多余记录失败: %w", err)
	}

	var paths []string
	for rows.Next() {
		var id int
		var path string
		if err := rows.Scan(&id, &path); err == nil && path != "" {
			paths = append(paths, path)
		}
	}
	rows.Close()

	if len(paths) == 0 {
		fmt.Println("没有需要清理的记录")
		return nil
	}

	// 2. 删除物理文件
	deletedFiles := 0
	for _, path := range paths {
		if err := os.Remove(path); err == nil {
			deletedFiles++
		}
	}

	// 3. 删除数据库记录
	result, err := m.db.Exec(sqlDeleteExcessHistory, limit)
	if err != nil {
		return fmt.Errorf("删除多余记录失败: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("已清理 %d 条多余记录，删除 %d 个日志文件\n", rowsAffected, deletedFiles)

	return nil
}

// ApplyRetention 根据策略执行日志保留逻辑。
func (m *Manager) ApplyRetention(policy RetentionPolicy) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("历史记录管理器未初始化")
	}
	if policy.IsZero() {
		return nil
	}
	if policy.MaxAgeDays > 0 {
		if err := m.DeleteOldRecords(policy.MaxAgeDays); err != nil {
			return err
		}
	}
	if policy.MaxKeepCount > 0 {
		if err := m.DeleteExcessRecords(policy.MaxKeepCount); err != nil {
			return err
		}
	}
	return nil
}

// Count 统计命令历史总数
func (m *Manager) Count(projectID int) (int, error) {
	query := sqlCountHistory
	args := []interface{}{}

	if projectID > 0 {
		query += " WHERE project_id = ?"
		args = append(args, projectID)
	}

	var count int
	err := m.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计失败: %w", err)
	}

	return count, nil
}

// parseDate 解析日期字符串（YYYY-MM-DD）
func parseDate(dateStr string) time.Time {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}
	}
	return t
}
