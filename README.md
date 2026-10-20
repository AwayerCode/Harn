# Harness

[![CI](https://github.com/AwayerCode/Harness/actions/workflows/ci.yml/badge.svg)](https://github.com/AwayerCode/Harness/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/AwayerCode/Harness)](https://github.com/AwayerCode/Harness/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Harness 把 Agent Skills 当作可版本化的项目依赖：Skill 源码集中维护，使用仓库时只下载明确选择的资源，并把来源、版本、文件哈希和权限记录在项目自己的 `.harness/downloads.json` 中。

它解决的不是“再安装一个全局工具”，而是让 Agent 能力随项目一起复现、审查和更新。业务项目提交 `.harness/downloads.json` 和下载资源后，换机器、换 Agent 或重新打开项目都能从同一份记录继续工作。

## 为什么使用 Harness

- 项目内可复现：资源落在 `.harness/<原路径>`，不依赖某个全局目录。
- 来源可审计：每次下载记录 GitHub 来源、分支/标签/提交、文件哈希和权限。
- 更新有保护：更新前核对所有已记录文件；本地修改、删除、权限变化或路径冲突会让整次更新在写入前失败。
- 发布即二进制：Release 直接提供六种平台产物和 `SHA256SUMS`，不使用压缩包。
- 并发安全：安装和更新持有项目级锁，避免两个进程同时覆盖状态记录。

## 快速开始

从 [最新 Release](https://github.com/AwayerCode/Harness/releases/latest) 下载当前平台的可执行文件：

| 系统 | `amd64` | `arm64` |
| --- | --- | --- |
| macOS | `harness_darwin_amd64` | `harness_darwin_arm64` |
| Linux | `harness_linux_amd64` | `harness_linux_arm64` |
| Windows | `harness_windows_amd64.exe` | `harness_windows_arm64.exe` |

下载 `SHA256SUMS` 后校验文件，再把二进制放到项目的 `.harness/bin/`。完整安装脚本和 Windows PowerShell 示例见 [HARNESS.md](HARNESS.md)。

```bash
.harness/bin/harness version
.harness/bin/harness install \
  --source https://github.com/AwayerCode/Harness \
  --version tag:v0.1.2 \
  --path Skills/concise-response \
  --path Skills/knowledge-doc-guard \
  --path Skills/swe
```

下载后的入口位于 `.harness/Skills/<name>/SKILL.md`。Agent 读取入口后，按该 Skill 自己的规范加载配套资源。

## CLI

`CLI/harness/` 是第一个独立 CLI，使用自己的 Go module，不依赖仓库根目录的 `go.mod`。后续 CLI 放在 `CLI/<name>/`。

```bash
harness install --source <url> --version <ref> --path <path> [--path <path>...]
harness update --path <path> [--path <path>...] [--version <ref>]
harness version
```

`--version` 支持 `branch:<name>`、`tag:<name>` 和 `commit:<sha>`。`--source` 可以是 GitHub 仓库 URL，也可以是直接指向 `.tar.gz` 的 URL；直接归档必须包含所选路径，工具会自动识别有无顶层归档目录。

## 目录结构

```tree
.
├── Skills/                 # Skill 源码、契约、模板、脚本和测试
├── CLI/
│   └── harness/            # 第一个自有 Go module 和构建脚本
│       ├── internal/       # CLI 内部实现包
│       ├── scripts/        # 发布构建脚本
│       └── dist/           # 本机构建产物，Git 忽略
├── .github/                # CI、Release 和协作模板
├── HARNESS.md              # 使用说明
├── AGENTS.md               # 仓库协作约定
└── LICENSE
```

发布二进制由 CI 构建并上传到 GitHub Releases，不提交到源码仓库。`CLI/harness/dist/` 只用于本地验证。

## 开发验证

需要 Go 1.24+、Python 3 和 Bash：

```bash
find CLI/harness -name '*.go' -type f -print0 | xargs -0 gofmt -l
(cd CLI/harness && go test ./...)
(cd CLI/harness && go test -race ./...)
(cd CLI/harness && go vet ./...)
python3 -B Skills/swe/scripts/tests/test_standards.py
python3 -B Skills/knowledge-doc-guard/scripts/knowledge_lint.py --all
```

构建六平台发布产物：

```bash
CLI/harness/scripts/build-release.sh v0.1.2
```

## 贡献与安全

提交前阅读 [CONTRIBUTING.md](CONTRIBUTING.md)。安全问题不要公开提交利用细节，请按 [SECURITY.md](SECURITY.md) 的私密渠道报告。

## 许可证

本项目使用 [MIT License](LICENSE)。
