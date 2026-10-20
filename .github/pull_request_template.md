# Pull Request

## 变更

简要说明问题和修复方式。

## 验证

- [ ] `find CLI/harness -name '*.go' -type f -print0 | xargs -0 gofmt -l`
- [ ] `(cd CLI/harness && go test ./...)`
- [ ] `(cd CLI/harness && go test -race ./...)`
- [ ] `(cd CLI/harness && go vet ./...)`
- [ ] `python3 -B Skills/swe/scripts/tests/test_standards.py`
- [ ] `python3 -B Skills/knowledge-doc-guard/scripts/knowledge_lint.py --all`
- [ ] `git diff --check`

## 风险

说明未覆盖场景、兼容影响或需要人工确认的事项。
