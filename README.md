# Harn

让 Agent 的工作有方法、有边界、有验证。

Harn 是一套 Agent harness，为设计、实现、测试和审查提供可按需使用的工作方法。能力接入项目后，Agent 根据任务从对应入口加载方法和约束。

代码审查方法关注问题的位置、影响和依据；代码修改方法明确修改范围和验证方式。这些方法以 Skills 保存，可以随项目一起管理和更新。

## 使用示例

项目内安装工具和 SWE 的命令（macOS / Linux）：

```bash
curl -fsSL https://raw.githubusercontent.com/AwayerCode/Harn/main/install.sh | sh
.harness/bin/harn install --source https://github.com/AwayerCode/Harn --version tag:v0.2.1 --path Skills/swe
```

支持读取项目文件的 Agent 可通过以下任务指令使用 SWE：

> 先读取 `.harness/Skills/swe/SKILL.md`，选择代码审查方法，审查当前项目尚未提交的代码改动。只报告有依据的问题，说明文件位置、影响和原因，不修改文件。缺少必要依据时明确说明。

审查对象也可以是具体文件或模块。

安装完成后，入口位于 `.harness/Skills/swe/SKILL.md`。项目接入、版本管理和更新说明见 [使用指南](HARN.md)。Windows 下载方式见 [安装](HARN.md#安装)。

## 许可证

[MIT](LICENSE)
