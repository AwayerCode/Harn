# Changelog

Harn 的版本记录以 GitHub Releases 为准；本文件保留主要变更的可读摘要。

## v0.2.1

- 修复 `go install` 安装的二进制把版本显示为 `dev`，改为回退到模块版本。

## v0.2.0

- 项目更名为 Harn，CLI 命令、Go module 路径和发布资产统一使用 `harn`。
- 使用说明文件更名为 `HARN.md`，仓库地址改为 `AwayerCode/Harn`。
- 保留项目内 `.harness/` 目录和 `downloads.json` 结构，已有项目无需迁移。
- 新增 `harn search`、`harn list`、`harn status` 和 `harn remove`。
- `update` 支持 `--all`，`install` 和 `update` 支持 `--dry-run`。
- 新增 `install.sh` 一行安装脚本，校验 `SHA256SUMS` 后写入 `.harness/bin/`。
- 新增 `Skills/index.json` 资源索引、Skill 创作指南和模板。
- 新增品牌资产、英文 README、治理文档和依赖更新配置。
- 发布流程增加构建出处证明和 SBOM，CI 增加 Windows 实测、覆盖率门槛和静态检查。

## v0.1.2

- 统一仓库定位和发布说明，补齐 README、许可证及协作模板。
- 移除迁移期规则、旧入口和旧文档格式兼容。
- 收敛共享文档校验器与测试 fixture。
- 为安装和更新增加项目级并发锁。
- 修复无顶层目录归档和单文件归档的提取。
- 更新正式 Release 的安装与更新验收。

## v0.1.1

- 发布无压缩二进制和 `SHA256SUMS`。
- 完成正式 Release 安装与更新验收。

## v0.1.0

- 初始 Harn CLI 分发能力。
