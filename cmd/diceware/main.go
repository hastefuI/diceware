package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"image/color"
	"io"
	"math"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"charm.land/lipgloss/v2"
)

const defaultWordCount = 6

const bannerArt = `
██████╗ ██╗ ██████╗███████╗██╗    ██╗ █████╗ ██████╗ ███████╗
██╔══██╗██║██╔════╝██╔════╝██║    ██║██╔══██╗██╔══██╗██╔════╝
██║  ██║██║██║     █████╗  ██║ █╗ ██║███████║██████╔╝█████╗  
██║  ██║██║██║     ██╔══╝  ██║███╗██║██╔══██║██╔══██╗██╔══╝  
██████╔╝██║╚██████╗███████╗╚███╔███╔╝██║  ██║██║  ██║███████╗
╚═════╝ ╚═╝ ╚═════╝╚══════╝ ╚══╝╚══╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝
Diceware passphrase generator by hasteful 🐇`

var defaultPhraseFG = lipgloss.Color("255")

type theme struct {
	container lipgloss.Style
	banner    lipgloss.Style
	label     lipgloss.Style
	command   lipgloss.Style
	hint      lipgloss.Style
	index     lipgloss.Style
	phrase    lipgloss.Style
}

func newTheme() theme {
	return theme{
		container: lipgloss.NewStyle().Padding(1, 2),
		banner:    lipgloss.NewStyle().Foreground(lipgloss.Color("105")),
		label:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")),
		command:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
		hint:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		index:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")),
		phrase:    lipgloss.NewStyle().Bold(true),
	}
}

type frame struct {
	banner      string
	command     string
	staticHint  string
	countdown   string
	passphrases []string
	phraseColor color.Color
}

func main() {
	wordlistPath := flag.String("i", "", "path to wordlist file (required)")
	passphraseCount := flag.Int("n", 1, "number of passphrases to generate")
	wordCount := flag.Int("w", defaultWordCount, "number of words per passphrase")
	interval := flag.Duration("interval", 5*time.Second, "refresh interval in live mode")
	once := flag.Bool("once", false, "generate once and exit")
	plain := flag.Bool("plain", false, "print only the passphrases, one per line, and exit (implies -once)")
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 0 {
		fail(fmt.Errorf("unexpected argument: %s", strings.Join(flag.Args(), " ")))
	}
	if strings.TrimSpace(*wordlistPath) == "" {
		fail(errors.New("missing required -i <wordlist-file>"))
	}
	if *passphraseCount <= 0 {
		fail(errors.New("invalid -n value; must be > 0"))
	}
	if *wordCount <= 0 {
		fail(errors.New("invalid -w value; must be > 0"))
	}
	if *interval <= 0 {
		fail(errors.New("invalid -interval value; must be > 0"))
	}

	words, err := loadWords(*wordlistPath)
	if err != nil {
		fail(err)
	}

	if *plain {
		phrases, err := generatePassphraseList(words, *passphraseCount, *wordCount)
		if err != nil {
			fail(err)
		}
		if err := writePlain(os.Stdout, phrases); err != nil {
			fail(err)
		}
		return
	}

	command := buildDisplayCommand(*wordlistPath, *passphraseCount, *wordCount, *interval, *once)
	t := newTheme()
	if *once {
		phrases, err := generatePassphraseList(words, *passphraseCount, *wordCount)
		if err != nil {
			fail(err)
		}
		lipgloss.Println(t.render(frame{
			command:     command,
			countdown:   "Single run mode.",
			passphrases: phrases,
		}))
		return
	}
	fmt.Print("\x1b[H\x1b[2J")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	base := frame{
		banner:     bannerArt,
		command:    command,
		staticHint: staticRefreshHint(*interval),
	}
	if err := runLive(ctx, t, base, words, *passphraseCount, *wordCount, *interval); err != nil {
		fail(err)
	}
}

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  diceware -i <wordlist-file> [-n <count>] [-w <words>] [-interval <duration>] [-once] [-plain]")
	fmt.Fprintln(out, "  diceware -h")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Options:")
	flag.PrintDefaults()
}

func runLive(
	ctx context.Context,
	t theme,
	base frame,
	words []string,
	passphraseCount int,
	wordCount int,
	interval time.Duration,
) error {
	hideCursor()
	defer showCursor()

	current, err := generatePassphraseList(words, passphraseCount, wordCount)
	if err != nil {
		return err
	}
	nextRefresh := time.Now().Add(interval)

	draw := func(countdown string, phrases []string) {
		f := base
		f.countdown = countdown
		f.passphrases = phrases
		renderScreen(t.render(f))
	}
	draw(countdownHint(secondsRemaining(nextRefresh)), current)

	refreshTimer := time.NewTimer(interval)
	defer refreshTimer.Stop()
	countdownTicker := time.NewTicker(1 * time.Second)
	defer countdownTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Print("\n")
			return nil
		case <-countdownTicker.C:
			draw(countdownHint(secondsRemaining(nextRefresh)), current)
		case <-refreshTimer.C:
			nowCountdown := countdownHint(0)
			draw(nowCountdown, current)
			next, err := generatePassphraseList(words, passphraseCount, wordCount)
			if err != nil {
				return err
			}
			transition := base
			transition.countdown = nowCountdown
			transition.passphrases = current
			fadeTransition(t, transition, next)
			current = next
			nextRefresh = time.Now().Add(interval)
			draw(countdownHint(secondsRemaining(nextRefresh)), current)
			refreshTimer.Reset(interval)
		}
	}
}

func generatePassphraseList(words []string, passphraseCount int, wordCount int) ([]string, error) {
	if passphraseCount <= 0 {
		return nil, errors.New("passphrase count must be > 0")
	}

	out := make([]string, passphraseCount)
	for i := range out {
		phrase, err := generatePassphrase(words, wordCount)
		if err != nil {
			return nil, err
		}
		out[i] = strings.Join(phrase, " ")
	}
	return out, nil
}

func writePlain(w io.Writer, phrases []string) error {
	for _, p := range phrases {
		if _, err := fmt.Fprintln(w, p); err != nil {
			return fmt.Errorf("write passphrase: %w", err)
		}
	}
	return nil
}

func fadeTransition(t theme, f frame, next []string) {
	ramp := []color.Color{
		lipgloss.Color("255"),
		lipgloss.Color("252"),
		lipgloss.Color("249"),
		lipgloss.Color("246"),
		lipgloss.Color("243"),
		lipgloss.Color("240"),
	}

	for _, c := range ramp {
		f.phraseColor = c
		renderScreen(t.render(f))
		time.Sleep(70 * time.Millisecond)
	}
	f.passphrases = next
	for _, c := range slices.Backward(ramp) {
		f.phraseColor = c
		renderScreen(t.render(f))
		time.Sleep(70 * time.Millisecond)
	}
}

func renderScreen(rendered string) {
	// Erase to end of line *after* writing each line, never before: a leading
	// erase leaves the row blank until the write lands, and a frame flushed in
	// chunks then shows as a black bar tearing through the output. Erasing
	// afterwards still drops stale characters when a line shortens, but the row
	// is only ever overwritten, never emptied. A full-screen clear would also
	// work but resets terminal selection.
	var b strings.Builder
	b.WriteString("\x1b[H")
	for i, l := range strings.Split(rendered, "\n") {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(l)
		b.WriteString("\x1b[0K")
	}

	// One write, bracketed by synchronized output where the terminal supports
	// it. This alone is not enough: mode 2026 is advisory and a large write can
	// still be split, which is why the erase order above matters.
	fmt.Print("\x1b[?2026h")
	lipgloss.Print(b.String())
	fmt.Print("\x1b[?2026l")
}

func countdownHint(seconds int) string {
	if seconds <= 0 {
		return "Regenerating next set of passphrases now..."
	}
	return fmt.Sprintf("Regenerating next set of passphrases in %ds...", seconds)
}

func staticRefreshHint(interval time.Duration) string {
	return fmt.Sprintf("Interactive mode. Passphrases will rotate every %s. Press Ctrl+C to exit.", interval)
}

func buildDisplayCommand(wordlistPath string, passphraseCount int, wordCount int, interval time.Duration, once bool) string {
	command := fmt.Sprintf(
		"go run ./cmd/diceware -i %s -n %d -w %d -interval %s",
		wordlistPath,
		passphraseCount,
		wordCount,
		interval,
	)
	if once {
		command += " -once"
	}
	return command
}

func secondsRemaining(next time.Time) int {
	remaining := int(math.Ceil(time.Until(next).Seconds()))
	if remaining < 0 {
		return 0
	}
	return remaining
}

func hideCursor() {
	fmt.Print("\x1b[?25l")
}

func showCursor() {
	fmt.Print("\x1b[?25h")
}

func (t theme) render(f frame) string {
	lines := make([]string, 0, len(f.passphrases)+8)
	if f.banner != "" {
		lines = append(lines, t.banner.Render(f.banner), "")
	}
	lines = append(lines, t.label.Render("Command"), t.command.Render("$ "+f.command))
	if f.staticHint != "" {
		lines = append(lines, t.hint.Render(f.staticHint))
	}
	lines = append(lines, t.hint.Render(f.countdown), "", t.label.Render("Passphrases"))

	fg := f.phraseColor
	if fg == nil {
		fg = defaultPhraseFG
	}
	styled := t.phrase.Foreground(fg)
	for i, p := range f.passphrases {
		line := fmt.Sprintf("%s %s", t.index.Render(fmt.Sprintf("0x%02X", i+1)), styled.Render(p))
		lines = append(lines, line)
	}

	return t.container.Render(strings.Join(lines, "\n"))
}

func loadWords(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open wordlist %q: %w", path, err)
	}
	defer f.Close()

	words := make([]string, 0, 8192)
	s := bufio.NewScanner(f)

	lineNo := 0
	for s.Scan() {
		lineNo++
		word, err := parseWord(s.Text(), lineNo)
		if err != nil {
			return nil, fmt.Errorf("parse wordlist %q: %w", path, err)
		}
		if word == "" {
			continue
		}
		words = append(words, word)
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("scan wordlist %q: %w", path, err)
	}
	if len(words) == 0 {
		return nil, errors.New("wordlist is empty")
	}

	return words, nil
}

func parseWord(line string, lineNo int) (string, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", nil
	}

	if code, wordPart, ok := strings.Cut(trimmed, "\t"); ok {
		if strings.ContainsRune(wordPart, '\t') {
			return "", fmt.Errorf("line %d has invalid tab format", lineNo)
		}
		if !isDiceCode(code) {
			return "", fmt.Errorf("line %d has invalid dice code", lineNo)
		}
		word := strings.TrimSpace(wordPart)
		if word == "" {
			return "", fmt.Errorf("line %d has empty word", lineNo)
		}
		return word, nil
	}

	if strings.ContainsAny(trimmed, " \t") {
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

func generatePassphrase(words []string, count int) ([]string, error) {
	if len(words) == 0 {
		return nil, errors.New("no words available")
	}
	if count <= 0 {
		return nil, fmt.Errorf("invalid word count %d", count)
	}

	out := make([]string, count)

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

func fail(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
