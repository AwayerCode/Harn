# 领域 Skill 与 CLI 配套

领域 Skill 定义方法、产物所有权与判断标准，项目设计提供具体规则，CLI 检查已编码约束，项目测试和运行工具提供行为证据。CLI 不选择任务顺序、不授予写入权限，也不以退出码证明整个任务完成。

## 安装与版本

仅安装 `skills/swe`，五个内部领域、共享规则、Schema 与 CLI 配套随包统一更新。领域不独立安装或登记版本；运行时工具版本仍由唯一的 `runtime.json` 固定。

`runtime.json` 只在 `swe` 根目录中维护，固定六平台 `swe-cli` 摘要；二进制位于 `<共享SWE目录>/bin/swe-cli`，Windows 使用 `.exe`。各领域不各自复制二进制、Schema 或共享规则。缺失包内资源、工具或绑定不一致时明确无法检查，先按项目安装方式补齐，不切换旧脚本。进行中的范围基线继续使用创建它的工具，不通过升级重建基线掩盖改动。

当前开发版配套工具为 `0.4.0-alpha.2`，发行资产尚未发布；源码开发可用本仓构建产物，不能承诺远端安装已可用。各领域 `contract.json` 的包内资源与 Schema 引用用于验证完整性，不是独立安装清单。

## 选择检查

| 任务对象 | 检查 | 结论边界 |
| --- | --- | --- |
| 业务模型 | `documents --kind business-model` | 格式与本地链接，抽象与规则仍由领域判断 |
| YAML 层／模块设计 | `architecture validate`，局部层设计加 `--profile architecture` | Schema、引用、路径冲突与声明依赖，不要求代码已存在 |
| YAML 设计视图 | `architecture render` | 设计通过后输出 HTML，不代表实现验收 |
| YAML 下的生产代码或测试 | `architecture check` | 目录归属、必要文件与单 Go module 导入；接口与行为未覆盖 |
| Markdown 系统设计及协议 | `documents`、`design` | 约定格式与显式关系，不等同 YAML 检查 |
| Markdown 代码映射 | `structure` | 显式位置与入口，不解析源码依赖 |
| 测试设计 | `documents --kind test-design`、`artifacts --kind test` | 风险、场景显式引用；断言与行为由测试验证 |
| 任意领域写入／只读审查 | `scope snapshot/check` 或 `scope readonly` | Git 工作区范围与指纹，不是权限隔离 |
| 构建、测试与部署 | 项目已有工具 | CLI 不替代语言工具、测试执行器和部署平台 |

只运行与当前对象相关的检查。YAML 项目不为满足旧 `design` 或 `structure` 命令复制 Markdown 架构表格。独立协议可运行 `documents --kind interface-protocol`，YAML 与协议的关联目前需要人工核对。不支持的语言、workspace 或关系说明未验证，不当作通过。

## 命令入口

以下 `<CLI>` 指SWE 包实际二进制，所有命令显式指定目标根目录，文件一般使用相对项目路径。

```bash
<CLI> version
<CLI> documents --root <项目根目录> --kind business-model --file <模型路径>
<CLI> architecture validate --root <项目根目录> --file <YAML路径>
<CLI> architecture validate --root <项目根目录> --file <YAML路径> --profile architecture
<CLI> architecture check --root <项目根目录> --file <YAML路径>
<CLI> architecture render --root <项目根目录> --file <YAML路径>
<CLI> design --root <项目根目录> --system-design <Markdown路径> --interface-protocol <协议路径>
<CLI> structure --root <项目根目录> --design <Markdown路径> --scope <相对路径>
<CLI> documents --root <项目根目录> --kind test-design --file <测试设计路径>
<CLI> artifacts --root <项目根目录> --kind test --file <测试设计路径>
```

旧交互 Skill 已删除；CLI 中已有交互文档检查能力保留兼容，不作为本包的领域入口。渲染写入项目文件是架构领域的明确写入动作，须纳入范围，使用临时输出成功后替换，避免错误诊断覆盖原图；具体规范见 [YAML 架构](architecture/guide.md)。

## 范围与结果

范围命令及副作用按[执行规范](standards/execution.md)使用。混合源码目录不能直接作为代码或测试的整目录写入范围；共享配置条目需要差异复核，CLI 目前只做到文件粒度。

```bash
<CLI> scope snapshot --root <Git根目录> --baseline <项目外全新文件> --allow-file <相对路径>
<CLI> scope check --root <Git根目录> --baseline <原文件>
<CLI> scope readonly --root <Git根目录>
<CLI> scope readonly --root <Git根目录> --expected <完整原指纹>
```

`allow-file` 或 `allow-dir` 可重复，不能通过扩大原基线消除违规。范围检查需要 Git，包含跟踪与非忽略文件和子模块，显式可写范围还覆盖其忽略文件；不初始化或拉取子模块，不允许父仓范围跨入子模块。

退出码 0 表示本次覆盖通过，1 表示违规，2 表示无法检查。`documents/design/artifacts/structure` 可由 JSON 协议版本 2 的 `check --request` 调用，结果数组为 `checks`，每项包含 `status`、`diagnostics`、`evidence` 和 `coverage`，`unverified` 为可选字段。输入观察不是原子快照。

`architecture` 是独立入口，正常检查及参数、文件读取等失败均使用 `swe-architecture-v1` 结果，包含设计摘要、诊断、观察依赖和覆盖边界；它没有旧请求的输入稳定性复核，应在静止工作区运行。`render` 成功输出 HTML，检查或生成失败输出同一架构 JSON；输出通道自身不可写时只能以退出码 2 报告，不能保证诊断可送达。必须同时检查退出码。未能完整读取设计时 `design_sha256` 为空。`scope` 也使用独立格式。不要按一套 JSON 字段解析全部命令，不把工具成功扩大为语义、行为或部署完成。
