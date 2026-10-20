# 贡献指南

感谢你改进 Harness。这个仓库同时维护 Skill 文档、共享校验器和项目内分发 CLI，提交应保持职责边界清晰。

## 开发环境

- Go 1.24 或更新版本
- Python 3
- Bash
- 支持 `git`、`sha256sum` 或 `shasum` 的本地环境

## 修改原则

1. 先读取目标 Skill 的 `SKILL.md` 及其相关规范。
2. 只修改任务需要的文件，不复制第二套共享规则。
3. CLI 源码放在 `CLI/<name>/`，每个 CLI 使用独立 Go module。
4. 二进制只发布到 GitHub Releases，不提交到源码仓库。
5. 不提交凭据、部署路径或本地构建产物。

## 验证

提交前至少运行：

```bash
find CLI/harness -name '*.go' -type f -print0 | xargs -0 gofmt -l
(cd CLI/harness && go test ./...)
(cd CLI/harness && go test -race ./...)
(cd CLI/harness && go vet ./...)
python3 -B Skills/swe/scripts/tests/test_standards.py
python3 -B Skills/knowledge-doc-guard/scripts/knowledge_lint.py --all
git diff --check
```

涉及发布脚本或 CLI 的改动还要验证六平台交叉编译和 `SHA256SUMS`。

## 提交与 Pull Request

使用 Conventional Commits 风格，例如 `fix(cli): hold the project lock through save`。一个 Pull Request 聚焦一个可验证的变更，说明动机、影响范围、实际执行的测试及未覆盖风险。

提交前确认工作区没有意外文件，文档链接和 JSON/YAML/Python 语法均可解析。
