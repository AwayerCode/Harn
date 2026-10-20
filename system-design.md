# Harness 系统设计

Harness 集中维护可供不同项目使用的 Skills、CLI 等内容。Agent 阅读使用说明，按项目需要下载和使用；Go 二进制只负责分发。当前已实现首个 CLI：`harness`。

## 入口

Harness 仓库根目录的 `HARNESS.md` 是对外使用入口，介绍有哪些资源、什么时候使用、如何下载 CLI，以及下载和使用资源的方法。详细用法放在各资源自己的文档中。

用户把仓库链接交给 Agent，让它阅读 `HARNESS.md`，再根据当前项目需要选择内容。业务项目也可以在自己的 `AGENTS.md` 中保留该入口链接。

Harness 仓库的 `AGENTS.md` 保留为维护本仓库的约定，不承担对外使用说明的职责。

## 工具

使用 Go 编写分发工具，提供不同操作系统和架构的二进制，使用者不需要安装 Go。

工具只提供下载和更新能力。Agent 按 `HARNESS.md` 获取工具，再调用它下载或更新指定内容。工具不解释使用说明、不选择资源、不区分 Agent，也不负责配置 Agent 或建立软链接。

新增资源或调整使用方法时，更新仓库内容和 `HARNESS.md` 即可；只有分发能力变化时才需要修改工具。

发布页直接提供各平台原生二进制，不再套一层压缩包；安装只需下载、校验并设置执行权限。

## 源码组织

根目录的 `CLI/` 只作为所有 CLI 的容器。每个 CLI 在 `CLI/` 下拥有自己的目录、Go module、实现、构建脚本和本地产物。

```tree
Harness/
├── CLI/
│   └── harness/             # 第一个 CLI，完整且自包含
│       ├── go.mod
│       ├── main.go
│       ├── internal/        # 当前 CLI 的共享实现，不对外暴露
│       ├── scripts/         # 当前 CLI 的构建与发布脚本
│       └── dist/            # 当前 CLI 的本机构建产物，Git 忽略
├── Skills/
├── HARNESS.md
└── system-design.md
```

后续 CLI 按同样的边界新增，例如 `CLI/<name>/`。每个 CLI 独立维护自己的 `go.mod`、`internal`、脚本和产物，避免多个 CLI 的 module 配置与构建流程混在 `CLI/` 根层。

Release 工作流从 `CLI/harness/` 构建各平台二进制，并把二进制和 `SHA256SUMS` 上传到 GitHub Releases。二进制和本机 `dist/` 产物不提交到源码仓库。

## 目录

下载内容统一放在业务项目的 `.harness/` 中，保留 Harness 仓库原有的相对路径，不重新分类或增加 `resources/` 层。

例如，仓库里的 `Skills/swe/` 下载到项目的 `.harness/Skills/swe/`。其他内容同样按原路径放置，打开目录就能看到项目下载了什么。

```tree
业务项目/
└── .harness/
    ├── bin/harness          # 分发工具，Windows 为 harness.exe
    ├── Skills/
    │   └── swe/
    ├── <其他原有目录>/
    └── downloads.json      # 自动维护的下载记录
```

Agent 按使用说明建立必要的相对软链接，或直接引用、执行这些内容。更新后保持原路径，方便继续使用已有引用。不设置固定的 `tmp/` 目录或 `.gitignore` 文件。

## 记录

工具自动在 `downloads.json` 中记录下载来源和版本，方便查看和后续更新，不要求用户手写配置。

每个项目独立选择和更新自己的内容，不使用全局配置，不修改业务项目的包管理配置。更新不能破坏已有内容，也不能擅自覆盖项目中的修改。

## 保存

`.harness/` 整体提交到业务项目的 Git 仓库，包括分发工具、Skills、其他下载内容和下载记录。这样 Harness 的内容和版本随项目一起保存，克隆项目时也能获得对应内容。

二进制按平台区分；换到其他平台时，按 `HARNESS.md` 获取对应版本。提交由用户或 Agent 按项目授权执行，分发工具不自动提交或推送。
