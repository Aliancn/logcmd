package stats

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aliancn/logcmd/internal/model"
)

// CacheManager 统计缓存管理器
type CacheManager struct {
	db *sql.DB
}

// NewCacheManager 创建统计缓存管理器
func NewCacheManager(db *sql.DB) *CacheManager {
	return &CacheManager{db: db}
}

// GenerateForDate 为指定日期生成统计缓存
func (m *CacheManager) GenerateForDate(projectID int, date string) error {
	// 从命令历史中统计数据
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
			SUM(duration_ms) as total_duration,
			AVG(duration_ms) as avg_duration,
			MAX(duration_ms) as max_duration,
			MIN(duration_ms) as min_duration
		FROM command_history
		WHERE project_id = ? AND log_date = ?
	`

	var total, success, failed int
	var totalDuration, maxDuration, minDuration sql.NullInt64
	var avgDuration sql.NullFloat64

	err := m.db.QueryRow(query, projectID, date).Scan(
		&total, &success, &failed,
		&totalDuration, &avgDuration, &maxDuration, &minDuration,
	)
	if err != nil {
		return fmt.Errorf("查询统计数据失败: %w", err)
	}

	// 如果没有数据，不生成缓存
	if total == 0 {
		return nil
	}

	// 获取命令分布
	cmdDist, err := m.getCommandDistribution(projectID, date)
	if err != nil {
		return fmt.Errorf("获取命令分布失败: %w", err)
	}

	// 获取退出码分布
	exitDist, err := m.getExitCodeDistribution(projectID, date)
	if err != nil {
		return fmt.Errorf("获取退出码分布失败: %w", err)
	}

	// 创建统计缓存对象
	cache := &model.ProjectStatsCache{
		ProjectID:            projectID,
		StatDate:             date,
		TotalCommands:        total,
		SuccessCommands:      success,
		FailedCommands:       failed,
		TotalDurationMs:      totalDuration.Int64,
		AvgDurationMs:        int64(avgDuration.Float64),
		MaxDurationMs:        maxDuration.Int64,
		MinDurationMs:        minDuration.Int64,
		CommandDistribution:  cmdDist,
		ExitCodeDistribution: exitDist,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	// 保存到数据库
	return m.Save(cache)
}

// getCommandDistribution 获取命令分布
func (m *CacheManager) getCommandDistribution(projectID int, date string) (map[string]int, error) {
	query := `
		SELECT command_name, COUNT(*) as count
		FROM command_history
		WHERE project_id = ? AND log_date = ?
		GROUP BY command_name
	`

	rows, err := m.db.Query(query, projectID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dist := make(map[string]int)
	for rows.Next() {
		var cmdName string
		var count int
		if err := rows.Scan(&cmdName, &count); err != nil {
			return nil, err
		}
		dist[cmdName] = count
	}

	return dist, nil
}

// getExitCodeDistribution 获取退出码分布
func (m *CacheManager) getExitCodeDistribution(projectID int, date string) (map[int]int, error) {
	query := `
		SELECT exit_code, COUNT(*) as count
		FROM command_history
		WHERE project_id = ? AND log_date = ?
		GROUP BY exit_code
	`

	rows, err := m.db.Query(query, projectID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dist := make(map[int]int)
	for rows.Next() {
		var exitCode int
		var count int
		if err := rows.Scan(&exitCode, &count); err != nil {
			return nil, err
		}
		dist[exitCode] = count
	}

	return dist, nil
}

// Save 保存统计缓存
func (m *CacheManager) Save(cache *model.ProjectStatsCache) error {
	if err := cache.BeforeSave(); err != nil {
		return fmt.Errorf("准备保存数据失败: %w", err)
	}

	query := `
		INSERT INTO project_stats_cache (
			project_id, stat_date,
			total_commands, success_commands, failed_commands,
			total_duration_ms, avg_duration_ms, max_duration_ms, min_duration_ms,
			command_distribution, exit_code_distribution,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, stat_date) DO UPDATE SET
			total_commands = excluded.total_commands,
			success_commands = excluded.success_commands,
			failed_commands = excluded.failed_commands,
			total_duration_ms = excluded.total_duration_ms,
			avg_duration_ms = excluded.avg_duration_ms,
			max_duration_ms = excluded.max_duration_ms,
			min_duration_ms = excluded.min_duration_ms,
			command_distribution = excluded.command_distribution,
			exit_code_distribution = excluded.exit_code_distribution,
			updated_at = excluded.updated_at
	`

	_, err := m.db.Exec(query,
		cache.ProjectID,
		cache.StatDate,
		cache.TotalCommands,
		cache.SuccessCommands,
		cache.FailedCommands,
		cache.TotalDurationMs,
		cache.AvgDurationMs,
		cache.MaxDurationMs,
		cache.MinDurationMs,
		cache.CommandDistJSON,
		cache.ExitCodeDistJSON,
		cache.CreatedAt,
		cache.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("保存统计缓存失败: %w", err)
	}

	return nil
}

// Get 获取指定日期的统计缓存
func (m *CacheManager) Get(projectID int, date string) (*model.ProjectStatsCache, error) {
	query := `
		SELECT id, project_id, stat_date,
			   total_commands, success_commands, failed_commands,
			   total_duration_ms, avg_duration_ms, max_duration_ms, min_duration_ms,
			   command_distribution, exit_code_distribution,
			   created_at, updated_at
		FROM project_stats_cache
		WHERE project_id = ? AND stat_date = ?
	`

	var cache model.ProjectStatsCache
	err := m.db.QueryRow(query, projectID, date).Scan(
		&cache.ID,
		&cache.ProjectID,
		&cache.StatDate,
		&cache.TotalCommands,
		&cache.SuccessCommands,
		&cache.FailedCommands,
		&cache.TotalDurationMs,
		&cache.AvgDurationMs,
		&cache.MaxDurationMs,
		&cache.MinDurationMs,
		&cache.CommandDistJSON,
		&cache.ExitCodeDistJSON,
		&cache.CreatedAt,
		&cache.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // 没有缓存，返回 nil 而不是错误
	}
	if err != nil {
		return nil, fmt.Errorf("查询统计缓存失败: %w", err)
	}

	if err := cache.AfterLoad(); err != nil {
		return nil, fmt.Errorf("加载缓存数据失败: %w", err)
	}

	return &cache, nil
}

// GetRange 获取日期范围内的统计缓存
func (m *CacheManager) GetRange(projectID int, startDate, endDate string) ([]*model.ProjectStatsCache, error) {
	query := `
		SELECT id, project_id, stat_date,
			   total_commands, success_commands, failed_commands,
			   total_duration_ms, avg_duration_ms, max_duration_ms, min_duration_ms,
			   command_distribution, exit_code_distribution,
			   created_at, updated_at
		FROM project_stats_cache
		WHERE project_id = ? AND stat_date BETWEEN ? AND ?
		ORDER BY stat_date ASC
	`

	rows, err := m.db.Query(query, projectID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("查询统计缓存失败: %w", err)
	}
	defer rows.Close()

	var caches []*model.ProjectStatsCache
	for rows.Next() {
		var cache model.ProjectStatsCache
		err := rows.Scan(
			&cache.ID,
			&cache.ProjectID,
			&cache.StatDate,
			&cache.TotalCommands,
			&cache.SuccessCommands,
			&cache.FailedCommands,
			&cache.TotalDurationMs,
			&cache.AvgDurationMs,
			&cache.MaxDurationMs,
			&cache.MinDurationMs,
			&cache.CommandDistJSON,
			&cache.ExitCodeDistJSON,
			&cache.CreatedAt,
			&cache.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("读取缓存数据失败: %w", err)
		}

		if err := cache.AfterLoad(); err != nil {
			return nil, fmt.Errorf("加载缓存数据失败: %w", err)
		}

		caches = append(caches, &cache)
	}

	return caches, nil
}

// GetOrGenerate 获取统计缓存，如果不存在则生成
func (m *CacheManager) GetOrGenerate(projectID int, date string) (*model.ProjectStatsCache, error) {
	cache, err := m.Get(projectID, date)
	if err != nil {
		return nil, err
	}

	if cache == nil {
		// 生成缓存
		if err := m.GenerateForDate(projectID, date); err != nil {
			return nil, fmt.Errorf("生成统计缓存失败: %w", err)
		}
		// 重新获取
		cache, err = m.Get(projectID, date)
		if err != nil {
			return nil, err
		}
	}

	return cache, nil
}

// Sync 确保所有历史记录都有对应的统计缓存
func (m *CacheManager) Sync(projectID int) error {
	// 找出有历史记录但没有缓存的日期
	query := `
		SELECT DISTINCT h.log_date
		FROM command_history h
		LEFT JOIN project_stats_cache s ON h.project_id = s.project_id AND h.log_date = s.stat_date
		WHERE h.project_id = ? AND s.id IS NULL
	`

	rows, err := m.db.Query(query, projectID)
	if err != nil {
		return fmt.Errorf("查询缺失的统计缓存日期失败: %w", err)
	}
	defer rows.Close()

	var missingDates []string
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			return fmt.Errorf("读取日期失败: %w", err)
		}
		missingDates = append(missingDates, date)
	}

	// 为缺失的日期生成缓存
	for _, date := range missingDates {
		if err := m.GenerateForDate(projectID, date); err != nil {
			return fmt.Errorf("生成日期 %s 的缓存失败: %w", date, err)
		}
	}

	return nil
}

// GenerateForProject 为项目生成所有日期的统计缓存
func (m *CacheManager) GenerateForProject(projectID int) error {
	// 获取项目中所有的日期
	query := `
		SELECT DISTINCT log_date
		FROM command_history
		WHERE project_id = ?
		ORDER BY log_date
	`

	rows, err := m.db.Query(query, projectID)
	if err != nil {
		return fmt.Errorf("查询日期列表失败: %w", err)
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			return fmt.Errorf("读取日期失败: %w", err)
		}
		dates = append(dates, date)
	}

	// 为每个日期生成缓存
	for _, date := range dates {
		if err := m.GenerateForDate(projectID, date); err != nil {
			return fmt.Errorf("生成日期 %s 的缓存失败: %w", date, err)
		}
	}

	return nil
}

// Delete 删除统计缓存
func (m *CacheManager) Delete(projectID int, date string) error {
	result, err := m.db.Exec(
		"DELETE FROM project_stats_cache WHERE project_id = ? AND stat_date = ?",
		projectID, date,
	)
	if err != nil {
		return fmt.Errorf("删除统计缓存失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("未找到统计缓存")
	}

	return nil
}

// DeleteByProject 删除项目的所有统计缓存
func (m *CacheManager) DeleteByProject(projectID int) error {
	_, err := m.db.Exec("DELETE FROM project_stats_cache WHERE project_id = ?", projectID)
	if err != nil {
		return fmt.Errorf("删除项目统计缓存失败: %w", err)
	}
	return nil
}

// GetSummary 获取汇总统计（多个日期的合并统计）
func (m *CacheManager) GetSummary(projectID int, startDate, endDate string) (*model.ProjectStatsCache, error) {
	caches, err := m.GetRange(projectID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if len(caches) == 0 {
		return nil, nil
	}

	// 合并统计数据
	summary := &model.ProjectStatsCache{
		ProjectID:            projectID,
		StatDate:             fmt.Sprintf("%s to %s", startDate, endDate),
		CommandDistribution:  make(map[string]int),
		ExitCodeDistribution: make(map[int]int),
	}

	var maxDur, minDur int64 = 0, 999999999

	for _, cache := range caches {
		summary.TotalCommands += cache.TotalCommands
		summary.SuccessCommands += cache.SuccessCommands
		summary.FailedCommands += cache.FailedCommands
		summary.TotalDurationMs += cache.TotalDurationMs

		if cache.MaxDurationMs > maxDur {
			maxDur = cache.MaxDurationMs
		}
		if cache.MinDurationMs < minDur && cache.MinDurationMs > 0 {
			minDur = cache.MinDurationMs
		}

		// 合并命令分布
		for cmd, count := range cache.CommandDistribution {
			summary.CommandDistribution[cmd] += count
		}

		// 合并退出码分布
		for code, count := range cache.ExitCodeDistribution {
			summary.ExitCodeDistribution[code] += count
		}
	}

	summary.MaxDurationMs = maxDur
	summary.MinDurationMs = minDur
	if summary.TotalCommands > 0 {
		summary.AvgDurationMs = summary.TotalDurationMs / int64(summary.TotalCommands)
	}

	// 序列化 JSON 字段
	if err := summary.BeforeSave(); err != nil {
		return nil, err
	}

	return summary, nil
}

// GetProjectSummary 获取指定项目所有缓存日期的汇总
func (m *CacheManager) GetProjectSummary(projectID int) (*model.ProjectStatsCache, error) {
	query := `
		SELECT MIN(stat_date), MAX(stat_date)
		FROM project_stats_cache
		WHERE project_id = ?
	`

	var minDate, maxDate sql.NullString
	if err := m.db.QueryRow(query, projectID).Scan(&minDate, &maxDate); err != nil {
		return nil, fmt.Errorf("查询统计范围失败: %w", err)
	}

	if !minDate.Valid || !maxDate.Valid {
		return nil, nil
	}

	return m.GetSummary(projectID, minDate.String, maxDate.String)
}

// ExportToJSON 导出统计缓存为 JSON
func (m *CacheManager) ExportToJSON(projectID int, startDate, endDate string) (string, error) {
	caches, err := m.GetRange(projectID, startDate, endDate)
	if err != nil {
		return "", err
	}

	jsonData, err := json.MarshalIndent(caches, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化JSON失败: %w", err)
	}

	return string(jsonData), nil
}
