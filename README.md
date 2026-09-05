# Diceware [![Build](https://github.com/hastefuI/diceware/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/hastefuI/diceware/actions/workflows/ci.yml) [![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)](https://go.dev) [![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/hastefuI/diceware/blob/main/LICENSE) [![Go Reference](https://pkg.go.dev/badge/go.hasteful.org/diceware.svg)](https://pkg.go.dev/go.hasteful.org/diceware)

A modern TUI-inspired [Diceware](https://en.wikipedia.org/wiki/Diceware) passphrase generator written in Go with [Lip Gloss](https://github.com/charmbracelet/lipgloss).

<img src="./demo.gif" alt="Diceware Demo" style="width:100%; max-width:900px;" />

## Installation

### Build From Source

```bash
$ go build -o diceware ./cmd/diceware
$ go install ./cmd/diceware
```

### Docker

```bash
$ docker build -t diceware .
```

### Verify Installation

```bash
$ diceware -version
```

## Quick Start

Generate passphrases:

```bash
$ diceware -i wordlists/wordlist-basque-diceware.txt -n 7 -interval 2s
```

Join the words with something other than a space:

```bash
$ diceware -i wordlists/wordlist-basque-diceware.txt -sep - -plain
malko-beira-soldadu-plastifikatu-ekuazio-datxa
```

## Usage

```bash
diceware -i <wordlist-file|-> [flags]
  -i          wordlist file, or - for stdin       (required)
  -n          number of passphrases to generate   (default 1)
  -w          number of words per passphrase      (default 6)
  -sep        string placed between words         (default " ")
  -interval   refresh interval in live mode       (default 5s)
  -once       generate once and exit              (default false)
  -plain      print only the passphrases and exit (default false)
  -version    print the version and exit
  -h          print help and exit
```

When standard output is not a terminal the live view is skipped and only the
passphrases are printed, so a pipe or a redirect behaves like `-plain` without
the flag. Colors follow [NO_COLOR](https://no-color.org/).

## Library

The wordlist reader and the generator are importable without the CLI:

```bash
$ go get go.hasteful.org/diceware
```

```go
package main

import (
	"fmt"

	"go.hasteful.org/diceware"
)

func main() {
	words, err := diceware.LoadWords("wordlists/wordlist-basque-diceware.txt")
	if err != nil {
		panic(err)
	}

	phrases, err := diceware.GenerateList(words, 1, 6, " ")
	if err != nil {
		panic(err)
	}
	fmt.Println(phrases[0])
}
```

## Motivation

<a href="https://xkcd.com/936/" title="Password Strength by xkcd"><img src="./xkcd-936.svg" width="550"></a>

[https://xkcd.com/936/](https://xkcd.com/936/)

## License

Licensed under the [MIT License](https://opensource.org/licenses/MIT). See
[LICENSE](./LICENSE) for details.

Copyright (c) 2026-present hasteful
