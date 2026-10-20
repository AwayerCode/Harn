# Harn

[![Latest release](https://img.shields.io/github/v/release/AwayerCode/Harn)](https://github.com/AwayerCode/Harn/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Harn 是个人维护的 Agent Skills 仓库，附带 Go CLI，将选定资源下载到项目自己的 `.harness/`，随项目保存和更新。

下载记录包含来源、版本、文件哈希和权限。更新拒绝覆盖本地修改，路径检查、原子写入、回滚和项目锁用于保护项目文件。

## 使用

按 [HARN.md](HARN.md) 选择资源、安装工具、下载和更新。支持 macOS、Linux、Windows 的 amd64 和 arm64，无需全局安装。

版本记录以 [GitHub Releases](https://github.com/AwayerCode/Harn/releases) 为准。更名后仍沿用 `.harness/` 和 `downloads.json`，已有项目无需迁移这两个路径。

原 v0.2.1 变更说明：修复 `go install` 安装的二进制把版本显示为 `dev`，改为回退到模块版本。此条保留待发布说明核对，发布后移入对应 Release。

## 源码

- `Skills/`：Skill 入口、专业方法、契约、模板及校验器。
- `CLI/harn/`：独立 Go module、测试和发布构建脚本。
- [AGENTS.md](AGENTS.md)：Agent 在本仓库工作时的约定。

本地构建后手动将六个平台的二进制和 `SHA256SUMS` 上传到 GitHub Releases，不提交到源码仓库。

## 开发

需要 Go 1.24+、Python 3、Bash 和 Git；校验发布资产需要 `sha256sum` 或 `shasum`。

修改 CLI 后运行：

```bash
find CLI/harn -name '*.go' -type f -print0 | xargs -0 gofmt -l
(cd CLI/harn && go test -race ./...)
(cd CLI/harn && go vet ./...)
CLI/harn/scripts/build-release.sh
```

构建脚本生成六平台二进制和 `SHA256SUMS`，默认放在被 Git 忽略的 `CLI/harn/dist/`。交叉编译不代替目标平台的实际运行验证。

修改 Skills 后运行对应测试；SWE 的共享标准和模块文档检查为：

```bash
python3 -B Skills/tests/test_index.py
python3 -B Skills/swe/scripts/tests/test_standards.py
for test in Skills/swe/*/scripts/tests/test_validate_documents.py; do
  python3 -B "$test"
done
```

修改安装或构建脚本后运行：

```bash
sh -n install.sh
bash -n CLI/harn/scripts/build-release.sh
bash scripts/test-install.sh
```

Windows 运行验证需在 Windows 上进入 `CLI/harn` 执行 `go test -race ./...` 和 `go vet ./...`。Skill 创作约定见 [Skills/authoring.md](Skills/authoring.md)。

修改文档后运行：

```bash
python3 -B Skills/knowledge-doc-guard/scripts/knowledge_lint.py --changed
git diff --check
```

提交或 PR 说明改动原因、实际验证和未覆盖风险即可。安全问题通过 [SECURITY.md](SECURITY.md) 的私密渠道报告。

## 许可证

[MIT License](LICENSE)
