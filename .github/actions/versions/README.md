# Pinned versions

This action reports the versions pinned in [`.versions`](../../../.versions),
so that calling workflows can get to them easily.

## Usage

```yaml
steps:
  - uses: smallstep/workflows/.github/actions/versions@<sha> # <tag>
    id: versions

  - uses: actions/setup-go@<sha> # <tag>
    with:
      go-version: ${{ steps.versions.outputs.go }}

  - uses: golangci/golangci-lint-action@<sha> # <tag>
    with:
      version: v${{ steps.versions.outputs['golangci-lint'] }}
```

Please note that the versions reported are what the `.versions` file holds
at the SHA you call `actions/versions` at.

## Outputs

An output is named by its key in `.versions`, less any `go:` prefix.

| Output                              | Description                                         |
| ----------------------------------- | --------------------------------------------------- |
| `__json`                            | All of the below, in a `{"key": "version"}` object. |
| `go`                                | The Go release to build, test and lint with.        |
| `golangci-lint`                     | The `golangci-lint` release to lint with.           |
| `golangci-lint-langserver`          | The `golangci-lint-langserver` release.             |
| `gotestsum`                         | The `gotestsum` release to run tests with.          |
| `golang.org/x/tools/cmd/goimports`  | The `goimports` release.                            |
| `golang.org/x/tools/gopls`          | The `gopls` release.                                |
| `golang.org/x/vuln/cmd/govulncheck` | The `govulncheck` release to scan with.             |
| `github.com/air-verse/air`          | The `air` release.                                  |
| `mirrord`                           | The `mirrord` release.                              |

The reported versions carry no leading `v`, e.g. `1.0.0` instead of `v1.0.0`,
which means that, depending where you want to pass the literal to, you may have
to prefix the version with a `v`. `golangci/golangci-lint-action`, for example,
requires it.

`__json` carries the same names and the same versions as one object, which is
how you reach a name via an expression and `fromJSON`.

```yaml
- run: go install "golang.org/x/vuln/cmd/govulncheck@v${VERSION}"
  env:
    VERSION: ${{ fromJSON(steps.versions.outputs.__json)['golang.org/x/vuln/cmd/govulncheck'] }}
```
