# Harn 协作约定

本仓是 Harn 的完整源码位置：`docs/` 保存系统设计，根目录 `HARN.md` 保存使用指南，`skills/` 保存可版本化的 Agent 能力，`cli/harn/` 保存项目内资源分发工具，测试与构建在本地执行，二进制手动发布到 GitHub Releases。业务项目只下载明确选择的资源，不复制整个仓库，也不依赖全局安装。

## Skill 入口

执行前按需读取目标 Skill 的入口与相关资源：

- [swe](skills/swe/SKILL.md)：软件工程总入口，以共同标准、系统分层和模块契约组织专业设计、按层实现与集成验收。
- [concise-response](skills/concise-response/SKILL.md)：每轮对话使用，控制立场和表达；简洁不减少必须交付的内容。
- [knowledge-doc-guard](skills/knowledge-doc-guard/SKILL.md)：用于知识文档读取、整理、写入和质量检查。

## 协作约定

先明确任务范围、权威依据和可写目录，只修改当前任务涉及的内容。设计或修改 Skill 时保留有效方法、职责边界和检查要求，不复制第二套共享规则。

按改动类型执行下列验证，检查受影响的链接、JSON、Python/YAML 语法和工作区差异。

部署路径和凭据不写入共享源码。提交、推送和发布遵循用户的具体授权。

## 本地验证

命令从仓库根目录执行。修改 Go 代码后检查格式、运行测试与竞态检测、执行 vet，并构建六平台产物：

```bash
find cli/harn -name '*.go' -type f -print0 | xargs -0 gofmt -l
(cd cli/harn && go test -race ./...)
(cd cli/harn && go vet ./...)
cli/harn/scripts/build-release.sh
```

`gofmt -l` 应无输出。构建产物位于 `cli/harn/dist/`，不提交。交叉编译不代替 Windows 实际运行验证；在 Windows 上进入 `cli/harn` 执行 `go test -race ./...` 和 `go vet ./...`。

修改 Skill 时，按涉及范围运行：

```bash
python3 -B skills/swe/scripts/tests/test_standards.py
for test in skills/swe/*/scripts/tests/test_validate_documents.py; do
  python3 -B "$test" || exit 1
done
```

修改安装或构建脚本时运行：

```bash
sh -n install.sh
bash -n cli/harn/scripts/build-release.sh
```

修改文档后运行：

```bash
python3 -B skills/knowledge-doc-guard/scripts/knowledge_lint.py --changed
git diff --check
```
