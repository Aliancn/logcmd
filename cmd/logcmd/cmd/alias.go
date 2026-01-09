package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var aliasPrefix string

var aliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "生成 Shell 别名脚本",
	Long: `生成 Shell 别名脚本，方便快速使用 logcmd。
使用方法:
  source <(logcmd alias)
  或将输出添加到 .bashrc / .zshrc`,
	Run: func(cmd *cobra.Command, args []string) {
		printAliases()
	},
}

func init() {
	aliasCmd.Flags().StringVar(&aliasPrefix, "prefix", "l", "别名前缀 (例如: l -> lrun, lsearch)")
	rootCmd.AddCommand(aliasCmd)
}

func printAliases() {
	p := aliasPrefix
	fmt.Printf("# LogCmd Aliases\n")
	fmt.Printf("alias %srun='logcmd run'\n", p)
	fmt.Printf("alias %ssearch='logcmd search'\n", p)
	fmt.Printf("alias %sstats='logcmd stats'\n", p)
	fmt.Printf("alias %sui='logcmd ui'\n", p)
	fmt.Printf("alias %stask='logcmd task'\n", p)
	fmt.Printf("alias %sproject='logcmd project'\n", p)
}
