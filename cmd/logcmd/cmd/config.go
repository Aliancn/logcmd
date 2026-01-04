package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/aliancn/logcmd/internal/config"
	"github.com/aliancn/logcmd/internal/template"
	"github.com/spf13/cobra"
)

var (
	globalFlag bool
	localFlag  bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理 LogCmd 配置",
	Long:  "管理 LogCmd 的配置选项。支持全局配置 (~/.logcmd/config.json) 和局部项目配置 (.logcmd/config.json)。",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> [value]",
	Short: "设置配置项",
	Example: `  logcmd config set buffer_size 10240
  logcmd config set auto_compress true --global
  logcmd config set time_format compact`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "获取配置项",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有配置",
	RunE:  runConfigList,
}

var configLogNameCmd = &cobra.Command{
	Use:   "logname",
	Short: "交互式配置日志命名模板",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := template.ConfigureInteractive(); err != nil {
			return fmt.Errorf("配置模板失败: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configLogNameCmd)

	configSetCmd.Flags().BoolVar(&globalFlag, "global", false, "使用全局配置")
	configSetCmd.Flags().BoolVar(&localFlag, "local", false, "使用局部配置 (默认)")
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	var val string

	// 处理 time_format 的特殊逻辑（支持交互式或别名）
	if key == "time_format" {
		if len(args) == 2 {
			// 用户提供了值，检查是否为有效别名
			input := args[1]
			if config.IsValidTimeFormat(input) {
				val = config.GetTimeFormat(input)
			} else {
				// 如果不是别名，显示错误和可用选项
				fmt.Printf("错误: '%s' 不是有效的时间格式别名。\n\n可用格式:\n", input)
				for _, desc := range config.GetTimeFormatDescriptions() {
					fmt.Printf("  %s\n", desc)
				}
				return fmt.Errorf("无效的时间格式")
			}
		} else {
			// 用户未提供值，进入交互式选择
			fmt.Println("请选择时间格式:")
			descriptions := config.GetTimeFormatDescriptions()
			// 简单的映射：index -> key
			keys := []string{"compact", "standard", "simple", "dateonly"}

			for i, desc := range descriptions {
				fmt.Printf("%d. %s\n", i+1, desc)
			}

			reader := bufio.NewReader(os.Stdin)
			fmt.Print("\n请输入选项 (1-4): ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			index, err := strconv.Atoi(input)
			if err != nil || index < 1 || index > len(keys) {
				return fmt.Errorf("无效的选项")
			}
			val = config.GetTimeFormat(keys[index-1])
		}
	} else {
		// 对于其他配置项，必须提供值
		if len(args) < 2 {
			return fmt.Errorf("必须提供值: logcmd config set %s <value>", key)
		}
		val = args[1]
	}

	if globalFlag && localFlag {
		return fmt.Errorf("不能同时指定 --global 和 --local")
	}

	var path string
	var err error

	if globalFlag {
		path, err = config.GetGlobalConfigPath()
	} else {
		// 默认为局部配置
		cwd, _ := os.Getwd()
		path, err = config.GetLocalConfigPath(cwd)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("未找到局部项目配置 (.logcmd目录不存在)。\n请先执行 'logcmd run' 初始化项目，或者使用 --global 设置全局配置。")
			}
			return err
		}
	}

	if err != nil {
		return err
	}

	// 加载现有文件（如果存在）
	cfg, err := config.LoadConfigFile(path)
	if err != nil {
		// 如果加载出错，可能是文件格式错，也可能是文件不存在
		// 如果是新文件，我们从空开始
		cfg = &config.PersistentConfig{}
	}
	if cfg == nil {
		cfg = &config.PersistentConfig{}
	}

	// 更新值
	switch key {
	case "buffer_size":
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("buffer_size 必须是整数: %w", err)
		}
		cfg.BufferSize = v
	case "auto_compress":
		v, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("auto_compress 必须是 boolean (true/false): %w", err)
		}
		cfg.AutoCompress = boolPtr(v)
	case "time_format":
		cfg.TimeFormat = val
	default:
		return fmt.Errorf("未知配置项: %s", key)
	}

	// 保存
	if err := config.SaveConfigFile(path, *cfg); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	scope := "局部"
	if globalFlag {
		scope = "全局"
	}

	// 为了显示友好，如果是 time_format，我们也许想显示别名？
	// 但实际上存的是格式串。这里直接显示值即可。
	fmt.Printf("已更新%s配置: %s = %v\n", scope, key, val)
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	// 加载最终合并后的配置
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	switch key {
	case "buffer_size":
		fmt.Println(cfg.BufferSize)
	case "auto_compress":
		fmt.Println(cfg.AutoCompress)
	case "time_format":
		fmt.Println(cfg.TimeFormat)
	default:
		return fmt.Errorf("未知配置项: %s", key)
	}

	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tVALUE")
	fmt.Fprintf(w, "buffer_size\t%d\n", cfg.BufferSize)
	fmt.Fprintf(w, "auto_compress\t%v\n", cfg.AutoCompress)
	fmt.Fprintf(w, "time_format\t%s\n", cfg.TimeFormat)
	w.Flush()

	return nil
}

func boolPtr(v bool) *bool {
	return &v
}
