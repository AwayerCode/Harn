# YAML 模块架构

YAML 是采用本规范的项目在层、模块、目录、必要文件和允许源码依赖方面的权威来源。文件结构落实架构，HTML 是由同一 YAML 生成的只读视图。项目约定声明设计路径和受管范围；原 Markdown 中对应的架构表格不再手工重复维护，业务依据、设计理由和接口契约保持各自权威来源。

## 数据与文件结构

格式由 [JSON Schema](schema.json) 定义，规范版本为 `1`。只接受一个 YAML 文档，禁止重复键、别名、合并键、自定义标签和未知字段；最大 1 MiB。层与模块标识使用小写字母、数字、连字符，并以字母开头。

| 字段 | 约束 |
| --- | --- |
| `source_root` | 项目根目录下的受管源码目录，使用规范化相对路径，不能是项目根本身 |
| `layers` | 层的 `id`、`directory`、`responsibility`、`allowed_dependencies` |
| `modules` | 模块的上述四项字段，以及所属 `layer`、`required_files` |
| `directory` | 单级目录名，使用字母、数字、下划线或连字符 |
| `required_files` | 相对模块目录的文件路径，可声明嵌套文件；允许空数组 |
| `allowed_dependencies` | 层引用层 ID，模块引用模块 ID，必须显式填写，包括空数组 |

模块目录固定为 `source_root / 层目录 / 模块目录`。同级目录按大小写不敏感方式检查冲突，受管目录中的所有普通文件必须唯一归属于模块，模块内部允许继续划分子目录。目录和文件不得通过符号链接跨越边界。未声明的层、模块目录及游离文件均违规；模块内新增文件允许，不要求列出全部文件。必要文件列表按大小写不敏感方式拒绝重名及祖先路径冲突，不能把同一路径同时声明为文件和目录。

测试和生成代码放在对应模块内，参与同样的归属与导入检查，不按 Git 忽略规则排除。启动组装代码作为明确模块放在指定层目录内。首版不提供排除规则或路径例外；不符合要求的项目先调整受管范围或文件结构。

## 依赖与验收语义

依赖仅指源码引用。未声明的跨模块依赖禁止；跨层引用必须同时满足层与模块允许关系。同层模块引用也须显式允许。同模块内部引用允许。允许依赖无需实际使用，声明的层、模块依赖图与观察到的模块依赖图均不得有环。

设计阶段校验字段、引用、路径和依赖规则，不要求源码存在。实现验收才要求层和模块目录、必要文件存在，并检查源码归属及实际导入。检查只覆盖 `source_root`，受管模块引用当前 Go module 中该范围以外的代码会报错；范围外代码自身不被扫描。

首版支持根目录含 `go.mod` 的单个 Go module。扫描所有 `.go` 文件的导入声明，包括测试、生成文件和所有构建标签分支；采用保守的依赖并集，不代表特定平台的编译结果。外部库导入不映射为内部模块。`go.mod` 使用 Go 官方解析库读取，合法注释与块语法不会改变依赖识别。`GOWORK` 未设置或为 `auto` 时检查祖先目录；设为 `off` 时禁用 workspace，显式路径（包括不存在的路径）返回无法检查。遇到启用的 `go.work`、`replace`、源码路径中的嵌套 Go module、受管目录内的 vendor、符号链接或读取解析失败时明确返回无法检查。接口、函数调用、类型解析和业务行为不在本检查范围，项目测试与构建仍需单独执行。

## 执行闭环

1. 架构设计维护源码根、层、职责及层间允许依赖，模块设计维护模块、目录、必要文件及模块依赖。局部交接可以使用 `modules: []`；只有 `validate --profile architecture` 允许无模块设计，局部通过不能作为实现验收。
2. 运行完整 `validate`；通过后生成只读 HTML。视图展示层、模块、目录、必要文件和可点击的依赖链接，标记设计摘要及生成器版本。
3. Agent 按 YAML 创建目录和文件，实现模块，在指定可写范围内执行项目测试与构建。
4. 运行 `check`，处理结构与源码依赖偏差，再执行范围检查。设计调整由对应设计责任方完成，不通过放宽规则掩盖实现问题。

```bash
swe-cli architecture validate --root /absolute/project --file architecture.yaml --profile architecture
swe-cli architecture validate --root /absolute/project --file architecture.yaml
swe-cli architecture check --root /absolute/project --file architecture.yaml
```

`validate` 与 `check` 输出 JSON。退出码 0 表示本次范围通过，1 表示违规，2 表示无法检查；结果包含设计 SHA256、生成器版本、覆盖范围、未验证项及实际跨模块依赖的文件和行号。检查读取当前文件，不提供原子快照，应在静止工作区运行。

`render` 通过设计校验后将 HTML 写到标准输出，失败输出 JSON 诊断且返回非零。使用临时文件避免失败时覆盖原图；仅在退出码为 0 时替换最终视图：

```bash
swe-cli architecture render --root /absolute/project --file architecture.yaml > architecture.pending.html && mv architecture.pending.html architecture.html
```

重新查看旧图前比较其设计 SHA256 与当前校验结果。生成视图属于明确写入动作，须纳入任务可写范围；检查命令不写项目。该入口独立于现有 JSON `check` 请求，不修改协议版本 2。

## 使用示例

以下声明对应 `src/application/orders/service.go` 与 `src/application/payments/service.go`，订单模块可以引用支付模块，反向引用禁止。

```yaml
schema_version: 1
source_root: src
layers:
  - id: application
    directory: application
    responsibility: 应用流程与业务协调
    allowed_dependencies: []
modules:
  - id: orders
    layer: application
    directory: orders
    responsibility: 订单处理
    required_files: [service.go]
    allowed_dependencies: [payments]
  - id: payments
    layer: application
    directory: payments
    responsibility: 支付处理
    required_files: [service.go]
    allowed_dependencies: []
```

仓库中的可运行三模块 Go 示例位于 `cli/swe-cli/internal/architecture/testdata/project/`，包含层间与同层模块依赖。开发阶段可从 `cli/swe-cli` 执行：

```bash
go run . architecture validate --root internal/architecture/testdata/project --file architecture.yaml
go run . architecture check --root internal/architecture/testdata/project --file architecture.yaml
go run . architecture render --root internal/architecture/testdata/project --file architecture.yaml > /tmp/harn-architecture.html
```

## 版本与配套工具

该能力需要 SWE CLI `0.4.0-alpha.1` 或包含本实现的开发构建。配套清单绑定本地六平台构建的确切摘要，发行资产尚未发布；不能用旧检查器验证新设计，也不能把本地构建通过当作跨平台运行验收。
