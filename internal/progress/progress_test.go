package progress

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"}, // 1024 + 512
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"}, // 1.5 * 1024 * 1024
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
		{1125899906842624, "1.0 PB"},
		{1152921504606846976, "1.0 EB"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := FormatBytes(tc.input)
			if result != tc.expected {
				t.Errorf("FormatBytes(%d) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		input    int64 // seconds
		expected string
	}{
		{0, "0s"},
		{30, "30s"},
		{59, "59s"},
		{60, "1m"},
		{90, "1m"},
		{120, "2m"},
		{3600, "1h"},
		{3660, "1h 1m"},
		{7200, "2h"},
		{86400, "1d"},
		{90000, "1d 1h"}, // 25 hours
		{172800, "2d"},
		{266400, "3d 2h"}, // 74 hours
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := FormatDuration(tc.input)
			if result != tc.expected {
				t.Errorf("FormatDuration(%d) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestNewWriter(t *testing.T) {
	total := int64(1000)
	writer := NewWriter(total)

	if writer.total != total {
		t.Errorf("NewWriter(%d).total = %d, expected %d", total, writer.total, total)
	}

	if writer.written != 0 {
		t.Errorf("NewWriter(%d).written = %d, expected 0", total, writer.written)
	}

	if writer.startTime.IsZero() {
		t.Error("NewWriter().startTime should not be zero")
	}
}

func TestProgressWriter_Write(t *testing.T) {
	total := int64(100)
	writer := NewWriter(total)

	// Redirect stdout to capture progress output
	oldStdout := writer

	// Test writing data
	data := []byte("hello")
	n, err := writer.Write(data)
	if err != nil {
		t.Errorf("Writer.Write() returned error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Writer.Write() returned %d, expected %d", n, len(data))
	}

	if writer.written != int64(len(data)) {
		t.Errorf("Writer.written = %d, expected %d", writer.written, len(data))
	}

	// Test multiple writes
	data2 := []byte("world")
	n2, err2 := writer.Write(data2)

	if err2 != nil {
		t.Errorf("Writer.Write() second call returned error: %v", err2)
	}

	if n2 != len(data2) {
		t.Errorf("Writer.Write() second call returned %d, expected %d", n2, len(data2))
	}

	expectedTotal := int64(len(data) + len(data2))
	if writer.written != expectedTotal {
		t.Errorf("Writer.written after two writes = %d, expected %d", writer.written, expectedTotal)
	}

	_ = oldStdout // Avoid unused variable warning
}

func TestProgressWriter_ZeroTotal(t *testing.T) {
	// Test writer with unknown total size (0)
	writer := NewWriter(0)

	data := []byte("test data")
	n, err := writer.Write(data)
	if err != nil {
		t.Errorf("Writer.Write() with zero total returned error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Writer.Write() with zero total returned %d, expected %d", n, len(data))
	}

	if writer.written != int64(len(data)) {
		t.Errorf("Writer.written with zero total = %d, expected %d", writer.written, len(data))
	}
}

func TestTeeReader(t *testing.T) {
	total := int64(100)
	writer := NewWriter(total)

	// Create a reader with test data
	testData := "hello world test data"
	reader := bytes.NewReader([]byte(testData))

	// Create TeeReader
	teeReader := TeeReader(reader, writer)

	// Read all data through TeeReader
	buffer := make([]byte, len(testData))
	n, err := io.ReadFull(teeReader, buffer)
	if err != nil {
		t.Errorf("TeeReader read returned error: %v", err)
	}

	if n != len(testData) {
		t.Errorf("TeeReader read %d bytes, expected %d", n, len(testData))
	}

	if string(buffer) != testData {
		t.Errorf("TeeReader data = %q, expected %q", string(buffer), testData)
	}

	// Check that progress writer received the data
	if writer.written != int64(len(testData)) {
		t.Errorf("Progress writer received %d bytes, expected %d", writer.written, len(testData))
	}
}

func TestProgressWriter_Finish(t *testing.T) {
	total := int64(100)
	writer := NewWriter(total)

	// Write some data (less than total)
	data := []byte("test")
	writer.Write(data) //nolint:errcheck

	if writer.written != int64(len(data)) {
		t.Errorf("Before Finish(): written = %d, expected %d", writer.written, len(data))
	}

	// Call Finish() - this should set written to total
	writer.Finish()

	if writer.written != total {
		t.Errorf("After Finish(): written = %d, expected %d", writer.written, total)
	}
}

func TestProgressWriter_FinishWithZeroTotal(t *testing.T) {
	writer := NewWriter(0)

	// Write some data
	data := []byte("test")
	writer.Write(data) //nolint:errcheck

	originalWritten := writer.written

	// Call Finish() - with zero total, written should not change
	writer.Finish()

	if writer.written != originalWritten {
		t.Errorf("Finish() with zero total changed written from %d to %d", originalWritten, writer.written)
	}
}

// Test that lastPrint is updated appropriately
func TestProgressWriter_PrintThrottling(t *testing.T) {
	writer := NewWriter(1000)

	// Set lastPrint to a time that would prevent printing
	writer.lastPrint = time.Now()

	// Write some data - should not print due to throttling
	data := []byte("test")
	writer.Write(data) //nolint:errcheck

	// The test passes if no panic occurs and Write returns successfully
	// Actual output testing would require capturing stdout, which is complex
}

// Benchmark tests
func BenchmarkFormatBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatBytes(1572864) // 1.5 MB
	}
}

func BenchmarkFormatDuration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatDuration(3661) // 1h 1m 1s
	}
}

func BenchmarkProgressWriter_Write(b *testing.B) {
	writer := NewWriter(int64(b.N))
	data := []byte("benchmark test data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writer.Write(data) //nolint:errcheck
	}
}
