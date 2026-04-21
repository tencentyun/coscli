package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetFileListStatistic(t *testing.T) {
	t.Run("目录不存在时返回错误", func(t *testing.T) {
		fo := &FileOperations{
			Operation: Operation{},
			Monitor:   &FileProcessMonitor{},
		}
		err := getFileListStatistic("/nonexistent/dir/", fo)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("普通目录统计文件数量和大小", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-stat-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		f1, _ := os.CreateTemp(tmpDir, "file1-*.txt")
		f1.WriteString("hello world") // 11 bytes
		f1.Close()
		f2, _ := os.CreateTemp(tmpDir, "file2-*.txt")
		f2.WriteString("test") // 4 bytes
		f2.Close()

		fo := &FileOperations{
			Operation: Operation{},
			Monitor:   &FileProcessMonitor{},
		}
		err := getFileListStatistic(tmpDir+string(os.PathSeparator), fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		// TotalSize 是公开字段，验证文件大小被统计
		if fo.Monitor.TotalSize == 0 {
			t.Errorf("期望 TotalSize>0，实际 %d", fo.Monitor.TotalSize)
		}
	})

	t.Run("OnlyCurrentDir=true 只统计当前目录文件", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-stat-onlycurrent")
		subDir := filepath.Join(tmpDir, "subdir")
		os.MkdirAll(subDir, 0755)
		defer os.RemoveAll(tmpDir)

		f1, _ := os.CreateTemp(tmpDir, "file1-*.txt")
		f1.WriteString("hello")
		f1.Close()
		f2, _ := os.CreateTemp(subDir, "file2-*.txt")
		f2.WriteString("world world world") // 子目录文件更大
		f2.Close()

		fo := &FileOperations{
			Operation: Operation{OnlyCurrentDir: true},
			Monitor:   &FileProcessMonitor{},
		}
		err := getFileListStatistic(tmpDir+string(os.PathSeparator), fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		// OnlyCurrentDir=true 只统计当前目录，TotalSize 应等于 f1 的大小
		if fo.Monitor.TotalSize != 5 {
			t.Errorf("期望 TotalSize=5（只统计当前目录），实际 %d", fo.Monitor.TotalSize)
		}
	})
}

func TestGetCurrentDirFilesStatistic(t *testing.T) {
	t.Run("统计当前目录文件（不含子目录）", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-current-stat")
		subDir := filepath.Join(tmpDir, "subdir")
		os.MkdirAll(subDir, 0755)
		defer os.RemoveAll(tmpDir)

		f1, _ := os.CreateTemp(tmpDir, "file1-*.txt")
		f1.WriteString("hello")
		f1.Close()
		f2, _ := os.CreateTemp(subDir, "file2-*.txt")
		f2.Close()

		fo := &FileOperations{
			Operation: Operation{},
			Monitor:   &FileProcessMonitor{},
		}
		err := getCurrentDirFilesStatistic(tmpDir, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if fo.Monitor.TotalSize != 5 {
			t.Errorf("期望 TotalSize=5，实际 %d", fo.Monitor.TotalSize)
		}
	})
}

func TestGenerateFileList(t *testing.T) {
	t.Run("单个文件路径发送到 channel", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "coscli-single-*.txt")
		if err != nil {
			t.Fatalf("创建临时文件失败: %v", err)
		}
		tmpFile.WriteString("content")
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		chFiles := make(chan fileInfoType, 10)
		chListError := make(chan error, 2)
		fo := &FileOperations{
			Operation: Operation{},
			Monitor:   &FileProcessMonitor{},
		}
		go generateFileList(tmpFile.Name(), chFiles, chListError, fo)

		var files []fileInfoType
		for f := range chFiles {
			files = append(files, f)
		}
		err = <-chListError
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(files) != 1 {
			t.Errorf("期望 1 个文件，实际 %d", len(files))
		}
	})

	t.Run("目录路径递归发送文件到 channel", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-filelist-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		for i := 0; i < 3; i++ {
			f, _ := os.CreateTemp(tmpDir, "file-*.txt")
			f.Close()
		}

		chFiles := make(chan fileInfoType, 10)
		chListError := make(chan error, 2)
		fo := &FileOperations{
			Operation: Operation{},
			Monitor:   &FileProcessMonitor{},
		}
		go generateFileList(tmpDir, chFiles, chListError, fo)

		var files []fileInfoType
		for f := range chFiles {
			files = append(files, f)
		}
		err := <-chListError
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(files) != 3 {
			t.Errorf("期望 3 个文件，实际 %d", len(files))
		}
	})

	t.Run("路径不存在时发送错误", func(t *testing.T) {
		chFiles := make(chan fileInfoType, 10)
		chListError := make(chan error, 2)
		fo := &FileOperations{
			Operation: Operation{},
			Monitor:   &FileProcessMonitor{},
		}
		go generateFileList("/nonexistent/path", chFiles, chListError, fo)

		// 消费 channel
		for range chFiles {
		}
		err := <-chListError
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})
}

func TestGetCurrentDirFileList(t *testing.T) {
	t.Run("只返回当前目录文件（不含子目录）", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-currentdir-list")
		subDir := filepath.Join(tmpDir, "subdir")
		os.MkdirAll(subDir, 0755)
		defer os.RemoveAll(tmpDir)

		f1, _ := os.CreateTemp(tmpDir, "file1-*.txt")
		f1.Close()
		f2, _ := os.CreateTemp(subDir, "file2-*.txt")
		f2.Close()

		chFiles := make(chan fileInfoType, 10)
		fo := &FileOperations{
			Operation: Operation{},
			Monitor:   &FileProcessMonitor{},
		}
		err := getCurrentDirFileList(tmpDir, chFiles, fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		close(chFiles)

		var files []fileInfoType
		for f := range chFiles {
			files = append(files, f)
		}
		if len(files) != 1 {
			t.Errorf("期望 1 个文件（只含当前目录），实际 %d", len(files))
		}
	})
}

func TestGetFileListFunc(t *testing.T) {
	t.Run("GetFileList 包装函数正常工作", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "coscli-getfilelist-test")
		os.MkdirAll(tmpDir, 0755)
		defer os.RemoveAll(tmpDir)

		f, _ := os.CreateTemp(tmpDir, "file-*.txt")
		f.Close()

		chFiles := make(chan fileInfoType, 10)
		chFinish := make(chan error, 2)
		fo := &FileOperations{
			Operation: Operation{},
			Monitor:   &FileProcessMonitor{},
		}
		go GetFileList(tmpDir+string(os.PathSeparator), chFiles, chFinish, fo)

		var files []fileInfoType
		for f := range chFiles {
			files = append(files, f)
		}
		if len(files) != 1 {
			t.Errorf("期望 1 个文件，实际 %d", len(files))
		}
	})
}

// TestFileStatistic 跳过：fileStatistic 内部调用 freshProgress() 向 chProgressSignal 发送信号，
// 需要 progressBar goroutine 消费，否则 channel 满后阻塞。
// 其内部逻辑已通过 TestGetFileListStatistic 和 TestGetCurrentDirFilesStatistic 覆盖。
