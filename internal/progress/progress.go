package progress

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Writer implements io.Writer to show download progress
type Writer struct {
	total     int64
	written   int64
	startTime time.Time
	lastPrint time.Time
}

// NewWriter creates a new progress writer
func NewWriter(total int64) *Writer {
	return &Writer{
		total:     total,
		written:   0,
		startTime: time.Now(),
	}
}

func (pw *Writer) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)

	// Update progress every 500ms or at completion
	now := time.Now()
	if now.Sub(pw.lastPrint) >= 500*time.Millisecond || pw.written >= pw.total {
		pw.printProgress()
		pw.lastPrint = now
	}

	return n, nil
}

func (pw *Writer) printProgress() {
	if pw.total <= 0 {
		fmt.Printf("\r📥 Downloaded: %s", FormatBytes(pw.written))
		return
	}

	percentage := float64(pw.written) / float64(pw.total) * 100
	elapsed := time.Since(pw.startTime)

	var speedStr string
	var etaStr string

	if elapsed > 0 {
		bytesPerSec := float64(pw.written) / elapsed.Seconds()
		speedStr = fmt.Sprintf(" • %s/s", FormatBytes(int64(bytesPerSec)))

		if bytesPerSec > 0 && pw.written < pw.total {
			remainingBytes := pw.total - pw.written
			eta := time.Duration(float64(remainingBytes)/bytesPerSec) * time.Second
			etaStr = fmt.Sprintf(" • ETA: %s", FormatDuration(int64(eta.Seconds())))
		}
	}

	// Progress bar (40 chars wide)
	barWidth := 40
	filled := int(percentage / 100.0 * float64(barWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	fmt.Printf("\r📥 [%s] %.1f%% (%s / %s)%s%s",
		bar, percentage,
		FormatBytes(pw.written), FormatBytes(pw.total),
		speedStr, etaStr)
}

// Finish completes the progress display
func (pw *Writer) Finish() {
	if pw.total > 0 {
		pw.written = pw.total // Ensure 100% is shown
	}
	pw.printProgress()
	fmt.Printf("\n")
}

// TeeReader creates a TeeReader with progress display
func TeeReader(r io.Reader, pw *Writer) io.Reader {
	return io.TeeReader(r, pw)
}

// FormatBytes converts bytes to human readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration converts seconds to human readable duration
func FormatDuration(seconds int64) string {
	duration := time.Duration(seconds) * time.Second

	days := int64(duration.Hours()) / 24
	hours := int64(duration.Hours()) % 24
	minutes := int64(duration.Minutes()) % 60

	if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	} else if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}
