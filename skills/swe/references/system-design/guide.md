# YAML 模块架构

YAML 是采用本规范的项目在层、模块、目录、必要文件和允许源码依赖方面的权威来源。文件结构落实架构，HTML 是由同一 YAML 生成的只读视图。项目约定声明设计路径和受管范围；原 Markdown 中对应的架构表格不再手工重复维护，业务依据、设计理由和接口契约保持各自权威来源。

## 发现与目录作用域

文件名统一为 `system-design.yaml`。CLI 以 YAML 所在目录为作用域根，`source_root`、层、模块及必要文件均相对此处解析，不能用 `..` 或符号链接越界。`--root` 仅用于发现起点或解析相对 `--file`，不会扩大 YAML 管理范围。Git 根目录仅用于变更范围检查和 Go 模块定位。

`list --root` 只发现作用域，递归查找并按路径排序，不解析或校验 YAML 内容；成功不表示设计合法。仅排除 `.git` 元数据，显式 `--file` 也拒绝其中的设计；隐藏目录、`node_modules`、`vendor` 内的设计参与发现和作用域接管。递归遍历不跟随遇到的目录符号链接；显式 `--root` 及 `--file` 的父目录会先解析到真实目录，报告使用真实路径。YAML 文件本身仍须是普通文件，不能是符号链接；源码路径仍禁止通过符号链接越界。不使用 Git ignore 过滤目录。`--file` 指定单份设计；`validate/check --all --root` 逐份检查，任一设计失败仍继续其他设计。`render` 只支持单份。没有发现设计返回退出码 2，不能声称检查通过。

每份设计接管自己所在目录及子目录，遇到下级 `system-design.yaml` 则交由子设计负责，父级报告列出 `excluded_scopes`。父级不得声明位于子作用域内的源码根、层、模块或必要文件；父模块下面可以包含独立子作用域，其子树不计入父模块。子 YAML 内容错误也不会使父级重新接管，批量检查会报告子设计错误。单份检查不会验证被排除子设计的内容。

发现阶段的读取错误记录为 `discovery_issues`，继续检查其他分支。错误标记到最近的已发现所属作用域，使其返回无法检查；父级已交给子设计的目录不会因此重新接管。没有已发现所属作用域的错误也会使批量结果返回 2，但不阻止正常设计产生结果。

单份与批量使用相同发现和检查逻辑，包括无效设计文件的失败报告。`validate` 除 Schema 与关系校验外检查已有子作用域边界，不要求源码目录已存在。新增或移动子 YAML 后应重新校验父设计。

```bash
swe-cli system-design list --root /absolute/repo
swe-cli system-design validate --root /absolute/repo --all
swe-cli system-design check --root /absolute/repo --all
swe-cli system-design check --file /absolute/repo/apps/api/system-design.yaml
```

单份结果格式为 `swe-system-design-v1`，含 `scope.file`、`scope.root`、`scope.excluded_scopes`；列表和批量结果为 `swe-system-design-batch-v1`，其中 `scopes` 列出作用域，`list` 的 `results` 固定为空数组；`validate/check --all` 的 `results` 与 `scopes` 数量、顺序及作用域身份一一对应，失败报告也占对应位置。批量退出码优先级为无法检查 2、违规 1、全部通过 0。无法发现或命令参数错误使用单份错误报告。批量结果不能只读取最后一份状态。

## 数据与文件结构

格式由 [JSON Schema](schema.json) 定义，规范版本为 `1`。只接受一个 YAML 文档，禁止重复键、别名、合并键、自定义标签和未知字段；最大 1 MiB。层与模块标识使用小写字母、数字、连字符，并以字母开头。

| 字段 | 约束 |
| --- | --- |
| `source_root` | YAML 所在目录下的受管源码目录，使用规范化相对路径，不能是作用域根本身 |
| `layers` | 层的 `id`、`directory`、`responsibility`、`allowed_dependencies` |
| `modules` | 模块的上述四项字段，以及所属 `layer`、`required_files` |
| `directory` | 单级目录名，使用字母、数字、下划线或连字符 |
| `required_files` | 相对模块目录的文件路径，可声明嵌套文件；允许空数组 |
| `allowed_dependencies` | 层引用层 ID，模块引用模块 ID，必须显式填写，包括空数组 |

模块目录固定为 `source_root / 层目录 / 模块目录`。同级目录按大小写不敏感方式检查冲突，受管目录中的所有普通文件必须唯一归属于模块，模块内部允许继续划分子目录。目录和文件不得通过符号链接跨越边界。未声明的层、模块目录及游离文件均违规；模块内新增文件允许，不要求列出全部文件。必要文件列表按大小写不敏感方式拒绝重名及祖先路径冲突，不能把同一路径同时声明为文件和目录。

测试和生成代码放在对应模块内，参与同样的归属与导入检查，不按 Git 忽略规则排除。启动组装代码作为明确模块放在指定层目录内。除子设计作用域自动接管外，不提供手工排除规则或路径例外；不符合要求的项目先调整受管范围或文件结构。

## 依赖与验收语义

依赖仅指源码引用。未声明的跨模块依赖禁止；跨层引用必须同时满足层与模块允许关系。同层模块引用也须显式允许。同模块内部引用允许。允许依赖无需实际使用，声明的层、模块依赖图与观察到的模块依赖图均不得有环。

设计阶段校验字段、引用、路径和依赖规则，不要求源码存在。实现验收才要求层和模块目录、必要文件存在，并检查源码归属及实际导入。检查只覆盖 `source_root`，受管模块引用当前 Go module 中该范围以外的代码会报错；范围外代码自身不被扫描。

每份设计使用当前目录或最近祖先目录的 `go.mod`，查找不跨 Git 仓库边界。同一仓库可有多个独立 Go module，也可在同一 Go module 内划分多个设计作用域。扫描所有 `.go` 文件的导入声明，包括测试、生成文件和所有构建标签分支；采用保守的依赖并集，不代表特定平台的编译结果。跨作用域导入目前禁止：同 Go module 的目标落在当前目录外或子作用域内均报错；同一 Git 仓库中其他设计对应的独立 Go module 也按本地跨作用域依赖报错。模块查找区分不存在与无法安全读取：符号链接、权限及缺失 module 声明等错误保留为不确定元数据；此时不能确定是否属于本地模块的导入返回无法检查，不按外部库放行。已确认的当前模块导入和标准库导入仍可检查，不因无关元数据错误全部中止。仅真正外部库导入不映射为内部模块；没有 Git 边界时只识别最近 Go module 目录下的其他设计，不承诺发现仓库外的本地依赖。`go.mod` 使用 Go 官方解析库读取，合法注释与块语法不会改变依赖识别。`GOWORK` 未设置或为 `auto` 时检查祖先目录；设为 `off` 时禁用 workspace，显式路径（包括不存在的路径）返回无法检查。遇到启用的 `go.work`、`replace`、未由子设计接管的源码路径中的嵌套 Go module、受管目录内的 vendor、符号链接或读取解析失败时明确返回无法检查。接口、函数调用、类型解析和业务行为不在本检查范围，项目测试与构建仍需单独执行。

## 执行闭环

1. 系统设计维护源码根、层、职责及层间允许依赖，模块设计维护模块、目录、必要文件及模块依赖。局部交接可以使用 `modules: []`；只有 `validate --profile layers` 允许无模块设计，局部通过不能作为实现验收。
2. 运行完整 `validate`；通过后生成只读 HTML。视图用分层容器、模块节点和有向连线展示允许依赖，支持悬停或键盘聚焦追踪直接上下游；目录与必要文件置于详情，标记设计摘要及生成器版本。
3. Agent 按 YAML 创建目录和文件，实现模块，在指定可写范围内执行项目测试与构建。
4. 运行 `check`，处理结构与源码依赖偏差，再执行范围检查。设计调整由对应设计责任方完成，不通过放宽规则掩盖实现问题。

```bash
swe-cli system-design validate --root /absolute/project --file system-design.yaml --profile layers
swe-cli system-design validate --root /absolute/project --file system-design.yaml
swe-cli system-design check --root /absolute/project --file system-design.yaml
```

`validate` 与 `check` 输出 JSON。退出码 0 表示本次范围通过，1 表示违规，2 表示无法检查；结果包含设计 SHA256、生成器版本、覆盖范围、未验证项及实际跨模块依赖的文件和行号。检查读取当前文件，不提供原子快照，应在静止工作区运行。

`render` 通过设计校验后将 HTML 写到标准输出，失败输出 JSON 诊断且返回非零。使用临时文件避免失败时覆盖原图；仅在退出码为 0 时替换最终视图：

```bash
swe-cli system-design render --root /absolute/project --file system-design.yaml > system-design.pending.html && mv system-design.pending.html system-design.html
```

请求查看图时按下述流程重新生成；显式查看历史产物时，以设计 SHA256 区分它与当前 YAML，不将历史视图称为最新结果。生成视图属于明确写入动作，须纳入任务可写范围；检查命令不写项目。该入口独立于现有 JSON `check` 请求，不修改协议版本 2。

## 查看与展示流程

system-design Skill 组织查看流程，SWE CLI 校验指定 YAML 并生成独立 HTML，Agent 保存产物并调用宿主提供的浏览器或 HTML 预览工具。用户请求查看已授权生成预览与打开；不因生成文件再次确认，也不因此修改 YAML 或扩展为实现检查。仅编辑 YAML 不自动打开，明确只读且不允许生成文件的请求只展示已有产物并说明时效。

1. 确定目标：优先使用用户明确指定、当前正在设计或刚编辑的 YAML。上下文不足时执行 `system-design list --root <项目目录>`；只有一份则直接使用，多份且无法确定目标时才询问，并列出相对路径。没有设计则报告未找到，不凭空创建。用户要求查看全部时逐份调用 `render --file`，不使用不存在的 `render --all`，也不合并各作用域为一张图。
2. 确定输出：默认在宿主允许写入的项目外缓存或临时目录保存。以项目真实根路径与 YAML 相对路径共同标识产物，例如对两者的组合取摘要作为子目录，保留源路径到 HTML 的对应关系；同一设计复用输出位置，不同项目或同名 YAML 不互相覆盖。正式交付物按用户指定目录保存，并纳入可写范围。
3. 生成最新视图：每次查看从当前 YAML 运行 `render`，它已包含完整设计校验，无需先重复执行 `validate`。输出先写到同一缓存子目录中的临时文件，退出码为 0 才替换预览 HTML。失败则报告该 YAML 的诊断，不替换旧图或把旧图作为本次结果；其他目标继续处理。仅层设计尚不能通过完整渲染时，说明当前限制，不补造模块或放宽校验。
4. 展示结果：使用当前宿主支持的浏览器或 HTML 预览能力打开成功产物；能够识别并操作对应标签页时复用并刷新，不能仅因路径相同就认为页面已更新。工具不可用或不支持该文件时提供可打开的 HTML 文件入口并说明限制，不绕过宿主访问限制。打开请求已排队不等于已展示，按工具反馈区分已打开、待打开与需用户手动打开。
5. 交付入口：返回目标 YAML 与对应 HTML 的可点击入口。多份结果提供以相对路径为标签的汇总入口，列明失败项；不默认打开大量标签页。汇总可使用回复中的链接清单，宿主适合时也可在同一缓存目录生成索引页。HTML 是派生预览，架构内容仍以 YAML 为准。

缓存位置和浏览器工具由 Agent 根据宿主选择，不写入共享源码中的机器路径，不要求安装全局工具或额外启动服务。项目外缓存不属于 Git 范围检查的覆盖对象，仍需限定本次产物位置。

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

仓库中的可运行三模块 Go 示例位于 `apps/swe-cli/internal/systemdesign/testdata/project/`，包含层间与同层模块依赖。开发阶段可从 `apps/swe-cli` 执行：

```bash
go run . system-design validate --root internal/systemdesign/testdata/project --file system-design.yaml
go run . system-design check --root internal/systemdesign/testdata/project --file system-design.yaml
go run . system-design render --root internal/systemdesign/testdata/project --file system-design.yaml > /tmp/harn-system-design.html
```

## 版本与配套工具

该能力需要包含本次实现的新版 SWE CLI。旧 `architecture` 命令和 `architecture.yaml` 文件名不再支持；YAML 层设计局部模式改为 `--profile layers`，旧 Markdown 文档的 `--profile architecture` 保持兼容。使用与 Skill 一起构建的工具，不把本地构建通过当作跨平台运行验收。
