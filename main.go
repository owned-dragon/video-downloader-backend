package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

func generateFileName() string {
	currTime := time.Now().Format("20060102150405")
	id := uuid.New()
	fileName := fmt.Sprintf("%s_%s_video", currTime, id)
	return fileName
}

func runCommand(command_name string, url string) error {

	if err := os.MkdirAll("downloads", 0755); err != nil {
		return nil
	}

	fileName := generateFileName()

	videoPath := filepath.Join("downloads", fileName)

	fmt.Printf("Running: %s %v\n", command_name, url)
	cmdStr := fmt.Sprintf(`yt-dlp -f mp4 -o - "%s" | pv > "%s"`, url, fileName)
	cmd := exec.Command("/bin/sh", "-c", cmdStr)

	stderr, err := cmd.StderrPipe()

	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	bar := progressbar.NewOptions(1000,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()), //you should install "github.com/k0kubun/go-ansi"
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription("[cyan][1/3][reset] Downloading..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	regrexPercentage := regexp.MustCompile(`([0-9]+\.[0.9]+)%`)
	scanner := bufio.NewScanner(stderr)

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		if match := regrexPercentage.FindStringSubmatch(line); len(match) == 2 {
			var percentage float64
			fmt.Sscanf(match[1], "%f", &percentage)
			_ = bar.Set(int(percentage))
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("yt-dlp failed: %w", err)
	}

	_ = bar.Finish()
	fmt.Println("\n✅ MP4 download complete:", videoPath)
	return nil

}

func main() {
	url := os.Args[1]

	if err := runCommand("yt-dlp", url); err != nil {
		fmt.Errorf(err.Error())
	}
}
