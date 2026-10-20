# Harn 使用指南

本指南以 SWE 代码审查为例，介绍能力安装、项目接入和日常管理。命令的工作目录为目标项目，无需克隆 Harn 仓库。

当前源码目录为 `cli/`、`skills/` 和 `docs/`。下文固定使用已发布的 `v0.2.1`，该版本资源路径仍为 `Skills/swe`；历史版本路径大小写不随源码改名而改变。新版使用其发行索引中的 `skills/swe`，不要混用两种路径。

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
.harness/bin/harn install --source https://github.com/AwayerCode/Harn --version tag:v0.2.1 --path Skills/swe
```

示例固定使用 `v0.2.1`，入口位于 `.harness/Skills/swe/SKILL.md`。[README 的任务指令](README.md#使用示例)可用于首次审查；审查对象也可以是具体文件或模块。Agent 读取对应方法后执行任务，报告问题依据和未验证项。

## 项目接入

项目约定文件可以保存能力入口，供 Agent 在后续任务中读取。对于支持 `AGENTS.md` 的 Agent，接入内容可追加到项目根目录的 `AGENTS.md`，与已有规则共存：

```markdown
## Harn

执行软件设计、实现、测试或审查任务前，先读取 `.harness/Skills/swe/SKILL.md`，按当前任务加载相关方法。

沿用本项目已有约定；只修改任务涉及的文件，保留已有改动。完成后说明结果、实际验证和未验证项。
```

该配置由项目维护，Harn 不自动写入。具体任务仍通过会话指定，例如“审查登录模块的错误处理，不修改代码”。

下载的能力文件和记录可随项目纳入 Git：

```bash
git add .harness/Skills/swe .harness/downloads.json
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

切换版本时使用 `--version` 指定目标版本，发布标签见 [Releases](https://github.com/AwayerCode/Harn/releases)。以下以 `v0.2.1` 为目标演示预演与实际更新；与安装示例版本相同，不代表升级到新版本：

```bash
.harness/bin/harn update --path Skills/swe --version tag:v0.2.1 --dry-run
.harness/bin/harn update --path Skills/swe --version tag:v0.2.1
```

`--dry-run` 会下载并检查资源，不改写项目资源和记录。不指定 `--version` 时沿用原版本，不会自动升级到最新标签；更新全部已安装能力可用 `--all` 替代 `--path Skills/swe`。

本地修改会阻止更新，`status` 可显示存在变化的资源。本地新增文件会保留，与上游文件路径冲突时拒绝覆盖。

SWE 的移除命令：

```bash
.harness/bin/harn remove --path Skills/swe
```

移除同样检查本地修改，并保留本地新增文件。项目约定中的接入文字需同步移除，Harn 不自动修改该文件。

`search` 用于查找其他能力。安装命令通过 `--path` 选择资源，重复该参数可一次下载多个能力：

```bash
.harness/bin/harn search
```

<details>
<summary>版本与检查规则</summary>

- 版本支持 `tag:<标签>`、`commit:<SHA>` 和 `branch:<分支>`。搜索默认读取主分支索引，可加 `--version tag:v0.2.1` 与安装版本保持一致。
- 来源支持 GitHub 仓库 URL 或直接 `.tar.gz` URL；归档需包含所选路径。
- 更新和移除检查本次选中资源的记录文件。内容改变、文件缺失或可执行状态改变时停止操作；Windows 不比较权限，其他平台仅区分是否可执行。

</details>
