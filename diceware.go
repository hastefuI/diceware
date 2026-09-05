// Package diceware turns a diceware wordlist into passphrases.
package diceware

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

// LoadWords reads a wordlist from the file at path.
func LoadWords(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open wordlist %q: %w", path, err)
	}
	defer f.Close()

	return ReadWords(f, fmt.Sprintf("%q", path))
}

// ReadWords reads a wordlist from r, accepting both "DDDDD<TAB>word" and bare
// word lines. name identifies the source in error messages.
func ReadWords(r io.Reader, name string) ([]string, error) {
	words := make([]string, 0, 8192)
	s := bufio.NewScanner(r)

	lineNo := 0
	for s.Scan() {
		lineNo++
		word, err := parseWord(s.Text(), lineNo)
		if err != nil {
			return nil, fmt.Errorf("parse wordlist %s: %w", name, err)
		}
		if word == "" {
			continue
		}
		words = append(words, word)
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("scan wordlist %s: %w", name, err)
	}
	if len(words) == 0 {
		return nil, fmt.Errorf("wordlist %s is empty", name)
	}

	return words, nil
}

func parseWord(line string, lineNo int) (string, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", nil
	}

	// Cut before trimming: trimming first turns "11111\t" into the word "11111".
	if code, wordPart, ok := strings.Cut(line, "\t"); ok {
		if strings.ContainsRune(wordPart, '\t') {
			return "", fmt.Errorf("line %d has invalid tab format", lineNo)
		}
		if !isDiceCode(strings.TrimSpace(code)) {
			return "", fmt.Errorf("line %d has invalid dice code", lineNo)
		}
		word := strings.TrimSpace(wordPart)
		if word == "" {
			return "", fmt.Errorf("line %d has empty word", lineNo)
		}
		return word, nil
	}

	if strings.ContainsRune(trimmed, ' ') {
		return "", fmt.Errorf("line %d is not a supported format", lineNo)
	}

	return trimmed, nil
}

func isDiceCode(s string) bool {
	if len(s) != 5 {
		return false
	}
	for _, r := range s {
		if r < '1' || r > '6' {
			return false
		}
	}
	return true
}

// GenerateList returns passphraseCount passphrases, each one wordCount words
// joined by separator.
func GenerateList(words []string, passphraseCount int, wordCount int, separator string) ([]string, error) {
	if passphraseCount <= 0 {
		return nil, errors.New("passphrase count must be > 0")
	}

	out := make([]string, passphraseCount)
	for i := range out {
		phrase, err := Generate(words, wordCount)
		if err != nil {
			return nil, err
		}
		out[i] = strings.Join(phrase, separator)
	}
	return out, nil
}

// Generate returns wordCount words drawn from words uniformly at random, with
// replacement, using crypto/rand.
func Generate(words []string, wordCount int) ([]string, error) {
	if len(words) == 0 {
		return nil, errors.New("no words available")
	}
	if wordCount <= 0 {
		return nil, fmt.Errorf("invalid word count %d", wordCount)
	}

	out := make([]string, wordCount)

	for i := range out {
		n, err := secureRandomIndex(len(words))
		if err != nil {
			return nil, fmt.Errorf("secure random index: %w", err)
		}
		out[i] = words[n]
	}

	return out, nil
}

func secureRandomIndex(limit int) (int, error) {
	if limit <= 0 {
		return 0, errors.New("limit must be > 0")
	}

	n := uint64(limit)
	bound := (math.MaxUint64 / n) * n
	var buf [8]byte

	for {
		if _, err := rand.Read(buf[:]); err != nil {
			return 0, err
		}
		v := binary.LittleEndian.Uint64(buf[:])
		if v < bound {
			return int(v % n), nil
		}
	}
}
