package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <url>\n", os.Args[0])
		os.Exit(1)
	}
	url := strings.TrimSpace(os.Args[1])
	if url == "" {
		fmt.Fprintln(os.Stderr, "error: URL must not be empty")
		os.Exit(1)
	}

<<<<<<< HEAD
	fileName := generateFileName()

	videoPath := filepath.Join("downloads", fileName)

	fmt.Printf("Running: %s %v\n", command_name, url)
	cmdStr := fmt.Sprintf(`yt-dlp -f mp4 -o - "%s" | pv > "%s"`, url, fileName)
	cmd := exec.Command("/bin/sh", "-c", cmdStr)

	stderr, err := cmd.StderrPipe()
=======
	downloadsDir := "downloads"
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: creating downloads directory %q: %v\n", downloadsDir, err)
		os.Exit(1)
	}
>>>>>>> cb976f38bd965329706112b5d86fdff801f61a26

	ytdlpPath, err := exec.LookPath("yt-dlp")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: yt-dlp not found in PATH. Install it from https://github.com/yt-dlp/yt-dlp and ensure it's on PATH.")
		os.Exit(1)
	}

	ctx := context.Background()

	ytdlpArgs := []string{
		"--progress",
		"--newline",
		"-P", downloadsDir,
		"-o", "%(title)s.%(ext)s",
		url,
	}

	cmd := exec.CommandContext(ctx, ytdlpPath, ytdlpArgs...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1", "PYTHONIOENCODING=UTF-8")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: creating stdout pipe:", err)
		os.Exit(1)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: creating stderr pipe:", err)
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "error: starting yt-dlp:", err)
		os.Exit(1)
	}

	percentRe := regexp.MustCompile(`(\d{1,3}(?:\.\d+)?)%`)

	updates := make(chan float64, 64)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		readStream("stderr", stderr, updates, percentRe)
	}()
	go func() {
		defer wg.Done()
		readStream("stdout", stdout, updates, percentRe)
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		var last float64 = -1
		for p := range updates {
			if p < 0 {
				p = 0
			}
			if p > 100 {
				p = 100
			}
			if last < 0 || abs(p-last) >= 0.1 || p == 100 {
				drawProgressBar(os.Stderr, p)
				last = p
			}
		}
	}()

	waitErr := cmd.Wait()
	wg.Wait()
	close(updates)
	<-done

	if waitErr == nil {
		drawProgressBar(os.Stderr, 100)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Download complete. Files saved to %q\n", downloadsDir)
	} else {
		fmt.Fprint(os.Stderr, "\r")
		fmt.Fprint(os.Stderr, clearLine())
		fmt.Fprintln(os.Stderr, "yt-dlp failed:", unwrapExitErr(waitErr))
		os.Exit(1)
	}
}

func readStream(_ string, r io.Reader, updates chan<- float64, percentRe *regexp.Regexp) {
	sc := bufio.NewScanner(r)
	sc.Split(splitOnCRorLF)
	const maxLine = 1024 * 1024
	sc.Buffer(make([]byte, 0, 64*1024), maxLine)

	for sc.Scan() {
		line := sc.Text()
		if m := percentRe.FindStringSubmatch(line); len(m) == 2 {
			p := parsePercent(m[1])
			select {
			case updates <- p:
			default:
			}
		}
	}
}

func parsePercent(s string) float64 {
	var p float64
	fmt.Sscanf(s, "%f", &p)
	return p
}

func drawProgressBar(w io.Writer, percent float64) {
	const width = 40
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := int((percent / 100.0) * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("=", filled) + strings.Repeat(" ", width-filled)
	fmt.Fprintf(w, "\r[%s] %6.2f%%", bar, percent)
}

func clearLine() string {
	return "\x1b[2K"
}

func unwrapExitErr(err error) error {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return fmt.Errorf("%v", ee)
	}
	return err
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func splitOnCRorLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i, b := range data {
		if b == '\n' || b == '\r' {
			return i + 1, dropTrailingCRLF(data[:i]), nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), dropTrailingCRLF(data), nil
	}
	return 0, nil, nil
}

func dropTrailingCRLF(b []byte) []byte {
	n := len(b)
	for n > 0 && (b[n-1] == '\n' || b[n-1] == '\r') {
		n--
	}
	return b[:n]
}