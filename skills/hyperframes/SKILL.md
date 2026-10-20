---
name: hyperframes
description: 使用 HyperFrames 将 HTML 内容制作成可编辑、可重复渲染的视频，适用于功能演示、数据解说、更新介绍、社媒短视频和多版本输出；支持现有 HyperFrames 项目的预览、检查与渲染。
---

# HyperFrames

本入口将 heygen-com/hyperframes 的公开 Skill 集合封装为一个 Harn 资源。上游内容保存在 `upstream/`，使用前读取 [HyperFrames 入口](upstream/hyperframes/SKILL.md)，按任务加载对应工作流。

## 本仓适配

- 上游的 `/hyperframes-core` 等 Skill 名称对应本目录 `upstream/<名称>/SKILL.md`。这里已包含公开的配套工作流，优先读取本地副本；上游的自动安装、刷新 Skill 步骤在此由 Harn 资源更新替代，不覆盖本仓的受管文件。
- 上游 Skill 之间的相对引用保持原样。按需加载选定工作流和领域规则，不一次读取全部内容。
- 在用户的目标项目中创建视频、媒体和配置，不在本 Skill 目录中生成作品或安装依赖。实际运行需要 Node.js、HyperFrames CLI 及所用渲染环境；登记本 Skill 不表示这些运行依赖已安装。
- 上游命令、工具名称和插件约定按当前宿主的实际能力映射。能力缺失时说明缺少的依赖，不把未执行的预览、渲染或发布报告为成功。
- 上游要求的外部发布、消息发送和付费服务使用仍遵循当前用户授权。

## 来源

上游仓库：[heygen-com/hyperframes](https://github.com/heygen-com/hyperframes)，来源目录 `skills/`。公开 Skill 保存在 `upstream/`；本文件为 Harn 新增的加载与目录适配入口。`embedded-captions/.gitignore` 增加了测试夹具例外，保证上游已跟踪的 `transcript.json` 随资源保留。许可见 [LICENSE](LICENSE)。
