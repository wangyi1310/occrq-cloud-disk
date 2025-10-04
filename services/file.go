package services

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileService interface {
	UploadFile(filename string, content io.Reader) error
	DownloadFile(filename string) (io.ReadCloser, error)
	ListFiles() ([]string, error)
	DeleteFile(filename string) error
}

// LocalFileService 本地文件系统实现
type LocalFileService struct {
	uploadDir string // 上传文件存储目录
	maxSize   int64  // 最大文件大小限制，单位字节
}

// NewLocalFileService 创建新的本地文件服务实例
func NewLocalFileService(uploadDir string, maxSize int64) (*LocalFileService, error) {
	// 确保上传目录存在
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, err
	}

	return &LocalFileService{
		uploadDir: uploadDir,
		maxSize:   maxSize,
	}, nil
}

// UploadFile 上传文件
func (s *LocalFileService) UploadFile(filename string, content io.Reader) error {
	if filename == "" {
		return errors.New("文件名不能为空")
	}

	// 清理文件名，防止路径遍历攻击
	filename = sanitizeFilename(filename)

	// 创建目标文件
	filePath := filepath.Join(s.uploadDir, filename)
	dst, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// 复制内容，同时检查文件大小
	written, err := io.CopyN(dst, content, s.maxSize+1)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	// 检查文件大小是否超过限制
	if written > s.maxSize {
		// 删除已部分写入的文件
		os.Remove(filePath)
		return errors.New("文件大小超过限制")
	}

	return nil
}

// DownloadFile 下载文件
func (s *LocalFileService) DownloadFile(filename string) (io.ReadCloser, error) {
	if filename == "" {
		return nil, errors.New("文件名不能为空")
	}

	// 清理文件名
	filename = sanitizeFilename(filename)

	// 打开文件
	filePath := filepath.Join(s.uploadDir, filename)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// ListFiles 列出所有上传的文件
func (s *LocalFileService) ListFiles() ([]string, error) {
	entries, err := os.ReadDir(s.uploadDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// DeleteFile 删除文件
func (s *LocalFileService) DeleteFile(filename string) error {
	if filename == "" {
		return errors.New("文件名不能为空")
	}

	// 清理文件名
	filename = sanitizeFilename(filename)

	filePath := filepath.Join(s.uploadDir, filename)
	return os.Remove(filePath)
}

// 清理文件名，防止路径遍历攻击
func sanitizeFilename(filename string) string {
	// 移除路径分隔符和特殊字符
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")
	filename = strings.ReplaceAll(filename, "../", "")
	filename = strings.ReplaceAll(filename, "..\\", "")
	return filename
}
