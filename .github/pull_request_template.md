# Pull Request

## 变更

简要说明问题和修复方式。

## 验证

- [ ] `find CLI/harn -name '*.go' -type f -print0 | xargs -0 gofmt -l`
- [ ] `(cd CLI/harn && go test ./...)`
- [ ] `(cd CLI/harn && go test -race ./...)`
- [ ] `(cd CLI/harn && go vet ./...)`
- [ ] `python3 -B Skills/swe/scripts/tests/test_standards.py`
- [ ] `python3 -B Skills/knowledge-doc-guard/scripts/knowledge_lint.py --all`
- [ ] `git diff --check`

## 风险

说明未覆盖场景、兼容影响或需要人工确认的事项。
