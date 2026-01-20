## Internal 层级说明

```
internal/
├── application   # 应用层，负责依赖容器、执行器、领域服务编排、CLI 辅助工具
├── domain        # 领域层，包含 Config/Model 等核心业务对象与逻辑
├── platform      # 平台层，处理数据库、任务、日志、统计等基础设施实现
└── presentation  # 展示层，目前为 TUI 组件与交互逻辑
```

- **application**：定义容器与服务，负责协调领域模型与平台实现。
- **domain**：保持纯粹的业务实体与规则，避免直接依赖外部设施。
- **platform**：封装数据库、执行器、日志管理、统计缓存等技术细节。
- **presentation**：面向用户界面的 UI 层，包括模式、组件和渲染服务。
