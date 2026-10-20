# Harn 使用指南

本指南以 SWE 代码审查为例，介绍能力安装、项目接入和日常管理。命令的工作目录为目标项目，无需克隆 Harn 仓库。

当前源码目录为 `cli/`、`skills/` 和 `docs/`。下文使用 `v0.2.2`，资源路径为 `skills/swe`；旧版 `v0.2.1` 使用 `Skills/swe`，历史版本路径大小写不随源码改名而改变。

## 安装

macOS / Linux：

```bash
curl -fsSL https://raw.githubusercontent.com/AwayerCode/Harn/main/install.sh | sh
.harness/bin/harn version
```

工具安装在项目的 `.harness/bin/`，`version` 输出当前工具版本。

Windows 从 [Releases](https://github.com/AwayerCode/Harn/releases/latest) 下载对应架构的二进制，用同页的 `SHA256SUMS` 校验后保存为 `.harness/bin/harn.exe`。下文命令相应使用 `.exe`。

## 代码审查示例

SWE 包含软件设计、实现、测试和审查方法，安装命令为：

```bash
.harness/bin/harn install --source https://github.com/AwayerCode/Harn --version tag:v0.2.2 --path skills/swe
```

示例固定使用 `v0.2.2`，入口位于 `.harness/skills/swe/SKILL.md`。[README 的任务指令](README.md#使用示例)可用于首次审查；审查对象也可以是具体文件或模块。Agent 读取对应方法后执行任务，报告问题依据和未验证项。

## 项目接入

项目约定文件可以保存能力入口，供 Agent 在后续任务中读取。对于支持 `AGENTS.md` 的 Agent，接入内容可追加到项目根目录的 `AGENTS.md`，与已有规则共存：

```markdown
## Harn

执行软件设计、实现、测试或审查任务前，先读取 `.harness/skills/swe/SKILL.md`，按当前任务加载相关方法。

沿用本项目已有约定；只修改任务涉及的文件，保留已有改动。完成后说明结果、实际验证和未验证项。
```

该配置由项目维护，Harn 不自动写入。具体任务仍通过会话指定，例如“审查登录模块的错误处理，不修改代码”。

下载的能力文件和记录可随项目纳入 Git：

```bash
git add .harness/skills/swe .harness/downloads.json
```

接入约定文件一并纳入版本管理；工具二进制与运行时锁文件由项目 `.gitignore` 排除：

```gitignore
.harness/bin/
.harness/.lock
```

忽略范围仅包含上述两项，能力目录和下载记录仍保留在 Git 中。项目克隆到其他机器后，Agent 可直接读取能力；下载或更新时需要安装对应平台的工具。

## 日常管理

`list` 列出已安装资源，`status` 检查资源与下载记录是否一致：

```bash
.harness/bin/harn list
.harness/bin/harn status
```

切换版本时使用 `--version` 指定目标版本，发布标签见 [Releases](https://github.com/AwayerCode/Harn/releases)。以下以 `v0.2.2` 为目标演示预演与实际更新；与安装示例版本相同，不代表升级到新版本：

```bash
.harness/bin/harn update --path skills/swe --version tag:v0.2.2 --dry-run
.harness/bin/harn update --path skills/swe --version tag:v0.2.2
```

`--dry-run` 会下载并检查资源，不改写项目资源和记录。不指定 `--version` 时沿用原版本，不会自动升级到最新标签；更新全部已安装能力可用 `--all` 替代 `--path skills/swe`。

本地修改会阻止更新，`status` 可显示存在变化的资源。本地新增文件会保留，与上游文件路径冲突时拒绝覆盖。

SWE 的移除命令：

```bash
.harness/bin/harn remove --path skills/swe
```

移除同样检查本地修改，并保留本地新增文件。项目约定中的接入文字需同步移除，Harn 不自动修改该文件。

`search` 用于查找其他能力。安装命令通过 `--path` 选择资源，重复该参数可一次下载多个能力：

```bash
.harness/bin/harn search
```

<details>
<summary>版本与检查规则</summary>

- 版本支持 `tag:<标签>`、`commit:<SHA>` 和 `branch:<分支>`。当前源码中的搜索默认读取主分支的 `skills/index.json`，按名称、路径和简介匹配，可加 `--version` 选择包含新格式索引的标签或 commit。harn v0.2.2 仍扫描 `SKILL.md`；新版搜索不会回退扫描，缺少索引的历史版本会报错。
- 来源支持 GitHub 仓库 URL 或直接 `.tar.gz` URL；归档需包含所选路径。
- 更新和移除检查本次选中资源的记录文件。内容改变、文件缺失或可执行状态改变时停止操作；Windows 不比较权限，其他平台仅区分是否可执行。

</details>

安装脚本安装二进制后自动列出可用 Skills；指定 `--version` 时搜索对应标签，未指定时搜索主分支。列表查询失败会提示重试，不撤销二进制安装。

## Skill 版本登记

[skills/index.json](skills/index.json) 记录仓库中每个顶层 Skill 的独立版本和来源，`schema_version` 表示登记表结构版本。名称与目录中的 `SKILL.md` 一致，每个目录只登记一次。搜索只读取登记表的名称、路径和 `description`。名称和简介与入口文件同步维护；构建测试检查公开顶层 Skill 是否完整登记，以及目录、名称、简介是否一致（简介比较忽略换行与重复空白）。

| 字段 | 含义 |
| --- | --- |
| `name`、`path` | Skill 名称及本仓库相对目录 |
| `description` | 与 `SKILL.md` frontmatter 一致的简介，用于搜索 |
| `version` | 本仓库维护的 Skill 版本，与 Harn 和 SWE CLI 版本独立 |
| `registered_at` | 首次纳入登记表的时间，后续更新保留 |
| `updated_at` | 当前 Skill 内容的更新时间；已有内容初始化时取最近一次目录提交时间 |
| `source.type` | 自有 Skill 为 `internal`，外部引入为 `external` |
| `source.repository`、`source.path` | 原始仓库 URL 和该仓库内的 Skill 目录 |
| `source.revision` | 外部 Skill 必填，实际导入内容对应的完整 commit SHA |
| `source.version` | 外部上游发布版本；上游无版本时为 `null`，不能用分支名代替版本 |
| `source.synced_at` | 外部 Skill 最近一次实际导入或同步的时间 |

时间使用带时区的 RFC 3339 格式。现有三个 Skill 从 `0.1.0` 建立独立版本基线，不代表追溯认定历史发布版本。后续内容修正递增 PATCH、兼容能力增加递增 MINOR、不兼容的使用契约变化递增 MAJOR；内容与版本、`updated_at` 同次提交。只修订登记信息时不改变内容版本或内容更新时间。历史记录由 Git 保存。

引入外部 Skill 时，将内容放入独立目录并登记来源与固定 commit；保留原有许可文件和署名，确认许可允许分发。本地修改只递增本仓库的 `version` 和 `updated_at`，保留原始上游版本、commit 和同步时间；再次同步上游时才更新来源记录，并核对本地修改。

CLI 使用登记表搜索，但尚不按这些独立版本解析安装目标，也不能用它们替代安装命令的 `--version`：安装仍选择仓库标签或 commit。SWE 配套二进制版本继续由其 `runtime.json` 固定。

## 开发版配套检查工具

当前源码中的 harn 支持 Skill 的 runtime.json：安装或更新 SWE 时下载清单固定版本的 swe-cli，验证 SHA256 后与资源一起写入 Skill 的 bin 目录。该能力尚未包含在 harn v0.2.2 中；当前绑定的 SWE CLI 0.3.0-alpha.1 也需单独发布。发布前按 CLI README 构建并本地验收，不能使用旧工具解释新清单。

业务项目应忽略 `.harness/skills/swe/bin/` 中的平台二进制。跨平台补装与专用修复命令尚未完成；不要把某个平台的二进制提交为通用资源。
