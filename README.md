# Harn

让 Agent 的工作有方法、有边界、有验证。

Harn 是一套随项目演进的 Agent harness，将专业方法、执行约束和验收标准组织为按需加载的能力，帮助 Agent 完成设计、实现、测试与审查。

## 如何工作

Agent 根据任务加载相关能力，在项目约定内执行，用检查结果验证交付。简单修改只用必要规则，复杂任务再展开设计与验收。

Skills 承载方法与工具，CLI 将这些能力按版本带入项目。

## 快速开始

在目标项目目录中安装工具（macOS / Linux）：

```bash
curl -fsSL https://raw.githubusercontent.com/AwayerCode/Harn/main/install.sh | sh
```

Windows 安装方式见 [使用指南](HARN.md#安装)。

下载所需能力：

```bash
.harness/bin/harn install \
  --source https://github.com/AwayerCode/Harn \
  --version tag:v0.2.1 \
  --path Skills/swe
```

让 Agent 从入口开始工作：

> 读取 `.harness/Skills/swe/SKILL.md`，按其中适用的方法审查当前项目，先给出发现，不修改文件。

下载的能力文件与版本记录保存在项目的 `.harness/` 中，可随代码一起提交；工具二进制和运行时锁文件不提交。安装、接入和更新方式见 [使用指南](HARN.md)。

## 许可证

[MIT](LICENSE)
