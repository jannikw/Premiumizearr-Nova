package progress_downloader

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

// WriteCounter counts the number of bytes written and tracks download progress.
type WriteCounter struct {
	StartTime  time.Time
	LastUpdate time.Time
	LastAmount uint64
	Total      uint64
}

func NewWriteCounter() *WriteCounter {
	return &WriteCounter{
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
		LastAmount: 0,
		Total:      0,
	}
}

func (wc *WriteCounter) GetSpeed() string {
	timeSince := time.Since(wc.StartTime).Seconds()

	if timeSince == 0 || timeSince < 1 {
		return "0 B/s"
	}

	return fmt.Sprintf("%s/s", humanize.Bytes(wc.Total/uint64(timeSince)))
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.LastAmount = wc.Total
	wc.Total += uint64(n)
	return n, nil
}

func (wc WriteCounter) GetProgress() string {
	return fmt.Sprintf("%s complete", humanize.Bytes(wc.Total))
}

// DownloadFile uses wget for downloading and updates WriteCounter for progress tracking.
func DownloadFile(url string, filepath string, counter *WriteCounter) error {
	// Prepare the wget command with progress in bytes
	cmd := exec.Command("wget", "-c", "--progress=dot:mega", "-O", filepath, url)

	// Get a pipe for the command's output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start wget: %w", err)
	}

	// Use a regex to parse progress from wget's output
	progressRegex := regexp.MustCompile(`\d+K`) // Matches progress like "1024K", "2048K"
	scanner := bufio.NewScanner(stdout)

	// Track progress
	for scanner.Scan() {
		line := scanner.Text()
		matches := progressRegex.FindAllString(line, -1)
		for _, match := range matches {
			// Convert progress to bytes and update WriteCounter
			sizeStr := strings.TrimSuffix(match, "K")
			sizeInBytes, _ := strconv.ParseUint(sizeStr, 10, 64)
			counter.Total = sizeInBytes * 1024 // K to bytes

			// Print progress
			fmt.Printf("\r%s complete (%s/s)", counter.GetProgress(), counter.GetSpeed())
		}
	}

	// Wait for wget to finish
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("wget command failed: %w", err)
	}

	fmt.Println("\nDownload completed successfully.")
	return nil
}
