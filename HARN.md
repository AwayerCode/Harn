# Harn 使用指南

## 安装

在目标项目目录执行（macOS / Linux）：

```bash
curl -fsSL https://raw.githubusercontent.com/AwayerCode/Harn/main/install.sh | sh
.harness/bin/harn version
```

脚本校验后将工具安装到 `.harness/bin/`。Windows 从 [Releases](https://github.com/AwayerCode/Harn/releases/latest) 下载对应架构的二进制，校验后保存为 `.harness/bin/harn.exe`；下文命令相应使用 `.exe`，多行命令合并为一行。

## 接入项目

先选择需要的能力，可用 `search` 查看或搜索：

```bash
.harness/bin/harn search
.harness/bin/harn search review
```

| 路径 | 用途 |
| --- | --- |
| `Skills/swe` | 软件设计、实现、测试与审查。 |
| `Skills/concise-response` | 简洁表达，保留完整交付内容。 |
| `Skills/knowledge-doc-guard` | 知识文档整理与质量检查。 |

以 SWE 和 `v0.2.1` 为例，版本可按需替换：

```bash
.harness/bin/harn install \
  --source https://github.com/AwayerCode/Harn \
  --version tag:v0.2.1 \
  --path Skills/swe
```

`--path` 可重复，一次安装多个资源。`--version` 支持 `tag:<版本>`、`commit:<SHA>` 和 `branch:<分支>`；需要固定版本时使用标签或提交。`search` 默认搜索主分支索引，可加 `--version tag:v0.2.1` 与安装版本保持一致。

`--source` 支持 GitHub 仓库或直接 `.tar.gz` URL；归档需包含所选路径，可带或不带顶层目录。安装前可加 `--dry-run` 下载并验证，不写入项目资源和下载记录。

下载后，让 Agent 读取入口：

> 读取 `.harness/Skills/swe/SKILL.md`，按其中适用的方法审查当前项目，先给出发现，不修改文件。

需要持续使用时，将该入口写入项目的 Agent 配置。Agent 按入口中的相对引用加载方法、模板和脚本；Harn 不自动修改配置或建立软链接。

## 日常管理

查看已安装资源和本地修改：

```bash
.harness/bin/harn list
.harness/bin/harn status
```

`status` 检查记录文件，发现修改或缺失时返回非零退出码。

先预演，再更新全部资源：

```bash
.harness/bin/harn update --all --dry-run
.harness/bin/harn update --all
```

只更新某项时，用 `--path Skills/swe` 替代 `--all`。更新默认沿用记录中的来源和版本，不会自动将固定标签升级到最新版本。切换版本时显式指定：

```bash
.harness/bin/harn update --path Skills/swe --version tag:<目标版本>
```

将 `<目标版本>` 替换为实际发布标签后执行。`--dry-run` 下载并验证更新，不写入项目资源和下载记录。

移除不再需要的能力：

```bash
.harness/bin/harn remove --path Skills/swe
```

更新和移除只检查本次选中资源的记录文件；文件内容改变、缺失或可执行状态改变时，整次操作在修改资源前失败。先处理本地修改再重试。本地新增文件会保留，更新也会拒绝覆盖与上游同路径的本地新增文件。

权限检查仅区分可执行与不可执行文件，Windows 跳过权限比较；`status` 使用相同规则。

## 保存到 Git

提交下载的资源目录和 `.harness/downloads.json`。记录文件保存来源、版本、文件哈希和权限，资源保留仓库中的相对路径。

在项目 `.gitignore` 中加入：

```gitignore
.harness/bin/
.harness/.lock
```

克隆项目后，Agent 可直接读取已保存的能力；需要下载或更新时，再安装当前平台的工具。Harn 不自动提交，也不修改 `.gitignore`。
