package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// Six words, the length EFF recommends.
const (
	effPassphrase     = "abacus abdomen abdominal abide abiding ability"
	effNextPassphrase = "ablaze able abnormal abrasion abrasive abreast"
)

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

func TestIsTerminalMode(t *testing.T) {
	tests := []struct {
		name string
		mode os.FileMode
		want bool
	}{
		{name: "pipe", mode: os.ModeNamedPipe},
		{name: "redirected file", mode: 0o644},
		{name: "socket", mode: os.ModeSocket},
		{name: "terminal", mode: os.ModeCharDevice | os.ModeDevice | 0o620, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTerminalMode(tt.mode); got != tt.want {
				t.Errorf("isTerminalMode(%v) = %t, want %t", tt.mode, got, tt.want)
			}
		})
	}
}

func TestIsTerminalPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	t.Cleanup(func() {
		r.Close()
		w.Close()
	})

	if isTerminal(w) {
		t.Error("isTerminal(pipe) = true, want false")
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
