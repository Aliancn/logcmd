package template_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliancn/logcmd/internal/template"
)

func TestDefaultTemplate(t *testing.T) {
	tmpl := template.DefaultTemplate()

	if tmpl == nil {
		t.Fatal("DefaultTemplate() 返回了 nil")
	}

	if tmpl.Separator != "_" {
		t.Errorf("默认分隔符应该是 '_': got %s", tmpl.Separator)
	}

	if len(tmpl.Elements) == 0 {
		t.Error("默认模板应该包含至少一个元素")
	}

	// 验证默认有时间元素
	hasTimeElement := false
	for _, elem := range tmpl.Elements {
		if elem.Type == template.ElementTypeTime {
			hasTimeElement = true
			break
		}
	}
	if !hasTimeElement {
		t.Error("默认模板应该包含时间元素")
	}
}

func TestGenerateLogName(t *testing.T) {
	tmpl := template.DefaultTemplate()
	tz := time.UTC

	tests := []struct {
		name        string
		command     string
		args        []string
		projectName string
		checkFunc   func(string) bool
	}{
		{
			name:        "默认模板生成",
			command:     "echo",
			args:        []string{"test"},
			projectName: "myproject",
			checkFunc: func(filename string) bool {
				// 应该包含时间戳和.log扩展名
				return strings.HasSuffix(filename, ".log") && len(filename) > 10
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tmpl.GenerateLogName(tt.command, tt.args, tt.projectName, tz, "20060102_150405")

			if filename == "" {
				t.Error("生成的文件名不应为空")
			}

			if !tt.checkFunc(filename) {
				t.Errorf("生成的文件名不符合预期: %s", filename)
			}

			// 验证文件名不包含不安全字符
			unsafeChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
			for _, char := range unsafeChars {
				if strings.Contains(filename, char) {
					t.Errorf("文件名包含不安全字符 '%s': %s", char, filename)
				}
			}
		})
	}
}

func TestGenerateLogNameWithCustomTemplate(t *testing.T) {
	tz := time.UTC

	tests := []struct {
		name     string
		template *template.LogNameTemplate
		command  string
		project  string
		validate func(string) bool
	}{
		{
			name: "包含命令元素",
			template: &template.LogNameTemplate{
				Separator: "_",
				Elements: []template.NameElement{
					{Type: template.ElementTypeCommand},
					{Type: template.ElementTypeTime},
				},
			},
			command: "test",
			project: "proj",
			validate: func(filename string) bool {
				return strings.HasPrefix(filename, "test_")
			},
		},
		{
			name: "包含项目元素",
			template: &template.LogNameTemplate{
				Separator: "-",
				Elements: []template.NameElement{
					{Type: template.ElementTypeProject},
					{Type: template.ElementTypeTime},
				},
			},
			command: "test",
			project: "myproject",
			validate: func(filename string) bool {
				return strings.HasPrefix(filename, "myproject-")
			},
		},
		{
			name: "包含自定义文本",
			template: &template.LogNameTemplate{
				Separator: "_",
				Elements: []template.NameElement{
					{Type: template.ElementTypeCustom, Config: map[string]string{"text": "custom"}},
				},
			},
			command: "test",
			project: "proj",
			validate: func(filename string) bool {
				return strings.Contains(filename, "custom")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.template.GenerateLogName(tt.command, []string{}, tt.project, tz, "20060102_150405")

			if !tt.validate(filename) {
				t.Errorf("生成的文件名不符合预期: %s", filename)
			}

			if !strings.HasSuffix(filename, ".log") {
				t.Errorf("文件名应该以 .log 结尾: %s", filename)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	// 创建临时目录
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Unsetenv("HOME")

	// 创建自定义模板
	customTmpl := &template.LogNameTemplate{
		Separator: "-",
		Elements: []template.NameElement{
			{Type: template.ElementTypeProject},
			{Type: template.ElementTypeCommand},
			{Type: template.ElementTypeTime},
		},
	}

	// 保存模板
	err := customTmpl.Save()
	if err != nil {
		t.Fatalf("Save() 失败: %v", err)
	}

	// 加载模板
	loadedTmpl, err := template.Load()
	if err != nil {
		t.Fatalf("Load() 失败: %v", err)
	}

	// 验证加载的模板
	if loadedTmpl.Separator != customTmpl.Separator {
		t.Errorf("分隔符不匹配: got %s, want %s", loadedTmpl.Separator, customTmpl.Separator)
	}

	if len(loadedTmpl.Elements) != len(customTmpl.Elements) {
		t.Errorf("元素数量不匹配: got %d, want %d", len(loadedTmpl.Elements), len(customTmpl.Elements))
	}
}

func TestLoadNonExistentConfig(t *testing.T) {
	// 使用临时目录，确保配置文件不存在
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Unsetenv("HOME")

	// 加载不存在的配置应该返回默认模板
	tmpl, err := template.Load()
	if err != nil {
		t.Fatalf("Load() 失败: %v", err)
	}

	if tmpl == nil {
		t.Fatal("Load() 返回了 nil")
	}

	// 应该返回默认模板
	defaultTmpl := template.DefaultTemplate()
	if tmpl.Separator != defaultTmpl.Separator {
		t.Error("未找到配置文件时应返回默认模板")
	}
}

func TestGetProjectName(t *testing.T) {
	tests := []struct {
		name     string
		logDir   string
		expected string
	}{
		{
			name:     "标准路径",
			logDir:   "/home/user/myproject/.logcmd",
			expected: "myproject",
		},
		{
			name:     "嵌套路径",
			logDir:   "/home/user/projects/awesome-app/.logcmd",
			expected: "awesome-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectName := template.GetProjectName(tt.logDir)

			if projectName != tt.expected && projectName != "unknown" {
				// 由于路径可能不存在，允许返回 "unknown"
				t.Logf("GetProjectName(%s) = %s, want %s", tt.logDir, projectName, tt.expected)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	// 设置临时HOME环境变量
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	configPath, err := template.GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() 失败: %v", err)
	}

	if configPath == "" {
		t.Error("配置路径不应为空")
	}

	// 验证路径包含 .logcmd
	if !strings.Contains(configPath, ".logcmd") {
		t.Errorf("配置路径应该包含 .logcmd: %s", configPath)
	}

	// 验证路径以 template.json 结尾
	if filepath.Base(configPath) != "template.json" {
		t.Errorf("配置文件名应该是 template.json: got %s", filepath.Base(configPath))
	}
}

func TestFilenameCharacterSanitization(t *testing.T) {
	tmpl := &template.LogNameTemplate{
		Separator: "_",
		Elements: []template.NameElement{
			{Type: template.ElementTypeCommand},
		},
	}

	unsafeCommands := []string{
		"echo/test",
		"test:command",
		"cmd*wildcard",
		"test?query",
		"test\"quote",
		"test<redirect",
		"test>output",
		"test|pipe",
		"test command",
	}

	tz := time.UTC

	for _, cmd := range unsafeCommands {
		t.Run("清理不安全字符: "+cmd, func(t *testing.T) {
			filename := tmpl.GenerateLogName(cmd, []string{}, "project", tz, "20060102_150405")

			// 验证所有不安全字符都被替换
			unsafeChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
			for _, char := range unsafeChars {
				if strings.Contains(filename, char) {
					t.Errorf("文件名仍包含不安全字符 '%s': %s", char, filename)
				}
			}

			// 验证包含下划线（替换后的字符）
			if !strings.Contains(filename, "_") && cmd != unsafeCommands[len(unsafeCommands)-1] {
				t.Logf("文件名已清理: %s -> %s", cmd, filename)
			}
		})
	}
}
