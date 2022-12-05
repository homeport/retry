# retry

[![License](https://img.shields.io/github/license/homeport/retry.svg)](https://github.com/homeport/retry/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/homeport/retry)](https://goreportcard.com/report/github.com/homeport/retry)
[![Tests](https://github.com/homeport/retry/workflows/Tests/badge.svg)](https://github.com/homeport/retry/actions?query=workflow%3A%22Tests%22)
[![Codecov](https://img.shields.io/codecov/c/github/homeport/retry/main.svg)](https://codecov.io/gh/homeport/retry)
[![Go Reference](https://pkg.go.dev/badge/github.com/homeport/retry.svg)](https://pkg.go.dev/github.com/homeport/retry)
[![Release](https://img.shields.io/github/release/homeport/retry.svg)](https://github.com/homeport/retry/releases/latest)

Tool to retry a command in case it fails. Prepend `retry` to your command and it will retry the command in case of an exit code other than zero.

Please note, in case data is piped into the tool, `retry` will read and buffer any data from standard input and reuse it with every other attempt.

![retry](.docs/example.png?raw=true "example usage of retry")

## Installation

### Homebrew

The `homeport/tap` has macOS and GNU/Linux pre-built binaries available:

```bash
brew install homeport/tap/retry
```

### Pre-built binaries in GitHub

Prebuilt binaries can be [downloaded from the GitHub Releases section](https://github.com/homeport/retry/releases/latest).

### Curl To Shell Convenience Script

There is a convenience script to download the latest release for Linux or macOS if you want to need it simple (you need `curl` and `jq` installed on your machine):

```bash
curl --silent --location https://raw.githubusercontent.com/homeport/retry/main/hack/download.sh | bash
```

## Contributing

We are happy to have other people contributing to the project. If you decide to do that, here's how to:

- get Go (`retry` requires Go version 1.19 or greater)
- fork the project
- create a new branch
- make your changes
- open a PR.

Git commit messages should be meaningful and follow the rules nicely written down by [Chris Beams](https://chris.beams.io/posts/git-commit/):
> The seven rules of a great Git commit message
>
> 1. Separate subject from body with a blank line
> 1. Limit the subject line to 50 characters
> 1. Capitalize the subject line
> 1. Do not end the subject line with a period
> 1. Use the imperative mood in the subject line
> 1. Wrap the body at 72 characters
> 1. Use the body to explain what and why vs. how

### Running test cases and binaries generation

Run test cases:

```bash
ginkgo run ./...
```

Create binaries:

```bash
goreleaser build --rm-dist --snapshot
```

## License

Licensed under [MIT License](https://github.com/homeport/retry/blob/main/LICENSE)
