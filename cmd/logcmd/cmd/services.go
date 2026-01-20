package cmd

import "github.com/aliancn/logcmd/internal/app"

// 为了保持 CLI 包的最小变更面，这里提供一个到新的依赖容器的别名。
// 后续命令调用 newCLIServices 即可获得 app.Container。
type cliServices = app.Container

func newCLIServices() (*cliServices, error) {
	return app.NewContainer()
}
