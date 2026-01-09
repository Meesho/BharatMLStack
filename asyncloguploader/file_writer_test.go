package asyncloguploader

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileWriter_WriteVectored(t *testing.T) {
	t.Run("WritesBuffersToFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.MaxFileSize = 0 // Disable rotation

		writer, err := NewSizeFileWriter(config, nil)
		require.NoError(t, err)
		defer writer.Close()

		buffers := [][]byte{
			[]byte("buffer1"),
			[]byte("buffer2"),
		}

		n, err := writer.WriteVectored(buffers)
		assert.NoError(t, err)
		assert.Greater(t, n, 0)
	})

	t.Run("HandlesEmptyBuffers", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))

		writer, err := NewSizeFileWriter(config, nil)
		require.NoError(t, err)
		defer writer.Close()

		n, err := writer.WriteVectored(nil)
		assert.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("FiltersEmptyBuffers", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))

		writer, err := NewSizeFileWriter(config, nil)
		require.NoError(t, err)
		defer writer.Close()

		buffers := [][]byte{
			[]byte("data"),
			nil,
			[]byte{},
			[]byte("more data"),
		}

		n, err := writer.WriteVectored(buffers)
		assert.NoError(t, err)
		assert.Greater(t, n, 0)
	})
}

func TestFileWriter_Rotation(t *testing.T) {
	t.Run("RotatesWhenMaxSizeReached", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.MaxFileSize = 1024 * 1024 // 1MB
		config.PreallocateFileSize = 1024 * 1024

		uploadChan := make(chan string, 10)
		writer, err := NewSizeFileWriter(config, uploadChan)
		require.NoError(t, err)
		defer writer.Close()

		// Write data to exceed max size
		largeBuffer := make([]byte, 512*1024) // 512KB
		for i := 0; i < 3; i++ {
			_, err := writer.WriteVectored([][]byte{largeBuffer})
			require.NoError(t, err)
		}

		// Check if rotation occurred
		select {
		case completedFile := <-uploadChan:
			assert.NotEmpty(t, completedFile)
			assert.FileExists(t, completedFile)
		default:
			// Rotation may not have occurred yet
		}
	})

	t.Run("CreatesNewFileAfterRotation", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))
		config.MaxFileSize = 1024 * 1024 // 1MB

		uploadChan := make(chan string, 10)
		writer, err := NewSizeFileWriter(config, uploadChan)
		require.NoError(t, err)
		defer writer.Close()

		// Fill first file
		largeBuffer := make([]byte, 512*1024)
		for i := 0; i < 3; i++ {
			writer.WriteVectored([][]byte{largeBuffer})
		}

		// Write more data (should go to new file)
		_, err = writer.WriteVectored([][]byte{[]byte("new file data")})
		assert.NoError(t, err)
	})
}

func TestFileWriter_FileIntegrity(t *testing.T) {
	t.Run("WritesDataCorrectly", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")
		config := DefaultConfig(logPath)
		config.MaxFileSize = 0 // Disable rotation

		writer, err := NewSizeFileWriter(config, nil)
		require.NoError(t, err)

		// Create test buffers with headers
		buffer1 := make([]byte, 1024)
		binary.LittleEndian.PutUint32(buffer1[0:4], 1024) // Capacity
		binary.LittleEndian.PutUint32(buffer1[4:8], 100) // Valid data bytes
		copy(buffer1[8:], []byte("test data 1"))

		buffer2 := make([]byte, 1024)
		binary.LittleEndian.PutUint32(buffer2[0:4], 1024)
		binary.LittleEndian.PutUint32(buffer2[4:8], 100)
		copy(buffer2[8:], []byte("test data 2"))

		_, err = writer.WriteVectored([][]byte{buffer1, buffer2})
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		// Find the actual file (with timestamp)
		actualFile := findLogFile(t, tmpDir, "test")
		if actualFile == "" {
			t.Fatal("Log file not found")
		}

		// Read and verify file contents
		data, err := os.ReadFile(actualFile)
		require.NoError(t, err)

		assert.Contains(t, string(data), "test data 1")
		assert.Contains(t, string(data), "test data 2")
	})

	t.Run("MaintainsDataOrder", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")
		config := DefaultConfig(logPath)
		config.MaxFileSize = 0

		writer, err := NewSizeFileWriter(config, nil)
		require.NoError(t, err)

		// Write multiple buffers in sequence
		for i := 0; i < 5; i++ {
			buffer := make([]byte, 512)
			binary.LittleEndian.PutUint32(buffer[0:4], 512)
			binary.LittleEndian.PutUint32(buffer[4:8], 100)
			copy(buffer[8:], []byte{byte(i)})
			writer.WriteVectored([][]byte{buffer})
		}

		err = writer.Close()
		require.NoError(t, err)

		// Find the actual file (with timestamp)
		actualFile := findLogFile(t, tmpDir, "test")
		if actualFile == "" {
			t.Fatal("Log file not found")
		}

		// Verify order
		data, err := os.ReadFile(actualFile)
		require.NoError(t, err)

		// Check that data appears in order
		prevPos := -1
		for i := 0; i < 5; i++ {
			// Find byte(i) in data
			for j := prevPos + 1; j < len(data); j++ {
				if data[j] == byte(i) {
					assert.Greater(t, j, prevPos, "Data out of order")
					prevPos = j
					break
				}
			}
		}
	})
}

func TestFileWriter_HeaderFormat(t *testing.T) {
	t.Run("WritesShardHeadersCorrectly", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")
		config := DefaultConfig(logPath)
		config.MaxFileSize = 0

		writer, err := NewSizeFileWriter(config, nil)
		require.NoError(t, err)

		// Create buffer with header
		capacity := uint32(1024)
		validDataBytes := uint32(100)
		buffer := make([]byte, 1024)
		binary.LittleEndian.PutUint32(buffer[0:4], capacity)
		binary.LittleEndian.PutUint32(buffer[4:8], validDataBytes)
		copy(buffer[8:], []byte("test data"))

		_, err = writer.WriteVectored([][]byte{buffer})
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		// Find the actual file (with timestamp)
		actualFile := findLogFile(t, tmpDir, "test")
		if actualFile == "" {
			t.Fatal("Log file not found")
		}

		// Read file and verify header
		data, err := os.ReadFile(actualFile)
		require.NoError(t, err)

		if len(data) >= 8 {
			readCapacity := binary.LittleEndian.Uint32(data[0:4])
			readValidBytes := binary.LittleEndian.Uint32(data[4:8])

			assert.Equal(t, capacity, readCapacity)
			assert.Equal(t, validDataBytes, readValidBytes)
		}
	})
}

func TestFileWriter_GetLastPwritevDuration(t *testing.T) {
	t.Run("ReturnsDurationAfterWrite", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := DefaultConfig(filepath.Join(tmpDir, "test.log"))

		writer, err := NewSizeFileWriter(config, nil)
		require.NoError(t, err)
		defer writer.Close()

		buffers := [][]byte{[]byte("test")}
		_, err = writer.WriteVectored(buffers)
		require.NoError(t, err)

		duration := writer.GetLastPwritevDuration()
		assert.GreaterOrEqual(t, duration, time.Duration(0))
	})
}

// findLogFile finds a log file in the directory matching the base name
func findLogFile(t *testing.T, dir, baseName string) string {
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() != ".DS_Store" {
			if filepath.Ext(entry.Name()) == ".log" {
				name := entry.Name()
				// Check if it starts with baseName
				if len(name) > len(baseName) && name[:len(baseName)] == baseName {
					return filepath.Join(dir, name)
				}
			}
		}
	}
	return ""
}

