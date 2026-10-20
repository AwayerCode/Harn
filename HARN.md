# Harn 使用指南

## 安装

在目标项目目录执行（macOS / Linux）：

```bash
curl -fsSL https://raw.githubusercontent.com/AwayerCode/Harn/main/install.sh | sh
.harness/bin/harn version
```

脚本下载最新发布的工具，校验 `SHA256SUMS` 后安装到 `.harness/bin/`。工具支持 macOS、Linux、Windows 的 amd64 和 arm64。手动安装与下文资源下载以 `v0.2.1` 为例，可替换为所需的发布版本。

<details>
<summary>macOS / Linux 手动安装</summary>

在子 Shell 中执行，下载或校验失败即停止，校验通过后才复制二进制。

```bash
(
set -eu
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
curl -fsSL "$base/$asset" -o "$tmp/$asset"
curl -fsSL "$base/SHA256SUMS" -o "$tmp/SHA256SUMS"
expected="$(awk -v asset="$asset" '$2 == asset { print $1; count++ } END { if (count != 1) exit 1 }' "$tmp/SHA256SUMS")"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$asset")"
else
  actual="$(shasum -a 256 "$tmp/$asset")"
fi
actual="${actual%% *}"
if [ "$actual" != "$expected" ]; then
  echo "SHA256 mismatch for $asset" >&2
  exit 1
fi
mkdir -p .harness/bin
cp "$tmp/$asset" .harness/bin/harn
chmod +x .harness/bin/harn
)
```

</details>

<details>
<summary>Windows PowerShell 安装</summary>

```powershell
$repo = "AwayerCode/Harn"
$version = "v0.2.1"
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$asset = "harn_windows_$arch.exe"
$base = "https://github.com/$repo/releases/download/$version"
$tmp = Join-Path $env:TEMP ("harn-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp -ErrorAction Stop | Out-Null
try {
  Invoke-WebRequest "$base/$asset" -OutFile (Join-Path $tmp $asset) -ErrorAction Stop
  Invoke-WebRequest "$base/SHA256SUMS" -OutFile (Join-Path $tmp "SHA256SUMS") -ErrorAction Stop
  $entries = @(Get-Content (Join-Path $tmp "SHA256SUMS") -ErrorAction Stop | Where-Object { $_ -match ("  " + [regex]::Escape($asset) + "$") })
  if ($entries.Count -ne 1) { throw "Expected one checksum for $asset" }
  $expected = ($entries[0] -split "\s+")[0]
  $actual = (Get-FileHash (Join-Path $tmp $asset) -Algorithm SHA256 -ErrorAction Stop).Hash.ToLower()
  if ($actual -ne $expected.ToLower()) { throw "SHA256 mismatch for $asset" }
  New-Item -ItemType Directory -Path ".harness/bin" -Force -ErrorAction Stop | Out-Null
  Copy-Item (Join-Path $tmp $asset) -Destination ".harness/bin/harn.exe" -Force -ErrorAction Stop
} finally {
  Remove-Item -Recurse -Force $tmp
}
.harness/bin/harn.exe version
```

</details>

Windows 使用 `.harness/bin/harn.exe` 替代下文的 `.harness/bin/harn`；多行命令请合并为一行执行。

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

以 SWE 为例：

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
