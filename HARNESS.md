# Harness

Harness 集中维护可复制到不同项目的 Agent Skills 和附属内容。用户或 Agent 先阅读本文件，选择当前项目需要的资源，再通过 Go 分发工具下载到项目自己的 `.harness/`。

> 不要把整个仓库直接复制进业务项目，也不要让业务项目依赖某个全局 Harness 安装。下载记录保存在项目内，随项目一起提交。

## 资源

| 路径 | 使用时机 |
| --- | --- |
| `Skills/swe` | 软件设计、实现、测试和审查任务；按当前层与模块选择专业工作流，不默认执行全流程。 |
| `Skills/concise-response` | 每轮对话和最终回复；控制表达密度，但不得削减任务交付内容。 |
| `Skills/knowledge-doc-guard` | 读取、整理、写入或检查知识文档时；约束内容完整性、Markdown 格式和质量检查。 |

资源的具体用法以各自 `SKILL.md` 及其相对引用为准。本文件只负责让 Agent 找到入口，不复制完整方法。

## 下载工具

工具提供 `linux`、`darwin` 和 `windows` 的 `amd64`、`arm64` 原生二进制，发布页直接提供可执行文件，不需要解压。以下命令以 `v0.1.1` 为例，并把工具放在当前项目的 `.harness/bin/`。

示例使用 `AwayerCode/Harness` 作为仓库地址；Agent 应根据实际收到的 Harness 仓库链接替换 `repo`。

macOS 和 Linux：

```bash
repo="AwayerCode/Harness"
version="v0.1.1"
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$(uname -m)" in
  arm64|aarch64) arch="arm64" ;;
  x86_64|amd64) arch="amd64" ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac
asset="harness_${os}_${arch}"
base="https://github.com/${repo}/releases/download/${version}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p .harness/bin
curl -fsSL "$base/$asset" -o "$tmp/$asset"
curl -fsSL "$base/SHA256SUMS" -o "$tmp/SHA256SUMS"
(
  cd "$tmp"
  if command -v sha256sum >/dev/null 2>&1; then
    grep "  $asset$" SHA256SUMS | sha256sum -c -
  else
    grep "  $asset$" SHA256SUMS | shasum -a 256 -c -
  fi
)
cp "$tmp/$asset" .harness/bin/harness
chmod +x .harness/bin/harness
```

Windows PowerShell：

```powershell
$repo = "AwayerCode/Harness"
$version = "v0.1.1"
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$asset = "harness_windows_$arch.exe"
$base = "https://github.com/$repo/releases/download/$version"
$tmp = Join-Path $env:TEMP ("harness-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  Invoke-WebRequest "$base/$asset" -OutFile (Join-Path $tmp $asset)
  Invoke-WebRequest "$base/SHA256SUMS" -OutFile (Join-Path $tmp "SHA256SUMS")
  $expected = ((Get-Content (Join-Path $tmp "SHA256SUMS")) | Where-Object { $_ -match "  $asset$" }).Split()[0]
  $actual = (Get-FileHash (Join-Path $tmp $asset) -Algorithm SHA256).Hash.ToLower()
  if ($actual -ne $expected.ToLower()) { throw "SHA256 mismatch for $asset" }
  New-Item -ItemType Directory -Path ".harness/bin" -Force | Out-Null
  Copy-Item (Join-Path $tmp $asset) -Destination ".harness/bin/harness.exe" -Force
} finally {
  Remove-Item -Recurse -Force $tmp
}
```

确认工具可用：

```bash
.harness/bin/harness version
```

Windows 使用 `.harness/bin/harness.exe` 替代本文命令中的 `.harness/bin/harness`。

## 下载资源

`--version` 使用 `branch:<name>`、`tag:<name>` 或 `commit:<sha>` 标识来源版本。为保持项目可复现，优先使用发布标签；需要跟踪更新时使用分支。`--path` 可重复，工具不会替 Agent 判断应该下载哪些资源。

```bash
.harness/bin/harness install \
  --source https://github.com/AwayerCode/Harness \
  --version tag:v0.1.1 \
  --path Skills/concise-response \
  --path Skills/knowledge-doc-guard \
  --path Skills/swe
```

下载保留 Harness 仓库中的相对路径，因此 `Skills/swe` 位于 `.harness/Skills/swe`。工具自动创建 `.harness/downloads.json`，记录来源、版本、文件哈希和权限。

## 使用资源

Agent 直接读取 `.harness/<原路径>/SKILL.md`，并按其中的相对引用加载配套方法、模板和脚本。需要稳定入口时，由 Agent 按业务项目约定建立相对软链接或修改项目级 Agent 配置；Harness 工具不建立软链接，也不修改包管理配置。

## 更新资源

```bash
.harness/bin/harness update --path Skills/swe
```

`update` 读取 `.harness/downloads.json` 的来源和版本并重新下载。需要切换版本时显式指定：

```bash
.harness/bin/harness update --path Skills/swe --version tag:v0.2.0
```

更新前会核对所有已记录文件。任一文件被修改、删除或权限改变时，整次更新在写入前失败；新增的本地文件不会被删除，遇到同路径上游文件也会拒绝覆盖。失败后先处理本地修改，或由用户决定保留本地版本。

查看当前项目记录：

```bash
cat .harness/downloads.json
```

Windows 可使用 `Get-Content .harness/downloads.json`。

## 保存

把整个 `.harness/` 提交到业务项目的 Git 仓库。克隆项目后可直接继续使用原有内容，不需要全局安装。

不同操作系统和架构需要各自的工具二进制。切换平台时按本文重新下载对应版本；应用资源和 `downloads.json` 保持原路径。提交、推送和软链接接入遵循业务项目自己的授权。

分发工具只下载和更新显式指定的资源，不选择资源、不解释 Skill、不自动提交，也不修改业务项目的 `.gitignore`。

## 源码布局

CLI 源码统一位于仓库的 `CLI/`：

```tree
CLI/
└── harness/                    # 第一个 CLI，完整且自包含
    ├── go.mod
    ├── main.go
    ├── internal/               # 当前 CLI 的共享实现
    ├── scripts/                # 当前 CLI 的构建脚本
    └── dist/                   # 本机构建产物，Git 忽略
```

`CLI/` 只是多个 CLI 的容器。后续 CLI 使用各自的子目录、Go module、实现和构建脚本。发布二进制由 CI 构建并上传到 GitHub Releases，不提交到源码仓库。
