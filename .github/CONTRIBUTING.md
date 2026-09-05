# Contributing to Diceware

Thanks for your interest in contributing. This is a small project, so the guidelines are short.

## Prerequisites

- [Go](https://go.dev/doc/install) 1.27 or later
- [Git](https://git-scm.com/downloads)
- [Docker](https://docs.docker.com/get-started/get-docker/), optional, and only
  for the container build

## Getting started

Clone this repo and then:

```bash
$ make build
$ ./diceware -i wordlists/wordlist-basque-diceware.txt -plain
```

`make help` lists the other targets: `install`, `run` and `test`.

The default mode is a full-screen live view that redraws every five seconds
until Ctrl+C. Use `-once` or `-plain` when you want the program to exit on its
own.

## Before you open a pull request

Run the checks CI runs:

```bash
$ go build ./...
$ go vet ./...
$ gofmt -l .
$ go test -race ./...
$ go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
```

CI also cross-compiles for linux/arm64, darwin/arm64, windows/arm64 and
freebsd/amd64, and builds the Docker image. Both are worth running locally if
you touched the build.

## Code style

- Standard Go, targeting the version in `go.mod`.
- Tests use the standard library only, please do not add a test dependency.
- The library at the module root imports nothing outside the standard library,
  and it should stay that way. Anything that reaches for a flag, the terminal
  or standard input belongs in `cmd/diceware`.

## Commits

[Conventional Commits](https://www.conventionalcommits.org), subject only. No
body, no footers, no trailers.

- Types in use: `feat`, `fix`, `docs`, `test`, `refactor`, `ci`, `build`,
  `chore`.
- Scope is optional, lowercase, and names an area rather than a file. `cli`,
  `readme`, `release` and `deps` are the established ones.
- Imperative mood, lowercase after the colon, no trailing period, and keep the
  whole subject under 72 characters.

```
feat(cli): add -sep to choose the word separator
docs(readme): document the Docker build
```

Signed commits are preferred. See [GitHub's guide to signature
verification](https://docs.github.com/en/authentication/managing-commit-signature-verification).

## Pull requests

1. Fork the repository and branch from `main`.
2. Keep the branch to one change.
3. Cover new behavior with a test in `diceware_test.go` or
   `cmd/diceware/main_test.go`.
4. Open the pull request with a conventional-commit title and a short summary
   of what changed and why.

## Security

Do not open a public issue for a vulnerability. Use [private vulnerability
reporting](https://github.com/hastefuI/diceware/security) as described in
[SECURITY.md](../SECURITY.md).

## License

By contributing you agree that your contributions are licensed under the
[MIT License](../LICENSE).
