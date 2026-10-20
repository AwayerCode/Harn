# Harn

Harn 集中维护可复制到不同项目的 Agent Skills 和附属内容。用户或 Agent 先阅读本文件，选择当前项目需要的资源，再通过 Go 分发工具下载到项目自己的 `.harness/`。

> 不要把整个仓库直接复制进业务项目，也不要让业务项目依赖某个全局 Harn 安装。下载记录保存在项目内，随项目一起提交。

## 资源

| 路径 | 使用时机 |
| --- | --- |
| `Skills/swe` | 软件设计、实现、测试和审查任务；按当前层与模块选择专业工作流，不默认执行全流程。 |
| `Skills/concise-response` | 每轮对话和最终回复；控制表达密度，但不得削减任务交付内容。 |
| `Skills/knowledge-doc-guard` | 读取、整理、写入或检查知识文档时；约束内容完整性、Markdown 格式和质量检查。 |

资源的具体用法以各自 `SKILL.md` 及其相对引用为准。本文件负责资源选择和使用，不复制 Skill 的专业方法。

机器可读索引位于 `Skills/index.json`，`harn search` 读取它列出可用 Skill；新增 Skill 时同步登记，规范见 [Skills/authoring.md](Skills/authoring.md)。

## 下载工具

工具提供 `linux`、`darwin` 和 `windows` 的 `amd64`、`arm64` 原生二进制，发布页直接提供可执行文件，不需要解压。以下命令以 `v0.2.1` 为例，并把工具放在当前项目的 `.harness/bin/`。

示例使用 `AwayerCode/Harn` 作为仓库地址；Agent 应根据实际收到的 Harn 仓库链接替换 `repo`。

日常安装只需一行，脚本会按平台选择资产并校验 `SHA256SUMS`：

```bash
curl -fsSL https://raw.githubusercontent.com/AwayerCode/Harn/main/install.sh | sh
```

需要逐步审计下载过程时使用手动流程。

macOS 和 Linux：

```bash
repo="AwayerCode/Harn"
version="v0.2.1"
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$(uname -m)" in
  arm64|aarch64) arch="arm64" ;;
  x86_64|amd64) arch="amd64" ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac
asset="harn_${os}_${arch}"
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
cp "$tmp/$asset" .harness/bin/harn
chmod +x .harness/bin/harn
```

Windows PowerShell：

```powershell
$repo = "AwayerCode/Harn"
$version = "v0.2.1"
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$asset = "harn_windows_$arch.exe"
$base = "https://github.com/$repo/releases/download/$version"
$tmp = Join-Path $env:TEMP ("harn-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  Invoke-WebRequest "$base/$asset" -OutFile (Join-Path $tmp $asset)
  Invoke-WebRequest "$base/SHA256SUMS" -OutFile (Join-Path $tmp "SHA256SUMS")
  $expected = ((Get-Content (Join-Path $tmp "SHA256SUMS")) | Where-Object { $_ -match "  $asset$" }).Split()[0]
  $actual = (Get-FileHash (Join-Path $tmp $asset) -Algorithm SHA256).Hash.ToLower()
  if ($actual -ne $expected.ToLower()) { throw "SHA256 mismatch for $asset" }
  New-Item -ItemType Directory -Path ".harness/bin" -Force | Out-Null
  Copy-Item (Join-Path $tmp $asset) -Destination ".harness/bin/harn.exe" -Force
} finally {
  Remove-Item -Recurse -Force $tmp
}
```

确认工具可用：

```bash
.harness/bin/harn version
```

Windows 使用 `.harness/bin/harn.exe` 替代本文命令中的 `.harness/bin/harn`。

## 下载资源

`--version` 使用 `branch:<name>`、`tag:<name>` 或 `commit:<sha>` 标识来源版本。为保持项目可复现，优先使用发布标签；需要跟踪更新时使用分支。`--path` 可重复，工具不会替 Agent 判断应该下载哪些资源。`--source` 支持 GitHub 仓库 URL 或直接 `.tar.gz` URL；直接归档需包含所选路径，可带或不带顶层目录。

```bash
.harness/bin/harn install \
  --source https://github.com/AwayerCode/Harn \
  --version tag:v0.2.1 \
  --path Skills/concise-response \
  --path Skills/knowledge-doc-guard \
  --path Skills/swe
```

下载保留 Harn 仓库中的相对路径，因此 `Skills/swe` 位于 `.harness/Skills/swe`。工具自动创建 `.harness/downloads.json`，记录来源、版本、文件哈希和权限。

不确定路径时先搜索索引：

```bash
.harness/bin/harn search
.harness/bin/harn search review
```

首次安装前可用 `--dry-run` 下载并验证资源，不写入项目资源和下载记录：

```bash
.harness/bin/harn install --source https://github.com/AwayerCode/Harn \
  --version tag:v0.2.1 --path Skills/swe --dry-run
```

## 使用资源

Agent 直接读取 `.harness/<原路径>/SKILL.md`，并按其中的相对引用加载配套方法、模板和脚本。需要稳定入口时，由 Agent 按业务项目约定建立相对软链接或修改项目级 Agent 配置；Harn 工具不建立软链接，也不修改包管理配置。

## 更新资源

```bash
.harness/bin/harn update --path Skills/swe
```

`update` 读取 `.harness/downloads.json` 的来源和版本并重新下载。需要切换版本时显式指定：

```bash
.harness/bin/harn update --path Skills/swe --version tag:<目标版本>
```

更新前会核对所有已记录文件。任一文件被修改、删除或权限改变时，整次更新在写入前失败；新增的本地文件不会被删除，遇到同路径上游文件也会拒绝覆盖。失败后先处理本地修改，或由用户决定保留本地版本。

一次更新全部记录，或先看会改什么：

```bash
.harness/bin/harn update --all --dry-run
.harness/bin/harn update --all
```

查看记录与本地文件是否一致：

```bash
.harness/bin/harn list
.harness/bin/harn status
```

`status` 逐文件比对哈希和权限，`ok` 之外的状态会以非零退出码结束，可用于脚本检查。

## 移除资源

```bash
.harness/bin/harn remove --path Skills/swe
```

`remove` 只删除仍与记录一致的文件。目录里的本地新增文件会保留，任一记录文件被修改时整次移除失败。

查看当前项目记录：

```bash
cat .harness/downloads.json
```

Windows 可使用 `Get-Content .harness/downloads.json`。

## 保存

提交 `.harness/downloads.json` 和下载后的资源目录。`.harness/bin/` 是按平台重新下载的本地工具目录，`.harness/.lock` 是运行时锁文件，两者都不应提交。克隆项目后，资源记录和内容可直接继续使用，不需要全局安装。

不同操作系统和架构需要各自的工具二进制。切换平台时按本文重新下载对应版本；应用资源和 `downloads.json` 保持原路径。提交、推送和软链接接入遵循业务项目自己的授权。

分发工具只下载和更新显式指定的资源，不选择资源、不解释 Skill、不自动提交，也不修改业务项目的 `.gitignore`。
