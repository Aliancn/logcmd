package template

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ConfigureInteractive 交互式配置日志命名模板
func ConfigureInteractive() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== 日志文件命名模板配置 ===")
	fmt.Println()

	// 加载现有模板或使用默认模板
	template, err := Load()
	if err != nil {
		fmt.Printf("加载配置失败，将使用默认模板: %v\n", err)
		template = DefaultTemplate()
	}

	// 显示当前模板
	fmt.Println("当前模板配置：")
	printTemplate(template)
	fmt.Println()

	// 询问是否要修改
	fmt.Print("是否要重新配置模板？(y/n): ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" && answer != "yes" {
		fmt.Println("配置已取消")
		return nil
	}

	// 创建新模板
	newTemplate := &LogNameTemplate{
		Elements:  []NameElement{},
		Separator: "_",
	}

	// 配置分隔符
	fmt.Println()
	fmt.Println("=== 配置分隔符 ===")
	fmt.Printf("当前分隔符: %s\n", template.Separator)
	fmt.Print("输入新的分隔符（直接回车保持当前值）: ")
	separator, _ := reader.ReadString('\n')
	separator = strings.TrimSpace(separator)
	if separator != "" {
		newTemplate.Separator = separator
	} else {
		newTemplate.Separator = template.Separator
	}

	// 配置命名元素
	fmt.Println()
	fmt.Println("=== 配置命名元素 ===")
	fmt.Println("可用的命名元素：")
	fmt.Println("1. command  - 命令名称")
	fmt.Println("2. time     - 时间戳")
	fmt.Println("3. project  - 项目名称（.logcmd父目录名）")
	fmt.Println("4. custom   - 自定义文本")
	fmt.Println()

	for {
		fmt.Println()
		fmt.Printf("当前已添加 %d 个元素\n", len(newTemplate.Elements))
		if len(newTemplate.Elements) > 0 {
			fmt.Println("当前元素顺序：")
			for i, elem := range newTemplate.Elements {
				fmt.Printf("  %d. %s", i+1, elem.Type)
				if elem.Type == ElementTypeCustom {
					fmt.Printf(" (文本: %s)", elem.Config["text"])
				}
				fmt.Println()
			}
		}

		fmt.Println()
		fmt.Println("请选择操作：")
		fmt.Println("1. 添加元素")
		fmt.Println("2. 删除元素")
		fmt.Println("3. 调整元素顺序")
		fmt.Println("4. 完成配置")
		fmt.Print("请输入选项 (1-4): ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			if err := addElement(reader, newTemplate); err != nil {
				fmt.Printf("添加元素失败: %v\n", err)
			}
		case "2":
			if err := removeElement(reader, newTemplate); err != nil {
				fmt.Printf("删除元素失败: %v\n", err)
			}
		case "3":
			if err := reorderElements(reader, newTemplate); err != nil {
				fmt.Printf("调整顺序失败: %v\n", err)
			}
		case "4":
			if len(newTemplate.Elements) == 0 {
				fmt.Println("错误: 至少需要添加一个元素")
				continue
			}
			goto done
		default:
			fmt.Println("无效的选项，请重新选择")
		}
	}

done:
	// 显示最终配置
	fmt.Println()
	fmt.Println("=== 最终配置 ===")
	printTemplate(newTemplate)
	fmt.Println()

	// 预览示例
	fmt.Println("=== 文件名预览 ===")
	exampleName := newTemplate.GenerateLogName("npm", []string{"test"}, "myproject", nil, "20060102_150405")
	fmt.Printf("示例文件名: %s\n", exampleName)
	fmt.Println()

	// 确认保存
	fmt.Print("确认保存配置？(y/n): ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "y" && confirm != "yes" {
		fmt.Println("配置已取消")
		return nil
	}

	// 保存配置
	if err := newTemplate.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	configPath, _ := GetConfigPath()
	fmt.Printf("配置已保存到: %s\n", configPath)
	return nil
}

// addElement 添加命名元素
func addElement(reader *bufio.Reader, template *LogNameTemplate) error {
	fmt.Println()
	fmt.Println("选择要添加的元素类型：")
	fmt.Println("1. command  - 命令名称")
	fmt.Println("2. time     - 时间戳")
	fmt.Println("3. project  - 项目名称")
	fmt.Println("4. custom   - 自定义文本")
	fmt.Print("请输入选项 (1-4): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var element NameElement
	element.Config = make(map[string]string)

	switch choice {
	case "1":
		element.Type = ElementTypeCommand
		fmt.Println("已添加：命令名称")
	case "2":
		element.Type = ElementTypeTime
		fmt.Println("已添加：时间戳（格式由全局/局部配置控制）")
	case "3":
		element.Type = ElementTypeProject
		fmt.Println("已添加：项目名称")
	case "4":
		element.Type = ElementTypeCustom
		fmt.Print("输入自定义文本: ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "" {
			return fmt.Errorf("自定义文本不能为空")
		}
		element.Config["text"] = text
		fmt.Printf("已添加：自定义文本（%s）\n", text)
	default:
		return fmt.Errorf("无效的选项")
	}

	template.Elements = append(template.Elements, element)
	return nil
}

// removeElement 删除命名元素
func removeElement(reader *bufio.Reader, template *LogNameTemplate) error {
	if len(template.Elements) == 0 {
		fmt.Println("没有可删除的元素")
		return nil
	}

	fmt.Println()
	fmt.Println("当前元素列表：")
	for i, elem := range template.Elements {
		fmt.Printf("%d. %s\n", i+1, elem.Type)
	}

	fmt.Print("请输入要删除的元素编号: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	index, err := strconv.Atoi(input)
	if err != nil || index < 1 || index > len(template.Elements) {
		return fmt.Errorf("无效的编号")
	}

	// 删除元素
	template.Elements = append(template.Elements[:index-1], template.Elements[index:]...)
	fmt.Println("元素已删除")
	return nil
}

// reorderElements 调整元素顺序
func reorderElements(reader *bufio.Reader, template *LogNameTemplate) error {
	if len(template.Elements) < 2 {
		fmt.Println("元素数量少于2个，无需调整顺序")
		return nil
	}

	fmt.Println()
	fmt.Println("当前元素顺序：")
	for i, elem := range template.Elements {
		fmt.Printf("%d. %s\n", i+1, elem.Type)
	}

	fmt.Print("输入要移动的元素编号: ")
	fromInput, _ := reader.ReadString('\n')
	fromInput = strings.TrimSpace(fromInput)

	fromIndex, err := strconv.Atoi(fromInput)
	if err != nil || fromIndex < 1 || fromIndex > len(template.Elements) {
		return fmt.Errorf("无效的编号")
	}

	fmt.Print("输入目标位置: ")
	toInput, _ := reader.ReadString('\n')
	toInput = strings.TrimSpace(toInput)

	toIndex, err := strconv.Atoi(toInput)
	if err != nil || toIndex < 1 || toIndex > len(template.Elements) {
		return fmt.Errorf("无效的位置")
	}

	// 转换为0-based索引
	fromIndex--
	toIndex--

	// 移动元素
	element := template.Elements[fromIndex]
	template.Elements = append(template.Elements[:fromIndex], template.Elements[fromIndex+1:]...)
	template.Elements = append(template.Elements[:toIndex], append([]NameElement{element}, template.Elements[toIndex:]...)...)

	fmt.Println("顺序已调整")
	return nil
}

// printTemplate 打印模板配置
func printTemplate(template *LogNameTemplate) {
	fmt.Printf("分隔符: %s\n", template.Separator)
	fmt.Println("命名元素：")
	if len(template.Elements) == 0 {
		fmt.Println("  (无)")
	} else {
		for i, elem := range template.Elements {
			fmt.Printf("  %d. %s", i+1, elem.Type)
			if elem.Type == ElementTypeCustom {
				fmt.Printf(" (文本: %s)", elem.Config["text"])
			}
			fmt.Println()
		}
	}
}
