# Harn

[![CI](https://github.com/AwayerCode/Harn/actions/workflows/ci.yml/badge.svg)](https://github.com/AwayerCode/Harn/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/AwayerCode/Harn)](https://github.com/AwayerCode/Harn/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

<img src="assets/social-preview.png" alt="Harn: Agent Skills as project dependencies" width="100%">

Harn treats Agent Skills as versioned project dependencies. Skill sources live in one repository, each project downloads only the paths it selects, and the source, version, file hashes and permissions are recorded in the project's own `.harness/downloads.json`.

The problem is not "install one more global tool". Skills that live only in a global directory cannot be reviewed, reproduced or updated together with the code they serve. Harn keeps that state inside the project, so a different machine, a different agent or a clean checkout continues from the same record.

## Why Harn

- Reproducible per project: resources land in `.harness/<original path>` and never depend on a global directory.
- Auditable provenance: every download records the GitHub source, branch, tag or commit, plus file hashes and permissions.
- Protected updates: every recorded file is verified before writing; local edits, deletions, permission changes or path collisions fail the whole update before anything is touched.
- Releases are binaries: GitHub Releases ship six platform binaries and `SHA256SUMS`, no archives to unpack.
- Concurrent-safe: install and update hold a project-level lock so two processes cannot overwrite the same record.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/AwayerCode/Harn/main/install.sh | sh
```

The script verifies `SHA256SUMS` and installs the binary to `.harness/bin/harn`. Pin a version with `--version`, choose another project root with `--root`, or pick a custom location with `--bin-dir`:

```bash
curl -fsSL https://raw.githubusercontent.com/AwayerCode/Harn/main/install.sh \
  | sh -s -- --version v0.2.0 --root .
```

If you prefer to install by hand, `HARN.md` contains the step-by-step download and verification commands for macOS, Linux and Windows PowerShell.

## Use

```bash
# Find a skill in the published index
harn search
harn search review

# Download the paths this project needs
harn install \
  --source https://github.com/AwayerCode/Harn \
  --version tag:v0.2.0 \
  --path Skills/concise-response \
  --path Skills/knowledge-doc-guard \
  --path Skills/swe

# Inspect and maintain what is recorded
harn list
harn status
harn update --all
harn update --path Skills/swe --version tag:v0.3.0 --dry-run
harn remove --path Skills/concise-response
```

<img src="assets/demo.gif" alt="harn search, install, list and status" width="100%">

Downloaded entry points live at `.harness/Skills/<name>/SKILL.md`. An agent reads that entry point and loads the supporting methods, templates and scripts through the skill's own relative references.

`--version` accepts `branch:<name>`, `tag:<name>` and `commit:<sha>`. `--source` accepts a GitHub repository URL or a direct `.tar.gz` URL; a direct archive must contain the selected paths, with or without a top-level directory. `--dry-run` performs resolution, download and verification without writing files or changing the record.

If you already have Go installed, install from the tagged submodule. The release workflow creates `CLI/harn/vX.Y.Z` alongside each release:

```bash
go install github.com/AwayerCode/Harn/CLI/harn@v0.2.0
```

## Migrating from Harness

The `harness` command is now `harn`, release assets changed from `harness_<os>_<arch>` to `harn_<os>_<arch>`, and the Go module path is now `github.com/AwayerCode/Harn/CLI/harn`.

The project state directory is still `.harness/` and `downloads.json` keeps its schema, so existing records need no migration. Run `harn list` and `harn status` after swapping the binary. The old `harness` binary is no longer updated; do not mix the two.

The repository moved from `AwayerCode/Harness` to `AwayerCode/Harn`. GitHub redirects the old web URLs, but Go module paths are not redirected, so projects importing this module must update their imports.

## Commands

| Command | Purpose |
| --- | --- |
| `harn search [query]` | Search the published skill index. |
| `harn install` | Download one or more resource paths. |
| `harn update` | Update recorded paths, or every path with `--all`. |
| `harn list` | List recorded paths, versions and file counts. |
| `harn status` | Compare recorded files with local content and permissions. |
| `harn remove` | Remove recorded paths, refusing when local changes exist. |
| `harn version` | Print the binary version. |

## How updates stay safe

`update` verifies every recorded file before it writes anything. A single modified, deleted or permission-changed file fails the whole operation and leaves both the files and `downloads.json` untouched. New local files are never deleted, and an upstream file that would land on an unrecorded local path is refused instead of overwritten.

`remove` follows the same rule: it only deletes files that are still byte-identical to the record. Local additions inside a removed directory are preserved.

## Repository layout

```tree
.
├── Skills/                 # Skill sources, contracts, templates, scripts and tests
├── CLI/
│   └── harn/               # First self-contained Go module and build scripts
├── assets/                 # Logo, wordmark, social preview and terminal demo
├── install.sh              # Verified one-line installer
├── .github/                # CI, release workflow and collaboration templates
├── HARN.md                 # Manual install and operating guide
├── AGENTS.md               # Repository collaboration rules
└── LICENSE
```

Release binaries are built by CI and uploaded to GitHub Releases. `CLI/harn/dist/` is for local verification only.

## Development

Requires Go 1.24+, Python 3 and Bash:

```bash
find CLI/harn -name '*.go' -type f -print0 | xargs -0 gofmt -l
(cd CLI/harn && go test ./...)
(cd CLI/harn && go test -race ./...)
(cd CLI/harn && go vet ./...)
python3 -B Skills/swe/scripts/tests/test_standards.py
python3 -B Skills/knowledge-doc-guard/scripts/knowledge_lint.py --all
```

Build the six release assets:

```bash
CLI/harn/scripts/build-release.sh v0.2.0
```

## Contributing and security

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request. Do not publish exploit details in a public issue; use the private channel described in [SECURITY.md](SECURITY.md).

## License

[MIT License](LICENSE)
