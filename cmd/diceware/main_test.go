package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Opening entries of the EFF long wordlist. See https://www.eff.org/dice.
var effEntries = []struct {
	code string
	word string
}{
	{code: "11111", word: "abacus"},
	{code: "11112", word: "abdomen"},
	{code: "11113", word: "abdominal"},
	{code: "11114", word: "abide"},
	{code: "11115", word: "abiding"},
	{code: "11116", word: "ability"},
	{code: "11121", word: "ablaze"},
	{code: "11122", word: "able"},
	{code: "11123", word: "abnormal"},
	{code: "11124", word: "abrasion"},
	{code: "11125", word: "abrasive"},
	{code: "11126", word: "abreast"},
	{code: "11131", word: "abridge"},
	{code: "11132", word: "abroad"},
	{code: "11133", word: "abruptly"},
	{code: "11134", word: "absence"},
}

// Six words, the length EFF recommends.
const (
	effPassphrase     = "abacus abdomen abdominal abide abiding ability"
	effNextPassphrase = "ablaze able abnormal abrasion abrasive abreast"
)

func effWords() []string {
	words := make([]string, len(effEntries))
	for i, e := range effEntries {
		words[i] = e.word
	}
	return words
}

func effWordlist() string {
	var b strings.Builder
	for _, e := range effEntries {
		fmt.Fprintf(&b, "%s\t%s\n", e.code, e.word)
	}
	return b.String()
}

func writeEFFWordlist(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "eff_wordlist.txt")
	if err := os.WriteFile(path, []byte(effWordlist()), 0o600); err != nil {
		t.Fatalf("write wordlist: %v", err)
	}
	return path
}

func TestLoadWordsDropsDiceCodes(t *testing.T) {
	got, err := loadWords(writeEFFWordlist(t))
	if err != nil {
		t.Fatalf("loadWords: %v", err)
	}
	if want := effWords(); !slices.Equal(got, want) {
		t.Errorf("loadWords returned %q, want %q", got, want)
	}
}

func TestReadWords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "dice codes", input: effWordlist(), want: effWords()},
		{name: "bare words", input: "abacus\nabdomen\nabide\n", want: []string{"abacus", "abdomen", "abide"}},
		{name: "blank lines skipped", input: "abacus\n\n   \nabdomen\n", want: []string{"abacus", "abdomen"}},
		{name: "whitespace-only line with tab skipped", input: "abacus\n \t \nabdomen\n", want: []string{"abacus", "abdomen"}},
		{name: "indented dice code", input: "  11111\tabacus\n", want: []string{"abacus"}},
		{name: "no trailing newline", input: "11111\tabacus", want: []string{"abacus"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readWords(strings.NewReader(tt.input), stdinName)
			if err != nil {
				t.Fatalf("readWords: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("readWords returned %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadWordsErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty input", input: ""},
		{name: "blank lines only", input: "\n   \n\n"},
		{name: "dice code out of range", input: "11111\tabacus\n11117\tabdomen\n"},
		{name: "dice code wrong length", input: "1111\tabacus\n"},
		{name: "second tab", input: "11111\tabacus\textra\n"},
		{name: "space separated", input: "11111 abacus\n"},
		{name: "dice code with no word", input: "11111\t\n"},
		{name: "empty word after code", input: "11111\t   \n"},
		{name: "bare word with trailing tab", input: "abacus\t\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readWords(strings.NewReader(tt.input), stdinName)
			if err == nil {
				t.Fatalf("readWords(%q) = %q, want error", tt.input, got)
			}
			if !strings.Contains(err.Error(), stdinName) {
				t.Errorf("readWords error = %v, want one naming %q", err, stdinName)
			}
		})
	}
}

func TestReadWordsScanError(t *testing.T) {
	sentinel := errors.New("boom")
	_, err := readWords(failingReader{err: sentinel}, stdinName)
	if !errors.Is(err, sentinel) {
		t.Fatalf("readWords error = %v, want one wrapping %v", err, sentinel)
	}
}

func FuzzParseWord(f *testing.F) {
	for _, e := range effEntries {
		f.Add(e.code + "\t" + e.word)
	}
	for _, seed := range []string{
		"", "   ", "\t", " \t ",
		"abacus", "abacus\t", "  11111\tabacus",
		"11111\t", "11111\t   ", "11111 abacus", "11111\tabacus\textra",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, line string) {
		word, err := parseWord(line, 1)
		if err != nil {
			if word != "" {
				t.Errorf("parseWord(%q) = %q with error %v, want no word alongside an error", line, word, err)
			}
			return
		}
		if word == "" {
			return
		}
		if word != strings.TrimSpace(word) {
			t.Errorf("parseWord(%q) = %q, which is not trimmed", line, word)
		}
		if strings.ContainsRune(word, '\t') {
			t.Errorf("parseWord(%q) = %q, which contains a tab", line, word)
		}
		if _, after, ok := strings.Cut(line, "\t"); ok {
			if want := strings.TrimSpace(after); word != want {
				t.Errorf("parseWord(%q) = %q, want the text after the tab, %q", line, word, want)
			}
		}
	})
}

func TestCheckPipedStdin(t *testing.T) {
	tests := []struct {
		name    string
		mode    os.FileMode
		wantErr bool
	}{
		{name: "pipe", mode: os.ModeNamedPipe},
		{name: "redirected file", mode: 0o644},
		{name: "socket", mode: os.ModeSocket},
		{name: "terminal", mode: os.ModeCharDevice | os.ModeDevice | 0o620, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkPipedStdin(tt.mode)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("checkPipedStdin(%v) = nil, want error", tt.mode)
				}
				return
			}
			if err != nil {
				t.Errorf("checkPipedStdin(%v) = %v, want nil", tt.mode, err)
			}
		})
	}
}

func TestGeneratePassphraseList(t *testing.T) {
	words := effWords()

	tests := []struct {
		name            string
		passphraseCount int
		wordCount       int
		wantErr         bool
	}{
		{name: "classic five words", passphraseCount: 1, wordCount: 5},
		{name: "recommended six words", passphraseCount: 1, wordCount: defaultWordCount},
		{name: "high security seven words", passphraseCount: 3, wordCount: 7},
		{name: "zero passphrases", passphraseCount: 0, wordCount: defaultWordCount, wantErr: true},
		{name: "zero words", passphraseCount: 1, wordCount: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generatePassphraseList(words, tt.passphraseCount, tt.wordCount, defaultSeparator)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("generatePassphraseList(%d, %d) = %q, want error", tt.passphraseCount, tt.wordCount, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("generatePassphraseList: %v", err)
			}
			if len(got) != tt.passphraseCount {
				t.Fatalf("got %d passphrases, want %d", len(got), tt.passphraseCount)
			}
			for _, phrase := range got {
				fields := strings.Fields(phrase)
				if len(fields) != tt.wordCount {
					t.Errorf("phrase %q has %d words, want %d", phrase, len(fields), tt.wordCount)
				}
				for _, w := range fields {
					if !slices.Contains(words, w) {
						t.Errorf("phrase %q contains %q, which is not in the wordlist", phrase, w)
					}
				}
			}
		})
	}
}

func TestGeneratePassphraseListSeparator(t *testing.T) {
	words := effWords()

	separators := []struct {
		name      string
		separator string
	}{
		{name: "space", separator: defaultSeparator},
		{name: "hyphen", separator: "-"},
		{name: "dot", separator: "."},
		{name: "multiple characters", separator: " :: "},
		{name: "empty", separator: ""},
	}

	for _, tt := range separators {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generatePassphraseList(words, 1, defaultWordCount, tt.separator)
			if err != nil {
				t.Fatalf("generatePassphraseList: %v", err)
			}

			if tt.separator == "" {
				if strings.ContainsAny(got[0], " -.:") {
					t.Errorf("phrase %q holds a separator, want the words joined directly", got[0])
				}
				return
			}

			parts := strings.Split(got[0], tt.separator)
			if len(parts) != defaultWordCount {
				t.Fatalf("phrase %q split into %d parts, want %d", got[0], len(parts), defaultWordCount)
			}
			for _, w := range parts {
				if !slices.Contains(words, w) {
					t.Errorf("phrase %q contains %q, which is not in the wordlist", got[0], w)
				}
			}
		})
	}
}

func TestCheckSeparator(t *testing.T) {
	tests := []struct {
		name      string
		separator string
		wantErr   bool
	}{
		{name: "space", separator: defaultSeparator},
		{name: "hyphen", separator: "-"},
		{name: "empty", separator: ""},
		{name: "tab", separator: "\t"},
		{name: "newline", separator: "\n", wantErr: true},
		{name: "carriage return", separator: "\r", wantErr: true},
		{name: "newline inside a longer separator", separator: " |\n| ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSeparator(tt.separator)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("checkSeparator(%q) = nil, want error", tt.separator)
				}
				return
			}
			if err != nil {
				t.Errorf("checkSeparator(%q) = %v, want nil", tt.separator, err)
			}
		})
	}
}

func TestWritePlain(t *testing.T) {
	tests := []struct {
		name     string
		phrases  []string
		expected string
	}{
		{name: "none", phrases: nil, expected: ""},
		{
			name:     "one passphrase",
			phrases:  []string{effPassphrase},
			expected: effPassphrase + "\n",
		},
		{
			name:     "several passphrases",
			phrases:  []string{effPassphrase, effNextPassphrase},
			expected: effPassphrase + "\n" + effNextPassphrase + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			if err := writePlain(&b, tt.phrases); err != nil {
				t.Fatalf("writePlain: %v", err)
			}
			if got := b.String(); got != tt.expected {
				t.Errorf("writePlain wrote %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestWritePlainError(t *testing.T) {
	sentinel := errors.New("boom")
	err := writePlain(failingWriter{err: sentinel}, []string{effPassphrase})
	if !errors.Is(err, sentinel) {
		t.Fatalf("writePlain error = %v, want one wrapping %v", err, sentinel)
	}
}

type failingWriter struct {
	err error
}

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) {
	return 0, r.err
}
