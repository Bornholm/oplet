package file

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type Storage struct {
	basePath string
	logger   *slog.Logger
}

type StoredFile struct {
	ID           string
	OriginalName string
	StoredPath   string
	Size         int64
	MimeType     string
	Checksum     string
}

func NewStorage(basePath string, logger *slog.Logger) *Storage {
	return &Storage{
		basePath: basePath,
		logger:   logger.With("component", "file-storage"),
	}
}

func (fs *Storage) GetBasePath() string {
	return fs.basePath
}

func (fs *Storage) StoreInputFile(executionID uint, filename string, reader io.Reader) (*StoredFile, error) {
	return fs.storeFile(executionID, "inputs", filename, reader)
}

func (fs *Storage) StoreOutputFile(executionID uint, filename string, reader io.Reader) (*StoredFile, error) {
	return fs.storeFile(executionID, "outputs", filename, reader)
}

func (fs *Storage) storeFile(executionID uint, subdir, filename string, reader io.Reader) (*StoredFile, error) {
	// Create directory structure
	dirPath := filepath.Join(fs.basePath, "executions", fmt.Sprintf("%d", executionID), subdir)

	if err := os.MkdirAll(dirPath, 0750); err != nil {
		return nil, errors.Wrapf(err, "failed to create directory %s", dirPath)
	}

	// Generate unique filename to avoid conflicts
	storedFilename := fs.generateUniqueFilename(filename)
	storedPath := filepath.Join(dirPath, storedFilename)

	// Create file
	file, err := os.Create(storedPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create file %s", storedPath)
	}
	defer file.Close()

	// Copy with checksum calculation
	hasher := sha256.New()
	multiWriter := io.MultiWriter(file, hasher)

	size, err := io.Copy(multiWriter, reader)
	if err != nil {
		os.Remove(storedPath) // Cleanup on error
		return nil, errors.Wrapf(err, "failed to write file %s", storedPath)
	}

	// Detect MIME type
	mimeType := fs.detectMimeType(storedPath)

	fs.logger.Debug("stored file",
		"execution_id", executionID,
		"original_name", filename,
		"stored_path", storedPath,
		"size", size,
		"mime_type", mimeType)

	return &StoredFile{
		ID:           generateFileID(),
		OriginalName: filename,
		StoredPath:   storedPath,
		Size:         size,
		MimeType:     mimeType,
		Checksum:     fmt.Sprintf("%x", hasher.Sum(nil)),
	}, nil
}

func (fs *Storage) GetFile(filePath string) (io.ReadCloser, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open file %s", filePath)
	}
	return file, nil
}

func (fs *Storage) DeleteExecution(executionID uint) error {
	execPath := filepath.Join(fs.basePath, "executions", fmt.Sprintf("%d", executionID))
	if err := os.RemoveAll(execPath); err != nil {
		return errors.Wrapf(err, "failed to delete execution directory %s", execPath)
	}

	fs.logger.Info("deleted execution directory", "execution_id", executionID, "path", execPath)
	return nil
}

func (fs *Storage) generateUniqueFilename(original string) string {
	ext := filepath.Ext(original)
	base := strings.TrimSuffix(original, ext)
	timestamp := time.Now().Unix()

	// Generate random suffix
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomSuffix := fmt.Sprintf("%x", randomBytes)

	return fmt.Sprintf("%s_%d_%s%s", base, timestamp, randomSuffix, ext)
}

func (fs *Storage) detectMimeType(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return "application/octet-stream"
	}
	defer file.Close()

	// Read first 512 bytes for MIME detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return "application/octet-stream"
	}

	return http.DetectContentType(buffer[:n])
}

func generateFileID() string {
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	return fmt.Sprintf("file_%d_%x", time.Now().Unix(), randomBytes)
}

// GetStorageStats returns storage statistics for an execution
func (fs *Storage) GetStorageStats(executionID uint) (*StorageStats, error) {
	execPath := filepath.Join(fs.basePath, "executions", fmt.Sprintf("%d", executionID))

	var totalSize int64
	var fileCount int

	err := filepath.Walk(execPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	if err != nil {
		return nil, errors.Wrapf(err, "failed to calculate storage stats for execution %d", executionID)
	}

	return &StorageStats{
		TotalSize: totalSize,
		FileCount: fileCount,
	}, nil
}

// EnsureDirectoryExists creates the base directory structure if it doesn't exist
func (fs *Storage) EnsureDirectoryExists() error {
	if err := os.MkdirAll(fs.basePath, 0755); err != nil {
		return errors.Wrapf(err, "failed to create base storage directory %s", fs.basePath)
	}

	// Create subdirectories
	subdirs := []string{"executions", "temp", "config"}
	for _, subdir := range subdirs {
		path := filepath.Join(fs.basePath, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return errors.Wrapf(err, "failed to create storage subdirectory %s", path)
		}
	}

	fs.logger.Info("storage directory structure initialized", "base_path", fs.basePath)
	return nil
}

// CleanupTempFiles removes temporary files older than the specified duration
func (fs *Storage) CleanupTempFiles(olderThan time.Duration) error {
	tempPath := filepath.Join(fs.basePath, "temp")
	cutoff := time.Now().Add(-olderThan)

	return filepath.Walk(tempPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}

		if !info.IsDir() && info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err != nil {
				fs.logger.Warn("failed to remove temp file", "path", path, "error", err)
			} else {
				fs.logger.Debug("removed temp file", "path", path)
			}
		}

		return nil
	})
}

// GetExecutionPath returns the filesystem path for an execution
func (fs *Storage) GetExecutionPath(executionID uint) string {
	return filepath.Join(fs.basePath, "executions", fmt.Sprintf("%d", executionID))
}

// FileExists checks if a file exists at the given path
func (fs *Storage) FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// GetFileInfo returns file information
func (fs *Storage) GetFileInfo(filePath string) (*FileInfo, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get file info for %s", filePath)
	}

	return &FileInfo{
		Name:    info.Name(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}, nil
}

type StorageStats struct {
	TotalSize int64
	FileCount int
}

type FileInfo struct {
	Name    string
	Size    int64
	ModTime time.Time
	IsDir   bool
}
