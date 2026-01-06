# LogCmd 数据库能力概览

LogCmd 依赖 SQLite 维护命令执行的全量元数据，本文件概述数据库层需要提供的核心能力与模块职责。

## 能力矩阵
- **项目管理**：`registry` 模块负责项目注册、统计增量更新、路径有效性校验以及最后检查时间维护。
- **命令历史**：`history.Manager` 将每次执行的命令要素（命令、参数、时间、退出码、日志路径、预览、运行环境）写入 `command_history`，并提供多维筛选接口。
- **统计缓存**：`stats.CacheManager` 以项目和日期为粒度缓存统计结果，包含命令分布、退出码分布、时长统计与趋势汇总。
- **迁移体系**：`migration` 模块负责版本检测、表结构升级、数据拷贝与索引创建，确保多版本平滑演进。
- **持久化协调**：`internal/persistence` 提供 `RunRepository`/`StatsUpdater`，复用 Registry 连接并统一封装项目注册、历史写入、统计更新。

## 架构关系
```
executor → logger → persistence.StatsUpdater → registry.UpdateStats
                  ↘ persistence.RunRepository.RegisterProject
                    ↘ history.Manager.Record → stats.CacheManager.Generate*
```
- Executor/Logger 只负责日志输出与元数据采集。
- Registry 维护项目生命周期、统计字段与数据库连接。
- History/Stats 共享同一 DB 连接，独立负责记录与聚合。

## 数据生命周期
1. Logger 通过 `RunRepository` 注册项目、写入命令元信息，并借助 `StatsUpdater` 触发 Registry 统计增量更新。
2. History 管理器在命令完成后记录结构化历史数据。
3. Stats 缓存管理器根据历史数据生成或刷新指定日期的缓存；CLI 查询时优先使用缓存，无缓存时触发即时生成。
4. 高层命令（`search`, `stats`, `project`）只依赖 Registry 暴露的 API，不直接访问底层表。

## 维护要点
- 所有路径与时间以绝对路径 / UTC 存储，展示层负责本地化。
- JSON 字段承担扩展角色：标签、模板配置、命令/退出码分布等都通过 JSON 存储以减少结构变更。
- 自动迁移必须在 Registry 初始化阶段执行，以保证所有 CLI 功能在统一的模式上运行。

本概览描述了数据库层的职责边界与协作方式，具体代码说明请参照各模块源文件注释。

## 持久化协调层

`internal/persistence` 让 Logger 只依赖两个接口：

- `RunRepository`：封装项目注册、命令历史写入与统计缓存刷新；内部连接 `registry.Registry`、`history.Manager`、`stats.CacheManager` 并共用同一个 DB。
- `StatsUpdater`：实现 Logger 需要的 `ProjectStatsUpdater` 接口，直接调用 `registry.UpdateStats`，确保 CLI 和后台任务通过一致的管道更新项目指标。

这种设计遵循单一职责与依赖倒置：

1. CLI/Logger 仅关心“项目、命令、结果”三个抽象；
2. Registry 暴露持久层能力，但不会渗透到业务调用层；
3. History/Stats 只关注自身的写入与聚合，不需要知道命令来自 CLI 还是后台任务。
