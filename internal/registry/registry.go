package registry

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// LogcmdEntry 表示一个已注册的.logcmd目录
type LogcmdEntry struct {
	ID          int       // 编号
	Path        string    // .logcmd目录路径
	CreatedAt   time.Time // 创建时间
	UpdatedAt   time.Time // 最后更新时间
	LastChecked time.Time // 最后检查时间
}

// Registry 管理所有.logcmd目录的注册信息
type Registry struct {
	db *sql.DB
}

// New 创建一个新的Registry实例
func New() (*Registry, error) {
	dbPath, err := getDBPath()
	if err != nil {
		return nil, fmt.Errorf("获取数据库路径失败: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	r := &Registry{db: db}
	if err := r.initDB(); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化数据库失败: %w", err)
	}

	return r, nil
}

// getDBPath 获取数据库文件路径
func getDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".logcmd_registry.db"), nil
}

// initDB 初始化数据库表
func (r *Registry) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS logcmd_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL UNIQUE,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		last_checked TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_path ON logcmd_entries(path);
	`

	_, err := r.db.Exec(schema)
	return err
}

// Register 注册一个.logcmd目录
func (r *Registry) Register(path string) error {
	// 规范化路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 检查目录是否存在
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("目录不存在: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("路径不是目录: %s", absPath)
	}

	now := time.Now()
	query := `
		INSERT INTO logcmd_entries (path, created_at, updated_at, last_checked)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			updated_at = ?,
			last_checked = ?
	`

	_, err = r.db.Exec(query, absPath, now, now, now, now, now)
	if err != nil {
		return fmt.Errorf("注册目录失败: %w", err)
	}

	return nil
}

// List 列出所有已注册的.logcmd目录
func (r *Registry) List() ([]LogcmdEntry, error) {
	query := `SELECT id, path, created_at, updated_at, last_checked FROM logcmd_entries ORDER BY id`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	var entries []LogcmdEntry
	for rows.Next() {
		var entry LogcmdEntry
		err := rows.Scan(&entry.ID, &entry.Path, &entry.CreatedAt, &entry.UpdatedAt, &entry.LastChecked)
		if err != nil {
			return nil, fmt.Errorf("读取数据失败: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Get 根据ID或路径获取条目
func (r *Registry) Get(idOrPath string) (*LogcmdEntry, error) {
	var entry LogcmdEntry
	var query string
	var args []interface{}

	// 尝试解析为ID
	var id int
	_, err := fmt.Sscanf(idOrPath, "%d", &id)
	if err == nil {
		// 按ID查询
		query = `SELECT id, path, created_at, updated_at, last_checked FROM logcmd_entries WHERE id = ?`
		args = []interface{}{id}
	} else {
		// 按路径查询
		absPath, err := filepath.Abs(idOrPath)
		if err != nil {
			return nil, fmt.Errorf("获取绝对路径失败: %w", err)
		}
		query = `SELECT id, path, created_at, updated_at, last_checked FROM logcmd_entries WHERE path = ?`
		args = []interface{}{absPath}
	}

	err = r.db.QueryRow(query, args...).Scan(&entry.ID, &entry.Path, &entry.CreatedAt, &entry.UpdatedAt, &entry.LastChecked)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("未找到目录: %s", idOrPath)
	}
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}

	return &entry, nil
}

// Delete 删除指定的.logcmd目录（支持ID或路径）
func (r *Registry) Delete(idOrPath string) error {
	var query string
	var args []interface{}

	// 尝试解析为ID
	var id int
	_, err := fmt.Sscanf(idOrPath, "%d", &id)
	if err == nil {
		// 按ID删除
		query = `DELETE FROM logcmd_entries WHERE id = ?`
		args = []interface{}{id}
	} else {
		// 按路径删除
		absPath, err := filepath.Abs(idOrPath)
		if err != nil {
			return fmt.Errorf("获取绝对路径失败: %w", err)
		}
		query = `DELETE FROM logcmd_entries WHERE path = ?`
		args = []interface{}{absPath}
	}

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("未找到目录: %s", idOrPath)
	}

	return nil
}

// DeleteAll 删除所有已注册的目录
func (r *Registry) DeleteAll() error {
	_, err := r.db.Exec(`DELETE FROM logcmd_entries`)
	if err != nil {
		return fmt.Errorf("删除所有目录失败: %w", err)
	}
	return nil
}

// UpdateLastChecked 更新最后检查时间（懒更新）
func (r *Registry) UpdateLastChecked(idOrPath string) error {
	var query string
	var args []interface{}

	now := time.Now()

	// 尝试解析为ID
	var id int
	_, err := fmt.Sscanf(idOrPath, "%d", &id)
	if err == nil {
		// 按ID更新
		query = `UPDATE logcmd_entries SET last_checked = ? WHERE id = ?`
		args = []interface{}{now, id}
	} else {
		// 按路径更新
		absPath, err := filepath.Abs(idOrPath)
		if err != nil {
			return fmt.Errorf("获取绝对路径失败: %w", err)
		}
		query = `UPDATE logcmd_entries SET last_checked = ? WHERE path = ?`
		args = []interface{}{now, absPath}
	}

	_, err = r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("更新检查时间失败: %w", err)
	}

	return nil
}

// CheckAndCleanup 检查所有目录是否仍然存在，删除不存在的条目（懒更新检查）
func (r *Registry) CheckAndCleanup() error {
	entries, err := r.List()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// 检查目录是否存在
		if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
			// 目录不存在，删除条目
			if err := r.Delete(fmt.Sprintf("%d", entry.ID)); err != nil {
				return fmt.Errorf("删除无效目录失败 [%d: %s]: %w", entry.ID, entry.Path, err)
			}
		} else {
			// 更新检查时间
			if err := r.UpdateLastChecked(fmt.Sprintf("%d", entry.ID)); err != nil {
				return fmt.Errorf("更新检查时间失败 [%d: %s]: %w", entry.ID, entry.Path, err)
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
