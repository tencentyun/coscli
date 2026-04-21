package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestDrawBar(t *testing.T) {
	t.Run("0% 时全部是 -", func(t *testing.T) {
		bar := drawBar(0)
		if strings.Contains(bar, "#") {
			t.Errorf("期望全部是 -，实际: %s", bar)
		}
	})

	t.Run("100% 时全部是 #", func(t *testing.T) {
		bar := drawBar(100)
		if strings.Contains(bar, "-") {
			t.Errorf("期望全部是 #，实际: %s", bar)
		}
	})

	t.Run("50% 时各占一半", func(t *testing.T) {
		bar := drawBar(50)
		hashCount := strings.Count(bar, "#")
		dashCount := strings.Count(bar, "-")
		if hashCount != dashCount {
			t.Errorf("期望 # 和 - 各占一半，实际 #=%d, -=%d", hashCount, dashCount)
		}
	})
}

func TestProgressChangedCallback(t *testing.T) {
	// ProgressChangedCallback 内部调用 freshProgress()，需要初始化通道
	chProgressSignal = make(chan chProgressSignalType, 10)
	defer func() {
		for len(chProgressSignal) > 0 {
			<-chProgressSignal
		}
	}()

	fpm := newTestMonitor(CpTypeUpload)
	counter := &Counter{}
	listener := &CosListener{fo: &FileOperations{Monitor: fpm}, counter: counter}

	t.Run("ProgressStartedEvent 不更新任何计数", func(t *testing.T) {
		listener.fo.Monitor.updateTransferSize(0)
		event := &cos.ProgressEvent{EventType: cos.ProgressStartedEvent, RWBytes: 100}
		listener.ProgressChangedCallback(event)
		if listener.fo.Monitor.TransferSize != 0 {
			t.Errorf("期望 TransferSize=0，实际: %d", listener.fo.Monitor.TransferSize)
		}
	})

	t.Run("ProgressDataEvent 更新 TransferSize 和 dealSize", func(t *testing.T) {
		listener.fo.Monitor.TransferSize = 0
		listener.fo.Monitor.dealSize = 0
		counter.TransferSize = 0
		event := &cos.ProgressEvent{EventType: cos.ProgressDataEvent, RWBytes: 512}
		listener.ProgressChangedCallback(event)
		if listener.fo.Monitor.TransferSize != 512 {
			t.Errorf("期望 TransferSize=512，实际: %d", listener.fo.Monitor.TransferSize)
		}
		if counter.TransferSize != 512 {
			t.Errorf("期望 counter.TransferSize=512，实际: %d", counter.TransferSize)
		}
	})

	t.Run("ProgressCompletedEvent 不更新任何计数", func(t *testing.T) {
		listener.fo.Monitor.TransferSize = 100
		event := &cos.ProgressEvent{EventType: cos.ProgressCompletedEvent, RWBytes: 200}
		listener.ProgressChangedCallback(event)
		if listener.fo.Monitor.TransferSize != 100 {
			t.Errorf("期望 TransferSize=100，实际: %d", listener.fo.Monitor.TransferSize)
		}
	})

	t.Run("ProgressFailedEvent 减少 dealSize", func(t *testing.T) {
		listener.fo.Monitor.dealSize = 1024
		event := &cos.ProgressEvent{EventType: cos.ProgressFailedEvent, ConsumedBytes: 512}
		listener.ProgressChangedCallback(event)
		if listener.fo.Monitor.dealSize != 512 {
			t.Errorf("期望 dealSize=512，实际: %d", listener.fo.Monitor.dealSize)
		}
	})

	t.Run("未知事件类型不 panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ProgressChangedCallback panic: %v", r)
			}
		}()
		event := &cos.ProgressEvent{EventType: 999}
		listener.ProgressChangedCallback(event)
	})
}

func TestCheckPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coscli-test-checkpath-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fileUrl := &FileUrl{urlStr: tmpDir + string(os.PathSeparator)}

	t.Run("TypeSnapshotPath 是子目录时返回错误", func(t *testing.T) {
		snapshotPath := filepath.Join(tmpDir, "snapshot")
		fo := &FileOperations{Operation: Operation{SnapshotPath: snapshotPath}}
		err := CheckPath(fileUrl, fo, TypeSnapshotPath)
		if err == nil {
			t.Error("期望返回错误（子目录），但得到 nil")
		}
	})

	t.Run("TypeSnapshotPath 不是子目录时返回 nil", func(t *testing.T) {
		otherDir, _ := os.MkdirTemp("", "coscli-test-other-*")
		defer os.RemoveAll(otherDir)
		fo := &FileOperations{Operation: Operation{SnapshotPath: otherDir}}
		err := CheckPath(fileUrl, fo, TypeSnapshotPath)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("TypeFailOutputPath 且 FailOutput=false 时直接返回 nil", func(t *testing.T) {
		fo := &FileOperations{Operation: Operation{FailOutput: false}}
		err := CheckPath(fileUrl, fo, TypeFailOutputPath)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("TypeFailOutputPath 且 FailOutput=true 且是子目录时返回错误", func(t *testing.T) {
		failOutputPath := filepath.Join(tmpDir, "failoutput")
		fo := &FileOperations{Operation: Operation{FailOutput: true, FailOutputPath: failOutputPath}}
		err := CheckPath(fileUrl, fo, TypeFailOutputPath)
		if err == nil {
			t.Error("期望返回错误（子目录），但得到 nil")
		}
	})

	t.Run("TypeProcessLogPath 且 ProcessLog=false 时直接返回 nil", func(t *testing.T) {
		fo := &FileOperations{Operation: Operation{ProcessLog: false}}
		err := CheckPath(fileUrl, fo, TypeProcessLogPath)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("TypeProcessLogPath 且 ProcessLog=true 且是子目录时返回错误", func(t *testing.T) {
		processLogPath := filepath.Join(tmpDir, "processlog")
		fo := &FileOperations{Operation: Operation{ProcessLog: true, ProcessLogPath: processLogPath}}
		err := CheckPath(fileUrl, fo, TypeProcessLogPath)
		if err == nil {
			t.Error("期望返回错误（子目录），但得到 nil")
		}
	})

	t.Run("无效 pathType 时返回错误", func(t *testing.T) {
		fo := &FileOperations{Operation: Operation{}}
		err := CheckPath(fileUrl, fo, "invalid-type")
		if err == nil {
			t.Error("期望返回错误（无效 pathType），但得到 nil")
		}
	})
}

func TestWriteError(t *testing.T) {
	t.Run("writeError 写入错误信息到文件", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "coscli-test-writeerror-*")
		if err != nil {
			t.Fatalf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		fo := &FileOperations{
			Operation:     Operation{FailOutputPath: tmpDir},
			OutPutDirName: "test-output",
			ErrOutput:     &ErrOutput{},
		}
		writeError("test error message\n", fo)

		// 验证文件被创建
		expectedPath := filepath.Join(tmpDir, "test-output", "error.report")
		if _, statErr := os.Stat(expectedPath); os.IsNotExist(statErr) {
			t.Errorf("期望错误文件存在: %s", expectedPath)
		}
	})

	t.Run("CloseErrorOutputFile 关闭文件", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "coscli-test-closeerror-*")
		if err != nil {
			t.Fatalf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		fo := &FileOperations{
			Operation:     Operation{FailOutputPath: tmpDir},
			OutPutDirName: "test-output",
			ErrOutput:     &ErrOutput{},
		}
		writeError("test\n", fo)
		// 不 panic 即可
		CloseErrorOutputFile(fo)
	})

	t.Run("CloseErrorOutputFile 文件为 nil 时不 panic", func(t *testing.T) {
		fo := &FileOperations{ErrOutput: &ErrOutput{}}
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("CloseErrorOutputFile panic: %v", r)
			}
		}()
		CloseErrorOutputFile(fo)
	})
}

func TestWriteProcessLog(t *testing.T) {
	t.Run("ProcessLog=false 时直接返回", func(t *testing.T) {
		fo := &FileOperations{
			Operation: Operation{ProcessLog: false},
		}
		// 不 panic 即可
		writeProcessLog("test log\n", fo)
	})

	t.Run("ProcessLog=true 时写入日志文件", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "coscli-test-processlog-*")
		if err != nil {
			t.Fatalf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		fo := &FileOperations{
			Operation:     Operation{ProcessLog: true, ProcessLogPath: tmpDir},
			OutPutDirName: "test-output",
			ProcessLogger: &ProcessLogger{},
		}
		writeProcessLog("test log message\n", fo)

		expectedPath := filepath.Join(tmpDir, "test-output", "process.log")
		if _, statErr := os.Stat(expectedPath); os.IsNotExist(statErr) {
			t.Errorf("期望日志文件存在: %s", expectedPath)
		}
	})

	t.Run("CloseProcessLoggerFile 关闭文件", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "coscli-test-closeprocesslog-*")
		if err != nil {
			t.Fatalf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		fo := &FileOperations{
			Operation:     Operation{ProcessLog: true, ProcessLogPath: tmpDir},
			OutPutDirName: "test-output",
			ProcessLogger: &ProcessLogger{},
		}
		writeProcessLog("test\n", fo)
		CloseProcessLoggerFile(fo)
	})

	t.Run("CloseProcessLoggerFile 文件为 nil 时不 panic", func(t *testing.T) {
		fo := &FileOperations{ProcessLogger: &ProcessLogger{}}
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("CloseProcessLoggerFile panic: %v", r)
			}
		}()
		CloseProcessLoggerFile(fo)
	})
}
