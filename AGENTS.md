# AGENTS.md

Guidance for AI coding agents working in this repository.

## Overview

`smallstep/workflows` holds the reusable GitHub Actions workflows that every Smallstep Go repository calls from its own `.github/workflows/*.yml`, plus the shared `.golangci.yml` that those repositories fetch at lint time and a `.versions` file that pins the org-wide Go toolchain. The repo is almost entirely YAML and shell; the only Go is `lintapp/`, a dummy module (`github.com/smallstep/workflows/lintapp`) that exists so CI can prove the shared lint config still works. There are no tags or releases: callers reference `@main`, so **a merge to `main` takes effect in every consuming repository immediately.**

## Commands

There is no Makefile. What CI runs (`.github/workflows/ci.yml`) can be reproduced locally:

```bash
actionlint -color                                         # lint every workflow file (.github/actionlint.yml holds the ignores)
zizmor --min-severity medium --min-confidence medium .    # security audit of the workflows (.github/zizmor.yml holds the ignores)
golangci-lint config verify --config=.golangci.yml        # validate the shared lint config (needs the golangci-lint pinned in .versions)
(cd lintapp && golangci-lint run --config=../.golangci.yml)  # lint the dummy app with the shared config
(cd lintapp && go test ./...)                             # test the dummy app
.github/scripts/generate-versions --no-refresh            # rebuild .versions from .versions.json; CI fails if the result differs from what is committed
.github/scripts/generate-versions                         # refresh .versions.json from upstream (needs gh, jq, curl, tar, go); normally run only by versions.yml
```

`golangci-lint config verify` and the lintapp lint need a golangci-lint built with a Go at least as new as the `go` line in `.versions`; an older local binary refuses to load the config. Do not run `go build` inside `lintapp/`: it drops an untracked `lintapp/lintapp` binary.

## Generated code - do not edit

| File | Generator |
|------|-----------|
| `.versions` | `.github/scripts/generate-versions`, run nightly by `versions.yml`, which opens a PR on the `ci/versions` branch. Edit `.versions.json` and regenerate; never hand-edit `.versions`. |

## Layout

```
.github/workflows/
├── goCI.yml                 # umbrella: fans out to goLint, goTest, goBuild, govulncheck, codeql-analysis
├── goLint.yml               # golangci-lint (+ optional go mod tidy check, go generate drift check)
├── goTest.yml               # gotestsum + optional codecov, Go stable/oldstable matrix
├── goBuild.yml              # runs the caller's build command on the Go matrix
├── govulncheck.yml          # govulncheck ./...
├── codeql-analysis.yml      # CodeQL for Go
├── code-scan.yml            # thin wrapper around codeql-analysis for nightly cron callers
├── actionci.yml             # umbrella: actionlint + zizmor (+ a no-op frizbee job kept for compatibility)
├── actionlint.yml           # actionlint via the pinned docker image
├── zizmor.yml               # zizmor; uploads SARIF to Advanced Security on public repos by default
├── frizbee.yml              # deprecated, echoes and exits 0
├── goreleaser.yml           # GoReleaser Pro release; optional GPG signing and package-repo upload
├── docker-buildx-push.yml   # multi-platform buildx push + cosign
├── dependabot-auto-merge.yml# enables auto-merge on Dependabot PRs
├── triage.yml               # labels PRs "needs triage", adds to the OSS triage project
├── ci.yml                   # THIS repo's CI (not reusable)
├── versions.yml             # THIS repo's nightly .versions sync (not reusable)
└── sync-winget-fork.yml     # THIS repo's monthly winget-pkgs fork sync (not reusable)
.github/actions/versions/    # composite action exposing .versions as step outputs (see its README.md)
.github/scripts/generate-versions
.golangci.yml                # shared lint config consumed remotely (see below)
.versions / .versions.json   # pinned toolchain: go, golangci-lint, gotestsum, govulncheck, goimports, gopls, air, mirrord
lintapp/                     # dummy Go module used only to exercise .golangci.yml in CI
```

## Reusable workflows and their inputs

All reusable workflows are `on: workflow_call`. Callers look like:

```yaml
jobs:
  ci:
    permissions: { actions: read, contents: read, security-events: write }
    uses: smallstep/workflows/.github/workflows/goCI.yml@main
    with:
      only-latest-golang: false
      run-codeql: true
    secrets: inherit
```

- **goCI.yml** - inputs: `run-lint`, `run-test`, `run-build`, `run-govulncheck`, `run-codeql` (default false), `run-codecov`, `only-latest-golang` (true: `stable` only; false: adds `oldstable`), `build-command` (`V=1 make build`), `test-command` (gotestsum with coverage), `golangci-lint-args` (`--timeout=30m`), `golangci-lint-version` / `gotestsum-version` / `govulncheck-version` (empty means take it from `.versions`), `goprivate`, `os-dependencies` and per-job `<job>-os-dependencies`, `runs-on` and per-job `<job>-runs-on`, `setup-bats`, `lint-skip-go-generate` (default false), `lint-skip-go-mod-tidy` (default true), `codeql-build-cmd`, `codeql-build-mode`, `codeql-make-bootstrap`. Secrets: `SSH_PRIVATE_KEY`, `PAT`, `CODECOV_TOKEN`.
- **goLint.yml** - if the caller has no `.golangci.*` file, it downloads this repo's `.golangci.yml` at the calling SHA and passes `--config`. With `skip-go-generate: false` (the default) it deletes every `// Code generated ... DO NOT EDIT.` file except `*.pb.go`, runs `go generate ./...`, and fails on any diff. `skip-go-mod-tidy: false` adds `go mod tidy -diff`.
- **goTest.yml** / **goBuild.yml** - same Go matrix logic; both `eval` the caller's command string. `goTest` sets `GOTESTSUM_JSONFILE` and annotates failures.
- **actionci.yml** - inputs: `run-actionlint`, `run-zizmor`, `run-frizbee` (all default true), `zizmor-advanced-security` (string; empty auto-enables on public repos). The calling job must grant `actions: read` and `security-events: write` or the zizmor job fails at startup.
- **code-scan.yml** - inputs: `run-codeql`, `runs-on`, `codeql-build-cmd`, `codeql-build-mode`, `codeql-os-dependencies`, `os-dependencies`. Secrets: `PAT`, `SSH_PRIVATE_KEY`.
- **goreleaser.yml** - inputs: `go-version` (`stable`), `cosign-version`, `is-prerelease` (default true), `enable-gpg-sign`, `enable-packages-upload`, `os-dependencies`, `runs-on`, `goprivate`. Required secrets: `GORELEASER_PAT`, `GORELEASER_KEY`, `AWS_S3_REGION`; the rest are optional. Runs GoReleaser Pro `release --clean`.
- **docker-buildx-push.yml** - required inputs `platforms`, `tags`, `docker_image`; optional `docker_file`, `docker_build_args`, `runs_on` (note the underscore). Required secrets `DOCKER_USERNAME`, `DOCKER_PASSWORD`. Signs the pushed digest with cosign.
- **dependabot-auto-merge.yml** - required secret `DEPENDABOT_TOKEN`; only acts when the PR author is `dependabot[bot]`.
- **triage.yml** - inputs `run-label-pr`, `run-add-to-oss-triage-project`; secret `TRIAGE_PAT`. Expects a `pull_request_target` caller for the label job.
- **.github/actions/versions** - composite action; outputs one value per `.versions` key (without the `go:` prefix, no leading `v`) plus `__json`.

## Conventions

- **Pin every third-party action to a commit SHA with a `# vX.Y.Z` comment.** zizmor enforces this; `.github/zizmor.yml` only relaxes rules for the reasons stated in its comments.
- Reusable workflows fetch `.versions` and `.golangci.yml` from their own commit via `job.workflow_repository` / `job.workflow_sha`, so a caller pinned to a SHA gets the pins from that SHA. actionlint does not know those properties; the ignore lives in `.github/actionlint.yml`.
- Workflow-level `permissions: contents: read` is the floor; jobs that need more declare it. A reusable workflow can never be granted more than its caller grants, so document any extra permission a caller must add (see the `NOTE(@azazeal)` comments in `goCI.yml` and `actionci.yml`).
- Shell steps use `set -euo pipefail` and pass inputs through `env:` rather than interpolating `${{ }}` into `run:` (zizmor's template-injection check).
- Caller-supplied command strings (`build-command`, `test-command`, `codeql-build-cmd`) are `eval`ed on purpose.
- Formatting: `.editorconfig` says tabs for Go, two-space indents for YAML and JSON, four spaces for `.github/scripts/*`, LF endings everywhere.
- Go toolchain moves are gated: `go` in `.versions` only advances when golangci-lint can lint it, `actions/setup-go` can install it, and Docker Hub has the image, and only after a 7-day cooldown. Do not bump `go` by hand.

## The shared `.golangci.yml`

Consuming repositories do not vendor this file. Their `make lint` targets run `golangci-lint run --config <(curl -s https://raw.githubusercontent.com/smallstep/workflows/main/.golangci.yml)` (some still say `master`; GitHub redirects it), and `goLint.yml` does the equivalent for any caller without a local config. A change here therefore alters lint results across the org on the next run, including on PRs that touched nothing. When editing it:

- Keep only settings that differ from golangci-lint defaults (the file header says so).
- Verify with the pinned golangci-lint: `golangci-lint config verify --config=.golangci.yml`, then lint `lintapp/` with it.
- Adding a linter or removing an exclusion should be tried against at least one large consuming repository before merging.
- `gomoddirectives.replace-allow-list` is the org-wide list of permitted `replace` directives; extend it rather than adding `//nolint` in callers.

## Testing a change before merging

CI in this repo only proves the files parse, the pins resolve, and the dummy app lints. To exercise a reusable workflow for real, push a branch here and point a consuming repository's caller at it temporarily:

```yaml
uses: smallstep/workflows/.github/workflows/goCI.yml@my-branch
```

Open a PR in the consuming repo, watch the run, then restore `@main` before merging either side. Public consumers you can use for this include `smallstep/cli`, `smallstep/certificates` and `smallstep/crypto`. Remember that the moment the branch here merges, every `@main` caller picks it up.
