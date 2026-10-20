# Harn 协作约定

本仓是 Harn 的完整源码位置：`Skills/` 保存可版本化的 Agent 能力，`CLI/harn/` 保存项目内资源分发工具，测试与构建在本地执行，二进制手动发布到 GitHub Releases。业务项目只下载明确选择的资源，不复制整个仓库，也不依赖全局安装。

## Skill 入口

执行前按需读取目标 Skill 的入口与相关资源：

- [swe](Skills/swe/SKILL.md)：软件工程总入口，以共同标准、系统分层和模块契约组织专业设计、按层实现与集成验收。
- [concise-response](Skills/concise-response/SKILL.md)：每轮对话使用，控制立场和表达；简洁不减少必须交付的内容。
- [knowledge-doc-guard](Skills/knowledge-doc-guard/SKILL.md)：用于知识文档读取、整理、写入和质量检查。

## 协作约定

先明确任务范围、权威依据和可写目录，只修改当前任务涉及的内容。设计或修改 Skill 时保留有效方法、职责边界和检查要求，不复制第二套共享规则。

按改动类型执行 [README.md](README.md) 中的验证命令，检查受影响的链接、JSON、Python/YAML 语法和工作区差异。

部署路径和凭据不写入共享源码。提交、推送和发布遵循用户的具体授权。
