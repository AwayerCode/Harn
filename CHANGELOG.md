# Changelog

Harn 的版本记录以 GitHub Releases 为准；本文件保留主要变更的可读摘要。

## v0.2.0

- 项目更名为 Harn，CLI 命令、Go module 路径和发布资产统一使用 `harn`。
- 使用说明文件更名为 `HARN.md`，仓库地址改为 `AwayerCode/Harn`。
- 保留项目内 `.harness/` 目录和 `downloads.json` 结构，已有项目无需迁移。

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
