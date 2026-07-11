# Contributing

Thanks for considering a contribution. This page is short on purpose — the
process is ordinary GitHub flow.

## How to contribute

- **Bugs and feature requests**: open a
  [GitHub issue](https://github.com/gumeniukcom/imgprobe/issues). For anything
  security-sensitive, use the private channel described in
  [SECURITY.md](SECURITY.md) instead.
- **Code**: fork, branch, open a pull request against `master`. Small, focused
  PRs review fastest. For substantial changes, open an issue first so the design
  can be discussed before you invest time.

## Requirements for changes

1. **Tests are required for functionality changes.** New functionality must come
   with tests; bug fixes must include a regression test that fails before the
   fix. Documentation-only changes are exempt.
2. **The full check suite must pass**: `make all` runs formatting, lint, and
   tests; CI additionally runs the race detector, a `CGO_ENABLED=0` build, both
   the floor and latest Go versions, and `govulncheck`.
3. **Style** is enforced by `gofmt` and `golangci-lint` (config in
   `.golangci.yml`) — no manual style rules beyond that. Commit messages follow
   [Conventional Commits](https://www.conventionalcommits.org) (`feat:`, `fix:`,
   `docs:`, `chore:`, ...).
4. **Zero cgo, minimal dependencies.** The package must build with
   `CGO_ENABLED=0`; new dependencies beyond the standard library and
   `golang.org/x/image` need prior discussion.

## Development quickstart

```bash
make test      # go test ./...
make testrace  # go test -race ./...
make lint      # golangci-lint
make vuln      # govulncheck
```

## Conduct

Be kind and constructive. Reports of unacceptable behavior go to the maintainer
via the contact on the GitHub profile.
