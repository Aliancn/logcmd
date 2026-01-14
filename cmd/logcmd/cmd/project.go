package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/template"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "管理已注册的项目",
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有项目",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listProjects()
	},
}

var projectCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "清理不存在的项目或旧日志",
	RunE: func(cmd *cobra.Command, args []string) error {
		if projectCleanDays <= 0 && projectCleanKeep <= 0 {
			return cleanProjects()
		}
		return runProjectClean()
	},
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <id|path>",
	Short: "删除指定项目",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteProject(args[0])
	},
}

var projectDeleteForce bool
var (
	projectCleanDays  int
	projectCleanKeep  int
	projectCleanForce bool
)

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectCleanCmd)
	projectCmd.AddCommand(projectDeleteCmd)

	projectDeleteCmd.Flags().BoolVar(&projectDeleteForce, "force", false, "跳过确认直接删除项目及日志目录")
	projectCleanCmd.Flags().IntVar(&projectCleanDays, "days", 0, "删除超过指定天数的日志")
	projectCleanCmd.Flags().IntVar(&projectCleanKeep, "keep", 0, "仅保留最近的 N 条记录")
	projectCleanCmd.Flags().BoolVarP(&projectCleanForce, "force", "f", false, "跳过 project clean 的确认提示")
}

func listProjects() error {
	services, err := newCLIServices()
	if err != nil {
		return err
	}
	defer services.Close()
	reg := services.Registry()

	entries, err := reg.List()
	if err != nil {
		return fmt.Errorf("列出项目失败: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("没有已注册的项目")
		return nil
	}

	rows := make([]projectRow, 0, len(entries))
	widths := baseProjectColumnWidths()
	header := projectRow{
		ID:            "ID",
		Name:          "项目名称",
		Path:          "路径",
		LastRun:       "最后执行",
		SuccessRate:   "成功率",
		TotalCommands: "命令数",
		Exists:        "存在",
	}
	widths.update(header)

	for _, entry := range entries {
		exists := "✓"
		if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
			exists = "✗"
		}
		lastRun := "-"
		if entry.LastCommandTime.Valid {
			lastRun = entry.LastCommandTime.Time.Format("2006-01-02 15:04:05")
		}
		name := entry.Name
		if strings.TrimSpace(name) == "" {
			name = template.GetProjectName(entry.Path)
		}
		row := projectRow{
			ID:            strconv.Itoa(entry.ID),
			Name:          name,
			Path:          entry.Path,
			LastRun:       lastRun,
			SuccessRate:   fmt.Sprintf("%.1f%%", entry.GetSuccessRate()),
			TotalCommands: strconv.Itoa(entry.TotalCommands),
			Exists:        exists,
		}
		widths.update(row)
		rows = append(rows, row)
	}

	fmt.Printf("已注册的项目 (共%d个):\n\n", len(entries))
	printProjectTable(header, rows, widths)

	return nil
}

func printProjectTable(header projectRow, rows []projectRow, widths columnWidths) {
	fmt.Println(formatProjectRow(header, widths))
	separatorWidth := widths.total() + projectColumnSpacing
	fmt.Println(strings.Repeat("-", separatorWidth))
	for _, row := range rows {
		fmt.Println(formatProjectRow(row, widths))
	}
}

func formatProjectRow(row projectRow, widths columnWidths) string {
	cells := []string{
		padRight(row.ID, widths.ID),
		padRight(row.Name, widths.Name),
		padRight(row.Path, widths.Path),
		padRight(row.LastRun, widths.LastRun),
		padRight(row.SuccessRate, widths.SuccessRate),
		padRight(row.TotalCommands, widths.TotalCommands),
		padRight(row.Exists, widths.Exists),
	}
	return strings.Join(cells, " ")
}

func padRight(text string, width int) string {
	padding := width - displayWidth(text)
	if padding <= 0 {
		return text
	}
	return text + strings.Repeat(" ", padding)
}

func displayWidth(value string) int {
	width := 0
	for _, r := range value {
		width += runeDisplayWidth(r)
	}
	return width
}

func runeDisplayWidth(r rune) int {
	if r == 0 {
		return 0
	}
	if r < 0x1100 {
		return 1
	}
	switch {
	case r >= 0x1100 && r <= 0x115f,
		r == 0x2329 || r == 0x232a,
		r >= 0x2e80 && r <= 0xa4cf && r != 0x303f,
		r >= 0xac00 && r <= 0xd7a3,
		r >= 0xf900 && r <= 0xfaff,
		r >= 0xfe10 && r <= 0xfe19,
		r >= 0xfe30 && r <= 0xfe6f,
		r >= 0xff00 && r <= 0xff60,
		r >= 0xffe0 && r <= 0xffe6,
		r >= 0x20000 && r <= 0x2fffd,
		r >= 0x30000 && r <= 0x3fffd:
		return 2
	default:
		return 1
	}
}

func baseProjectColumnWidths() columnWidths {
	return columnWidths{
		ID:            5,
		Name:          20,
		Path:          45,
		LastRun:       19,
		SuccessRate:   8,
		TotalCommands: 10,
		Exists:        2,
	}
}

type projectRow struct {
	ID            string
	Name          string
	Path          string
	LastRun       string
	SuccessRate   string
	TotalCommands string
	Exists        string
}

type columnWidths struct {
	ID            int
	Name          int
	Path          int
	LastRun       int
	SuccessRate   int
	TotalCommands int
	Exists        int
}

func (w *columnWidths) update(row projectRow) {
	w.ID = maxInt(w.ID, displayWidth(row.ID))
	w.Name = maxInt(w.Name, displayWidth(row.Name))
	w.Path = maxInt(w.Path, displayWidth(row.Path))
	w.LastRun = maxInt(w.LastRun, displayWidth(row.LastRun))
	w.SuccessRate = maxInt(w.SuccessRate, displayWidth(row.SuccessRate))
	w.TotalCommands = maxInt(w.TotalCommands, displayWidth(row.TotalCommands))
	w.Exists = maxInt(w.Exists, displayWidth(row.Exists))
}

func (w columnWidths) total() int {
	return w.ID + w.Name + w.Path + w.LastRun + w.SuccessRate + w.TotalCommands + w.Exists
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

const (
	projectColumnCount   = 7
	projectColumnSpacing = projectColumnCount - 1
)

func cleanProjects() error {
	services, err := newCLIServices()
	if err != nil {
		return err
	}
	defer services.Close()
	reg := services.Registry()

	projects, err := reg.List()
	if err != nil {
		return fmt.Errorf("获取项目列表失败: %w", err)
	}

	totalProjects := len(projects)
	if totalProjects == 0 {
		fmt.Println("没有需要清理的项目。")
		return nil
	}

	var invalidProjects []*model.Project
	for _, project := range projects {
		if _, statErr := os.Stat(project.Path); os.IsNotExist(statErr) {
			invalidProjects = append(invalidProjects, project)
		}
	}

	validCount := totalProjects - len(invalidProjects)
	fmt.Printf("共检查 %d 个项目，%d 个目录正常，%d 个目录不存在。\n", totalProjects, validCount, len(invalidProjects))

	if len(invalidProjects) == 0 {
		if err := reg.CheckAndCleanup(); err != nil {
			return fmt.Errorf("更新检查时间失败: %w", err)
		}
		fmt.Println("未发现需要清理的项目。")
		return nil
	}

	fmt.Println("以下项目将被删除（日志目录不存在）：")
	for _, project := range invalidProjects {
		displayName := strings.TrimSpace(project.Name)
		if displayName == "" {
			displayName = template.GetProjectName(project.Path)
		}
		fmt.Printf("- ID=%d 名称=\"%s\" 路径=%s\n", project.ID, displayName, project.Path)
	}

	if !projectCleanForce {
		fmt.Print("确认删除以上项目? [yes/N]: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		normalized := strings.ToLower(strings.TrimSpace(input))
		if normalized != "yes" && normalized != "y" && normalized != "确认" {
			fmt.Println("清理操作已取消")
			return nil
		}
	}

	if err := reg.CheckAndCleanup(); err != nil {
		return fmt.Errorf("清理失败: %w", err)
	}

	fmt.Printf("已删除 %d 个无效项目，目录不存在。\n", len(invalidProjects))
	return nil
}

func runProjectClean() error {
	if projectCleanDays <= 0 && projectCleanKeep <= 0 {
		return fmt.Errorf("请指定清理条件: --days 或 --keep")
	}

	services, err := newCLIServices()
	if err != nil {
		return err
	}
	defer services.Close()

	histMgr := services.HistoryManager()

	if !projectCleanForce {
		fmt.Println("⚠️  警告: 此操作将永久删除日志文件和数据库记录。")
		if projectCleanDays > 0 {
			fmt.Printf("- 删除 %d 天前的记录\n", projectCleanDays)
		}
		if projectCleanKeep > 0 {
			fmt.Printf("- 仅保留最近 %d 条记录\n", projectCleanKeep)
		}
		fmt.Print("确认执行? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(input)) != "y" {
			fmt.Println("操作已取消")
			return nil
		}
	}

	if projectCleanDays > 0 {
		if err := histMgr.DeleteOldRecords(projectCleanDays); err != nil {
			return err
		}
	}

	if projectCleanKeep > 0 {
		if err := histMgr.DeleteExcessRecords(projectCleanKeep); err != nil {
			return err
		}
	}

	return nil
}

func deleteProject(target string) error {
	services, err := newCLIServices()
	if err != nil {
		return err
	}
	defer services.Close()
	reg := services.Registry()

	project, err := reg.Get(target)
	if err != nil {
		return fmt.Errorf("查询项目失败: %w", err)
	}

	if !projectDeleteForce {
		confirmed, confirmErr := confirmProjectDeletion(project)
		if confirmErr != nil {
			return fmt.Errorf("读取用户输入失败: %w", confirmErr)
		}
		if !confirmed {
			fmt.Println("已取消删除操作")
			return nil
		}
	}

	if err := reg.Delete(target); err != nil {
		return fmt.Errorf("删除项目失败: %w", err)
	}

	if err := os.RemoveAll(project.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除项目日志目录失败: %w", err)
	}

	fmt.Printf("成功删除项目: %s\n", target)
	return nil
}

func confirmProjectDeletion(project *model.Project) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	displayName := project.Name
	if strings.TrimSpace(displayName) == "" {
		displayName = template.GetProjectName(project.Path)
	}

	fmt.Printf("⚠️ 即将删除项目 ID=%d 名称=\"%s\"\n", project.ID, displayName)
	fmt.Printf("日志目录: %s\n", project.Path)
	fmt.Print("请输入 yes/确认 继续删除，其他输入取消: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	normalized := strings.ToLower(strings.TrimSpace(input))
	return normalized == "yes" || normalized == "确认" || normalized == "y", nil
}
