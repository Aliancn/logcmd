package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aliancn/logcmd/internal/domain/model"
	"github.com/aliancn/logcmd/internal/platform/logparser"
	"github.com/aliancn/logcmd/internal/platform/registry"
	"github.com/aliancn/logcmd/internal/platform/stats"
	"github.com/aliancn/logcmd/internal/platform/walker"
)

// SyncReport 表示一次同步的整体结果。
type SyncReport struct {
	CleanedProjects int
	ProjectCount    int
	LogFiles        int
	Inserted        int
	Updated         int
	Deleted         int
	Skipped         int
	Projects        []*ProjectSyncResult
}

// ProjectSyncResult 表示单个项目的同步结果。
type ProjectSyncResult struct {
	ProjectID int
	Path      string
	LogFiles  int
	Inserted  int
	Updated   int
	Deleted   int
	Skipped   int
}

// SyncService 负责根据日志文件同步数据库。
type SyncService struct {
	registry *registry.Registry
	db       *sql.DB
	cache    *stats.CacheManager
}

// NewSyncService 创建同步服务。
func NewSyncService(reg *registry.Registry) *SyncService {
	if reg == nil {
		return nil
	}
	db := reg.GetDB()
	return &SyncService{
		registry: reg,
		db:       db,
		cache:    stats.NewCacheManager(db),
	}
}

// SyncAll 同步所有已注册项目。
func (s *SyncService) SyncAll(ctx context.Context) (*SyncReport, error) {
	if s == nil || s.registry == nil {
		return nil, fmt.Errorf("同步服务未初始化")
	}

	projectsBefore, err := s.registry.List()
	if err != nil {
		return nil, fmt.Errorf("获取项目列表失败: %w", err)
	}

	if err := s.registry.CheckAndCleanup(); err != nil {
		return nil, fmt.Errorf("清理无效项目失败: %w", err)
	}

	projects, err := s.registry.List()
	if err != nil {
		return nil, fmt.Errorf("获取项目列表失败: %w", err)
	}

	report := &SyncReport{
		CleanedProjects: len(projectsBefore) - len(projects),
		ProjectCount:    len(projects),
	}

	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		result, err := s.SyncProject(ctx, project)
		if err != nil {
			return nil, err
		}
		report.Projects = append(report.Projects, result)
		report.LogFiles += result.LogFiles
		report.Inserted += result.Inserted
		report.Updated += result.Updated
		report.Deleted += result.Deleted
		report.Skipped += result.Skipped
	}

	return report, nil
}

// SyncProject 同步单个项目的日志记录与数据库。
func (s *SyncService) SyncProject(ctx context.Context, project *model.Project) (*ProjectSyncResult, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	if project == nil {
		return nil, fmt.Errorf("项目不能为空")
	}

	logFiles, err := collectLogFiles(ctx, project.Path)
	if err != nil {
		return nil, fmt.Errorf("扫描日志失败: %w", err)
	}

	existing, err := s.loadHistoryIndex(project.ID)
	if err != nil {
		return nil, fmt.Errorf("读取历史记录失败: %w", err)
	}

	result := &ProjectSyncResult{
		ProjectID: project.ID,
		Path:      project.Path,
		LogFiles:  len(logFiles),
	}

	seen := make(map[string]struct{}, len(logFiles))

	for _, logPath := range logFiles {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		seen[logPath] = struct{}{}
		meta, err := logparser.ParseFile(ctx, logPath)
		if err != nil {
			result.Skipped++
			continue
		}

		record, err := buildHistoryRecord(project, logPath, meta)
		if err != nil {
			result.Skipped++
			continue
		}

		if ids, ok := existing[logPath]; ok && len(ids) > 0 {
			if err := s.updateHistory(ids[0], record); err != nil {
				return nil, err
			}
			result.Updated++
			if len(ids) > 1 {
				if err := s.deleteHistoryIDs(ids[1:]); err != nil {
					return nil, err
				}
				result.Deleted += len(ids) - 1
			}
		} else {
			if err := s.insertHistory(record); err != nil {
				return nil, err
			}
			result.Inserted++
		}
	}

	for logPath, ids := range existing {
		if _, ok := seen[logPath]; ok {
			continue
		}
		if err := s.deleteHistoryIDs(ids); err != nil {
			return nil, err
		}
		result.Deleted += len(ids)
	}

	if err := s.updateProjectStats(project.ID); err != nil {
		return nil, err
	}

	if err := s.syncProjectCache(project.ID); err != nil {
		return nil, err
	}

	return result, nil
}

func collectLogFiles(ctx context.Context, root string) ([]string, error) {
	fileWalker, err := walker.New(walker.Options{
		Root:    root,
		Workers: 1,
		FileFilter: func(path string, info os.FileInfo) bool {
			return strings.HasSuffix(path, ".log")
		},
	})
	if err != nil {
		return nil, err
	}

	var files []string
	err = fileWalker.Walk(ctx, func(ctx context.Context, path string, info os.FileInfo) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		if ctx.Err() != nil && err == ctx.Err() {
			return nil, err
		}
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func (s *SyncService) loadHistoryIndex(projectID int) (map[string][]int, error) {
	rows, err := s.db.Query(`SELECT id, log_file_path FROM command_history WHERE project_id = ?`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	index := make(map[string][]int)
	for rows.Next() {
		var id int
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, err
		}
		index[path] = append(index[path], id)
	}
	return index, nil
}

func buildHistoryRecord(project *model.Project, logPath string, meta *logparser.Metadata) (*model.CommandHistory, error) {
	if meta == nil {
		return nil, fmt.Errorf("日志元数据为空")
	}
	if meta.CommandLine == "" {
		return nil, fmt.Errorf("日志缺少命令信息")
	}
	if meta.StartTime.IsZero() || meta.EndTime.IsZero() {
		return nil, fmt.Errorf("日志缺少时间信息")
	}
	if !meta.ExitCodeSet {
		return nil, fmt.Errorf("日志缺少退出码")
	}

	duration := meta.Duration
	if duration == 0 {
		duration = meta.EndTime.Sub(meta.StartTime)
	}

	status := "failed"
	if meta.Success {
		status = "success"
	}

	workingDir := ""
	if project != nil {
		workingDir = filepath.Dir(project.Path)
	}

	logDate := meta.LogDate
	if logDate == "" {
		logDate = meta.StartTime.Format("2006-01-02")
	}

	record := &model.CommandHistory{
		ProjectID:        project.ID,
		Command:          meta.CommandLine,
		CommandArgs:      meta.CommandArgs,
		StartTime:        meta.StartTime,
		EndTime:          meta.EndTime,
		DurationMs:       duration.Milliseconds(),
		ExitCode:         meta.ExitCode,
		Status:           status,
		LogFilePath:      logPath,
		LogDate:          logDate,
		HasError:         !meta.Success,
		WorkingDirectory: workingDir,
		CreatedAt:        time.Now(),
	}

	return record, nil
}

func (s *SyncService) insertHistory(cmd *model.CommandHistory) error {
	if err := cmd.BeforeSave(); err != nil {
		return fmt.Errorf("准备保存数据失败: %w", err)
	}

	const sqlRecordHistory = `
		INSERT INTO command_history (
			project_id, command, command_name, command_args,
			start_time, end_time, duration_ms, exit_code, status,
			log_file_path, log_date,
			stdout_preview, stderr_preview, has_error,
			working_directory, environment_info,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(sqlRecordHistory,
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

func (s *SyncService) updateHistory(id int, cmd *model.CommandHistory) error {
	if err := cmd.BeforeSave(); err != nil {
		return fmt.Errorf("准备保存数据失败: %w", err)
	}

	const sqlUpdateHistory = `
		UPDATE command_history SET
			command = ?,
			command_name = ?,
			command_args = ?,
			start_time = ?,
			end_time = ?,
			duration_ms = ?,
			exit_code = ?,
			status = ?,
			log_file_path = ?,
			log_date = ?,
			stdout_preview = ?,
			stderr_preview = ?,
			has_error = ?,
			working_directory = ?,
			environment_info = ?
		WHERE id = ?
	`

	_, err := s.db.Exec(sqlUpdateHistory,
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
		id,
	)
	if err != nil {
		return fmt.Errorf("更新命令历史失败: %w", err)
	}
	return nil
}

func (s *SyncService) deleteHistoryIDs(ids []int) error {
	for _, id := range ids {
		if _, err := s.db.Exec(`DELETE FROM command_history WHERE id = ?`, id); err != nil {
			return fmt.Errorf("删除命令历史失败: %w", err)
		}
	}
	return nil
}

func (s *SyncService) updateProjectStats(projectID int) error {
	const sqlStats = `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
			SUM(duration_ms) as total_duration
		FROM command_history
		WHERE project_id = ?
	`

	var total, success, failed int
	var totalDuration sql.NullInt64
	if err := s.db.QueryRow(sqlStats, projectID).Scan(&total, &success, &failed, &totalDuration); err != nil {
		return fmt.Errorf("查询项目统计失败: %w", err)
	}

	const sqlLast = `
		SELECT command, status, end_time
		FROM command_history
		WHERE project_id = ?
		ORDER BY end_time DESC
		LIMIT 1
	`

	var lastCommand, lastStatus string
	var lastTime time.Time
	lastTimeValid := false

	if err := s.db.QueryRow(sqlLast, projectID).Scan(&lastCommand, &lastStatus, &lastTime); err != nil {
		if err != sql.ErrNoRows {
			return fmt.Errorf("查询最后命令失败: %w", err)
		}
	} else {
		lastTimeValid = true
	}

	now := time.Now()
	var lastTimeValue interface{} = nil
	if lastTimeValid {
		lastTimeValue = lastTime
	}

	const sqlUpdateProject = `
		UPDATE projects SET
			total_commands = ?,
			success_commands = ?,
			failed_commands = ?,
			total_duration_ms = ?,
			last_command = ?,
			last_command_status = ?,
			last_command_time = ?,
			updated_at = ?,
			last_checked = ?
		WHERE id = ?
	`

	totalDurationValue := int64(0)
	if totalDuration.Valid {
		totalDurationValue = totalDuration.Int64
	}

	if _, err := s.db.Exec(sqlUpdateProject,
		total,
		success,
		failed,
		totalDurationValue,
		lastCommand,
		lastStatus,
		lastTimeValue,
		now,
		now,
		projectID,
	); err != nil {
		return fmt.Errorf("更新项目统计失败: %w", err)
	}

	return nil
}

func (s *SyncService) syncProjectCache(projectID int) error {
	if s.cache == nil {
		return nil
	}
	if err := s.cache.DeleteByProject(projectID); err != nil {
		return fmt.Errorf("清理项目统计缓存失败: %w", err)
	}
	if err := s.cache.GenerateForProject(projectID); err != nil {
		return fmt.Errorf("生成项目统计缓存失败: %w", err)
	}
	return nil
}
