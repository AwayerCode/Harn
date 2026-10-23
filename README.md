# Harn

让 Agent 的工作有方法、有边界、有验证。

Harn 是一套以 Skill 为核心的 Agent Harness：通过 Skill 连接 Agent 与方法、规则和工具，通过任务内外两个循环管理任务质量与任务推进。

## 两层架构

Agent 层负责理解任务、判断和执行；Harness 层以 Skill 为核心，组织方法流程、标准契约、CLI 和脚本等工具。

Skill 既是两层的关联，也是 Harness 内部的能力入口：Agent 读取 Skill，确定如何开展工作、何时调用工具，以及如何根据结果继续行动。

> Agent 层 ⇄ Skill ⇄ Harness 的方法、规则与工具

SWE 由业务模型、系统设计、代码实现、测试、发布交付五个领域 Skill 组成，各自拥有明确的产物边界和自检／只读审查方式；它们不是流水线。五个领域统一位于 `skills/swe`，共享规范、Schema 与一个 CLI，统一安装和版本。代码修改遵守任务范围并验证实际结果；代码审查报告问题的位置、影响和依据。

## 两个循环

任务外循环负责分发任务、接收结果，并决定下一项任务；任务内循环负责执行、校验、反馈和修正。

> 外循环：分发任务 → 任务内循环 → 接收结果 → 决定并分发下一任务
>
> 内循环：Agent 执行 → 工具校验 → 反馈问题 → Agent 修正 → 再校验

任务内满足完成条件后交付结果；受阻、停滞或达到约定上限时，返回当前状态与所需决策，由外循环决定后续行动。循环遵守任务职责：实现任务可以在授权范围内修正，只读审查以交付有依据的发现为完成，不自动进入修复。

## 稳定校验与持续改进

Skill 定义方法与流程，工具提供可执行的检查和反馈。校验通过只证明已检查的要求满足，Agent 仍需结合任务目标确认是否完成。

SWE 采用 Go CLI + Skill 的演进方案：稳定校验放入 Go 独立可执行文件 `swe-cli`，方法、流程与项目适配由 Skill、项目约定和配置承担。项目已有测试、lint 和构建由 Agent 按 Skill 调用，不设置 TS 扩展层。改进方法不能顺带降低验收标准。

当前仓库包含 Skills、Go 分发工具 `harn` 和 Go swe-cli 检查核心；knowledge-doc-guard 独立使用 Python 检查。现有 SWE 检查已迁入 Go，harn 已支持安装内含 CLI 的完整 Skill 包；发行资产及跨平台运行验收需另行完成，目标方案以系统设计和接口协议为准；任务分发调度器不在本阶段范围。

开发版增加可选的 [YAML 模块架构](skills/swe/references/architecture/guide.md)：以层目录和模块目录落实设计，校验 Go 模块依赖并生成只读 HTML；需要新版配套检查器，尚待发布。

本阶段聚焦独立任务：领域 Skill 按范围执行与检查，Agent 负责专业判断，swe-cli 提供单次确定性校验；流程类 Skill 暂不建设。目标架构、模块边界及迁移安排见[系统设计](docs/system-design.md)，数据与失败语义见[接口协议](docs/interface-protocol.md)；两份文档描述待实施方案。

## 使用指南

Skill 是交付单元，方法、资源和配套 CLI 一起安装到 `.agents/skills/<名称>/`。harn 只负责查找与分发，不常驻业务项目，不保存下载记录或项目锁，不依赖全局安装。

## 安装

当前源码的新分发行为尚未随旧版 `v0.2.2` 发布。验证当前实现时，在本仓根目录构建到项目外临时目录：

```bash
go build -C cli/harn -o /tmp/harn .
/tmp/harn version
```

以下命令在目标业务项目根目录执行，使用这个临时二进制。Windows 可构建或下载对应平台的 `harn.exe`，放在项目外临时目录；下载发行物时用同页 `SHA256SUMS` 校验。旧发行版仍有旧安装行为，不能用它验证本文新布局。

`install.sh` 现在是临时启动器：下载并校验 harn，执行 `--` 后的命令，退出后删除临时二进制。无命令时执行 `search`；`--version` 在分隔符之前选择 harn 发行版，在分隔符之后属于 harn 命令。新版本发布后可这样使用：

```bash
curl -fsSL https://raw.githubusercontent.com/AwayerCode/Harn/main/install.sh | sh -s -- -- search
```

启动器不安装 harn 到业务项目，也不自动安装 Skill。

## 安装 Skill

SWE 使用包含 CLI 的平台 Skill 包。以下示例的 `vX.Y.Z` 需替换为实际 Harn 发行标签，平台名按本机选择；新打包方式尚未发布：

```bash
/tmp/harn install --source https://github.com/AwayerCode/Harn/releases/download/vX.Y.Z/swe_darwin_arm64.tar.gz --version tag:vX.Y.Z --path skills/swe
```

无需 CLI 的 Skill 仍可从仓库标签或 commit 安装。SWE 源码目录不包含构建产物，直接安装源码归档不会自动获得 CLI；应选择完整平台包。
安装只接受 `skills/<名称>` 目录，历史路径 `Skills/<名称>` 也映射到 `.agents/skills/<名称>/`。入口必须是具有名称与简介的 `SKILL.md`，名称须与目录一致；不再分发普通文件或整个仓库。可重复 `--path` 选择多个 Skill。

harn 只下载所选 Skill 归档，将包内文件原样安装，不解释 `runtime.json` 或额外下载 CLI。SWE 包已经包含本平台 `bin/swe-cli` 和打包生成的本地摘要清单；CLI 启动时核对自身版本和摘要。安装前可用发行页 `SHA256SUMS` 验证整个包，选择正确平台。版本支持 `tag:<标签>`、`commit:<SHA>` 和 `branch:<分支>`；需要可复现安装时使用固定标签或 commit。

`--dry-run` 会下载、检查 Skill 包结构，但不写业务项目。来源支持 GitHub 仓库 URL 或直接 `.tar.gz` URL；归档需包含所选 Skill 目录。

## 项目接入

支持 `.agents/skills/` 的 Agent 可原生发现入口，无需追加 `AGENTS.md`。其他宿主可显式读取 `.agents/skills/swe/SKILL.md`。harn 不改写项目指令，发现后的刷新和启用方式遵循宿主规则。

任务指令示例：

> 使用 SWE 的代码审查方法，审查当前项目尚未提交的代码改动。只报告有依据的问题，说明文件位置、影响和原因，不修改文件。缺少必要依据时明确说明。

项目中的内容为：

```plaintext
.agents/skills/swe/
  SKILL.md
  references/
  runtime.json
  bin/swe-cli
```

Skill 文本可随项目保存；本平台二进制及随包生成的本地摘要清单由 `.gitignore` 排除：

```gitignore
.agents/skills/swe/bin/
.agents/skills/swe/runtime.json
```

```bash
git add .agents/skills/swe
```

配套 CLI 跟随 Skill 使用和替换。跨平台克隆后，二进制补装目前需先向临时项目安装相同修订，对比 Skill 文本后，将该平台 `bin/` 与 `runtime.json` 一起复制；没有专用修复命令。

## 查找与查看

```bash
/tmp/harn search
/tmp/harn list
```

`search` 从所选来源的 `skills/index.json` 按名称、路径和简介查找，默认主分支；可加 `--version` 指定含登记表的修订。没有登记表时会报错，不回退扫描旧仓库。

`list` 直接扫描 `.agents/skills/` 下含 `SKILL.md` 的目录，也能列出手动安装的 Skill。输出名称、路径和简介，不推断来源、已安装版本或本地修改状态；无下载记录也能使用。

## 换版与移除

harn 不再提供 `update`、`status`、`remove`。没有持久记录，就不声称能识别上游文件、本地新增文件或本地修改。

`install` 遇到已有目标目录一律拒绝覆盖。换版时先向另一临时项目安装指定版本，对比现有 Skill、保留本地修改，再显式替换整个 Skill 目录。移除时先核对并备份自有内容，再删除该 Skill 目录；配套 CLI 随之移除。项目 Git 负责历史与差异追踪。

安装先暂存并检查全部所选 Skill 入口，再独占创建各目标目录，完整复制资源和 CLI 后发布顶层 `SKILL.md`。普通错误会清理本次创建的目录，不覆盖并发安装者的目录。没有持久事务日志或崩溃自动恢复；异常终止留下的不完整目录需人工核对清理后重装。不要在安装期间编辑目标目录或运行尚未完成的 Skill。

## 旧安装迁移

新 harn 不读取、创建或迁移旧 `.harness/`、下载记录和锁。它只扫描当前 Skill 目录，不会自动删除旧数据。

1. 备份旧 Skill 中的本地修改和自有文件；从原记录或项目 Git 找回需安装的来源与版本。
2. 使用新版 harn 向 `.agents/skills/` 安装所选版本；如果已有同名目录，先在临时项目安装并比较。
3. 确认 Skill 和配套 CLI 可用后，移入需保留的自有内容并更新旧路径引用。
4. 确认旧目录无独有数据后，由项目维护者清理 `.harness/` 及其旧忽略规则。

## Skill 版本登记

[skills/index.json](skills/index.json) 记录仓库中每个顶层 Skill 的独立版本和来源，`schema_version` 表示登记表结构版本。名称与目录中的 `SKILL.md` 一致，每个目录只登记一次。本仓目录由 `skills/<name>` 推导；搜索读取登记表的名称和 `description`，并支持匹配推导出的路径。名称和简介与入口文件同步维护；构建测试检查公开顶层 Skill 是否完整登记，以及目录、名称、简介是否一致（简介比较忽略换行与重复空白）。

| 字段 | 含义 |
| --- | --- |
| `name` | 唯一名称，与目录名一致，使用小写字母、数字和单连字符，最长 64 字符 |
| `description` | 与 `SKILL.md` frontmatter 一致的简介，用于搜索 |
| `version` | 本仓库维护的 Skill 版本，与 Harn 和 SWE CLI 版本独立 |
| `updated_at` | 当前 Skill 内容的更新时间；已有内容初始化时取最近一次目录提交时间 |
| `source.repository`、`source.path` | 自有 Skill 均为 `null`；外部 Skill 填上游仓库 URL 和目录，仓库根目录为 `"."` |

时间使用带时区的 RFC 3339 格式。现有三个 Skill 从 `0.1.0` 建立独立版本基线，不代表追溯认定历史发布版本。后续内容修正递增 PATCH、兼容能力增加递增 MINOR、不兼容的使用契约变化递增 MAJOR；内容与版本、`updated_at` 同次提交。只修订登记信息时不改变内容版本或内容更新时间。历史记录由 Git 保存。

所有 Skill 使用相同字段结构。首次登记时填写 `updated_at`，后续内容修改或同步上游时更新。引入外部 Skill 时保留原有许可文件和署名，确认许可允许分发；同步时核对本地修改，并递增本仓维护的 `version`。索引不记录上游版本或 commit，首次加入时间、导入内容及变更历史由本仓 Git 保存。

CLI 使用登记表搜索，但尚不按这些独立版本解析安装目标，也不能用它们替代安装命令的 `--version`：安装仍选择仓库标签或 commit。SWE 配套二进制版本继续由其 `runtime.json` 固定。

## 开发版配套检查工具

仓库只创建 Harn Release。`cli/harn/scripts/build-release.sh <版本>` 构建六平台 harn，并调用 SWE 打包脚本，将六个 `swe_<系统>_<架构>.tar.gz` 与 harn 一起放入 `cli/harn/dist/`，生成统一 `SHA256SUMS`。SWE CLI 只存在于 Skill 包内部，不发布裸二进制或 `swe-cli-v*` 标签。

`cli/swe-cli/scripts/build-package.sh <版本>` 可独立用于本地打包验收；中间二进制位于其 `dist/bin/`。打包从 SWE 源码复制方法与资源，加入对应平台 CLI，并生成不含下载地址的 `runtime.json`。构建不修改 Skill 源码；完整包发布前不能承诺远端安装可用。

## 五个领域与共享配套

SWE 统一安装 `skills/swe`，内部包含 `business-model/`、`system-design/`、`implementation/`、`testing/`、`delivery/` 五个领域入口。它们代表责任与改动范围，不代表执行顺序。顶层入口按任务加载内部 Skill，不依赖宿主发现嵌套目录。旧设计／实现／Review 和交互目录已删除，有效专业方法已迁入新领域。

安装时只传入 `--path skills/swe`，使用包含新结构的 commit 或后续标签。五个领域不单独安装、更新或登记版本；`skills/index.json` 只登记一个 SWE 版本。此前试用的同级 `swe-*` 领域包应移除，避免旧入口继续被发现。

共享规则、Schema 和唯一的 `runtime.json` 位于包内，五个领域共用 `bin/swe-cli`。各领域的 `contract.json` 仅描述包内 Schema 引用，由测试检查完整性，不控制安装与权限。完整平台 Skill 包随 Harn 发布，开发环境可使用本仓打包产物。

领域负责专业判断与产物边界，CLI 检查已编码规则，项目测试与运行工具提供行为证据。文件范围检查不替代共享文件内部条目的差异复核。

## 许可证

[MIT](LICENSE)
