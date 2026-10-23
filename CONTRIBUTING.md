# 贡献指南

开发与验证遵循 [AGENTS.md](AGENTS.md)。修改范围、验证命令与发布授权以仓库约定为准。

## Skill 版本登记

[skills/index.json](skills/index.json) 记录仓库中每个顶层 Skill 的独立版本和来源，`schema_version` 表示登记表结构版本。名称与目录中的 `SKILL.md` 一致，每个目录只登记一次。本仓目录由 `skills/<name>` 推导；搜索读取登记表的名称和 `description`，并支持匹配推导出的路径。名称和简介与入口文件同步维护；构建测试检查公开顶层 Skill 是否完整登记，以及目录、名称、简介是否一致（简介比较忽略换行与重复空白）。

| 字段 | 含义 |
| --- | --- |
| `name` | 唯一名称，与目录名一致，使用小写字母、数字和单连字符，最长 64 字符 |
| `description` | 与 `SKILL.md` frontmatter 一致的简介，用于搜索 |
| `version` | 本仓库维护的 Skill 版本，与 Harn 和 SWE CLI 版本独立 |
| `updated_at` | 当前 Skill 内容的更新时间；已有内容初始化时取最近一次目录提交时间 |
| `source.repository`、`source.path` | 自有 Skill 均为 `null`；外部 Skill 填上游仓库 URL 和目录，仓库根目录为 `"."` |

时间使用带时区的 RFC 3339 格式。初始登记的 Skill 从 `0.1.0` 建立独立版本基线，不代表追溯认定历史发布版本。后续内容修正递增 PATCH、兼容能力增加递增 MINOR、不兼容的使用契约变化递增 MAJOR；内容与版本、`updated_at` 同次提交。只修订登记信息时不改变内容版本或内容更新时间。历史记录由 Git 保存。

所有 Skill 使用相同字段结构。首次登记时填写 `updated_at`，后续内容修改或同步上游时更新。引入外部 Skill 时保留原有许可文件和署名，确认许可允许分发；同步时核对本地修改，并递增本仓维护的 `version`。索引不记录上游版本或 commit，首次加入时间、导入内容及变更历史由本仓 Git 保存。

CLI 使用登记表搜索，但尚不按这些独立版本解析安装目标，也不能用它们替代安装命令的 `--version`：安装仍选择仓库标签或 commit。SWE 配套二进制版本继续由其 `runtime.json` 固定。

## 开发版配套检查工具

仓库只创建 Harn Release。`cli/harn/scripts/build-release.sh <版本>` 构建六平台 harn，并调用 SWE 打包脚本，将六个 `swe_<系统>_<架构>.tar.gz` 与 harn 一起放入 `cli/harn/dist/`，生成统一 `SHA256SUMS`。SWE CLI 只存在于 Skill 包内部，不发布裸二进制或 `swe-cli-v*` 标签。

`cli/swe-cli/scripts/build-package.sh <版本>` 可独立用于本地打包验收；中间二进制位于其 `dist/bin/`。打包从 SWE 源码复制方法与资源，加入对应平台 CLI，并生成不含下载地址的 `runtime.json`。构建不修改 Skill 源码；完整包发布前不能承诺远端安装可用。

跨平台构建不替代实机验收。完整开发验证见[本地验证](AGENTS.md#本地验证)，检查核心的开发说明见 [SWE CLI](cli/swe-cli/README.md)。
