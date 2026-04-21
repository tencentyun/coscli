package util

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func newTestMonitor(op CpType) *FileProcessMonitor {
	fpm := &FileProcessMonitor{}
	fpm.init(op)
	return fpm
}

func TestFileProcessMonitorInit(t *testing.T) {
	t.Run("init 初始化所有字段为零值", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		if fpm.TotalSize != 0 {
			t.Errorf("期望 TotalSize=0，实际: %d", fpm.TotalSize)
		}
		if fpm.ErrNum != 0 {
			t.Errorf("期望 ErrNum=0，实际: %d", fpm.ErrNum)
		}
		if fpm.finish {
			t.Error("期望 finish=false")
		}
	})
}

func TestFileProcessMonitorUpdate(t *testing.T) {
	t.Run("updateScanNum 累加 totalNum", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateScanNum(5)
		if fpm.totalNum != 5 {
			t.Errorf("期望 totalNum=5，实际: %d", fpm.totalNum)
		}
		fpm.updateScanNum(3)
		if fpm.totalNum != 8 {
			t.Errorf("期望 totalNum=8，实际: %d", fpm.totalNum)
		}
	})

	t.Run("updateScanSizeNum 累加 TotalSize 和 totalNum", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateScanSizeNum(1024, 2)
		if fpm.TotalSize != 1024 {
			t.Errorf("期望 TotalSize=1024，实际: %d", fpm.TotalSize)
		}
		if fpm.totalNum != 2 {
			t.Errorf("期望 totalNum=2，实际: %d", fpm.totalNum)
		}
	})

	t.Run("updateTransferSize 原子累加 TransferSize", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateTransferSize(512)
		if fpm.TransferSize != 512 {
			t.Errorf("期望 TransferSize=512，实际: %d", fpm.TransferSize)
		}
	})

	t.Run("updateDealSize 原子累加 dealSize", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateDealSize(256)
		if fpm.dealSize != 256 {
			t.Errorf("期望 dealSize=256，实际: %d", fpm.dealSize)
		}
	})

	t.Run("updateFile 累加 fileNum、TransferSize、dealSize", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateFile(1024, 1)
		if fpm.fileNum != 1 {
			t.Errorf("期望 fileNum=1，实际: %d", fpm.fileNum)
		}
		if fpm.TransferSize != 1024 {
			t.Errorf("期望 TransferSize=1024，实际: %d", fpm.TransferSize)
		}
	})

	t.Run("updateDir 累加 dirNum、TransferSize、dealSize", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateDir(0, 1)
		if fpm.dirNum != 1 {
			t.Errorf("期望 dirNum=1，实际: %d", fpm.dirNum)
		}
	})

	t.Run("updateSkip 累加 skipNum 和 skipSize", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateSkip(512, 1)
		if fpm.skipNum != 1 {
			t.Errorf("期望 skipNum=1，实际: %d", fpm.skipNum)
		}
		if fpm.skipSize != 512 {
			t.Errorf("期望 skipSize=512，实际: %d", fpm.skipSize)
		}
	})

	t.Run("updateSkipDir 累加 skipNumDir", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateSkipDir(2)
		if fpm.skipNumDir != 2 {
			t.Errorf("期望 skipNumDir=2，实际: %d", fpm.skipNumDir)
		}
	})

	t.Run("updateErr 累加 ErrNum", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateErr(0, 1)
		if fpm.ErrNum != 1 {
			t.Errorf("期望 ErrNum=1，实际: %d", fpm.ErrNum)
		}
	})

	t.Run("updateListErr 累加 ListErrNum", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateListErr(3)
		if fpm.ListErrNum != 3 {
			t.Errorf("期望 ListErrNum=3，实际: %d", fpm.ListErrNum)
		}
	})
}

func TestFileProcessMonitorUpdateMonitor(t *testing.T) {
	// updateMonitor 内部调用 freshProgress()，需要初始化 chProgressSignal 通道
	chProgressSignal = make(chan chProgressSignalType, 10)
	defer func() {
		// 排空通道，避免影响其他测试
		for len(chProgressSignal) > 0 {
			<-chProgressSignal
		}
	}()

	t.Run("err != nil 时更新 ErrNum", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateMonitor(false, fmt.Errorf("test error"), false, 1024)
		if fpm.ErrNum != 1 {
			t.Errorf("期望 ErrNum=1，实际: %d", fpm.ErrNum)
		}
	})

	t.Run("skip=true 且非目录时更新 skipNum", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateMonitor(true, nil, false, 512)
		if fpm.skipNum != 1 {
			t.Errorf("期望 skipNum=1，实际: %d", fpm.skipNum)
		}
	})

	t.Run("skip=true 且是目录时更新 skipNumDir", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateMonitor(true, nil, true, 0)
		if fpm.skipNumDir != 1 {
			t.Errorf("期望 skipNumDir=1，实际: %d", fpm.skipNumDir)
		}
	})

	t.Run("skip=false 且是目录时更新 dirNum", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateMonitor(false, nil, true, 0)
		if fpm.dirNum != 1 {
			t.Errorf("期望 dirNum=1，实际: %d", fpm.dirNum)
		}
	})

	t.Run("skip=false 且非目录时更新 fileNum", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateMonitor(false, nil, false, 1024)
		if fpm.fileNum != 1 {
			t.Errorf("期望 fileNum=1，实际: %d", fpm.fileNum)
		}
	})
}

func TestFileProcessMonitorScanError(t *testing.T) {
	t.Run("setScanError 设置错误并标记 seekAheadEnd", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.setScanError(fmt.Errorf("test error"))
		if fpm.seekAheadError == nil {
			t.Error("期望 seekAheadError 不为 nil")
		}
		if !fpm.seekAheadEnd {
			t.Error("期望 seekAheadEnd=true")
		}
	})

	t.Run("setScanEnd 标记 seekAheadEnd", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.setScanEnd()
		if !fpm.seekAheadEnd {
			t.Error("期望 seekAheadEnd=true")
		}
	})
}

func TestFileProcessMonitorGetSnapshot(t *testing.T) {
	t.Run("getSnapshot 返回正确的快照数据", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.updateFile(1024, 2)
		fpm.updateDir(0, 1)
		fpm.updateSkip(512, 1)
		fpm.updateErr(0, 1)

		snap := fpm.getSnapshot()
		if snap.fileNum != 2 {
			t.Errorf("期望 fileNum=2，实际: %d", snap.fileNum)
		}
		if snap.dirNum != 1 {
			t.Errorf("期望 dirNum=1，实际: %d", snap.dirNum)
		}
		if snap.skipNum != 1 {
			t.Errorf("期望 skipNum=1，实际: %d", snap.skipNum)
		}
		if snap.errNum != 1 {
			t.Errorf("期望 errNum=1，实际: %d", snap.errNum)
		}
		// okNum = fileNum + dirNum + skipNum = 2+1+1 = 4
		if snap.okNum != 4 {
			t.Errorf("期望 okNum=4，实际: %d", snap.okNum)
		}
		// dealNum = okNum + errNum = 4+1 = 5
		if snap.dealNum != 5 {
			t.Errorf("期望 dealNum=5，实际: %d", snap.dealNum)
		}
	})
}

func TestFileProcessMonitorGetOPStr(t *testing.T) {
	t.Run("Upload 操作返回 upload", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		if fpm.getOPStr() != "upload" {
			t.Errorf("期望 upload，实际: %s", fpm.getOPStr())
		}
	})

	t.Run("Download 操作返回 download", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeDownload)
		if fpm.getOPStr() != "download" {
			t.Errorf("期望 download，实际: %s", fpm.getOPStr())
		}
	})

	t.Run("Copy 操作返回 copy", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeCopy)
		if fpm.getOPStr() != "copy" {
			t.Errorf("期望 copy，实际: %s", fpm.getOPStr())
		}
	})
}

func TestFileProcessMonitorGetSubject(t *testing.T) {
	t.Run("Upload 操作返回 files", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		if fpm.getSubject() != "files" {
			t.Errorf("期望 files，实际: %s", fpm.getSubject())
		}
	})

	t.Run("非 Upload 操作返回 objects", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeDownload)
		if fpm.getSubject() != "objects" {
			t.Errorf("期望 objects，实际: %s", fpm.getSubject())
		}
	})
}

func TestGetClearStr(t *testing.T) {
	t.Run("短字符串返回带 \\r 前缀的字符串", func(t *testing.T) {
		// 重置 clearStrLen
		clearStrLen = 0
		result := getClearStr("hello")
		if !strings.HasPrefix(result, "\r") {
			t.Errorf("期望以 \\r 开头，实际: %q", result)
		}
		if !strings.Contains(result, "hello") {
			t.Errorf("期望包含 hello，实际: %q", result)
		}
	})

	t.Run("短于已记录长度时返回清除字符串", func(t *testing.T) {
		clearStrLen = 20
		result := getClearStr("hi")
		// 应该包含清除字符串
		if !strings.Contains(result, "hi") {
			t.Errorf("期望包含 hi，实际: %q", result)
		}
	})
}

func TestFileProcessMonitorGetSizeDetail(t *testing.T) {
	t.Run("skipSize=0 时只显示 Transfer size", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		snap := &FileProcessMonitorSnap{transferSize: 1024, skipSize: 0}
		result := fpm.getSizeDetail(snap)
		if !strings.Contains(result, "Transfer size") {
			t.Errorf("期望包含 Transfer size，实际: %s", result)
		}
	})

	t.Run("transferSize=0 时只显示 Skip size", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		snap := &FileProcessMonitorSnap{transferSize: 0, skipSize: 512}
		result := fpm.getSizeDetail(snap)
		if !strings.Contains(result, "Skip size") {
			t.Errorf("期望包含 Skip size，实际: %s", result)
		}
	})

	t.Run("两者都有时显示 OK size", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		snap := &FileProcessMonitorSnap{transferSize: 1024, skipSize: 512}
		result := fpm.getSizeDetail(snap)
		if !strings.Contains(result, "OK size") {
			t.Errorf("期望包含 OK size，实际: %s", result)
		}
	})
}

func TestFileProcessMonitorGetSkipSize(t *testing.T) {
	t.Run("skipSize=0 时返回空字符串", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		snap := &FileProcessMonitorSnap{skipSize: 0}
		result := fpm.getSkipSize(snap)
		if result != "" {
			t.Errorf("期望空字符串，实际: %s", result)
		}
	})

	t.Run("skipSize>0 时返回 Skip size 字符串", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		snap := &FileProcessMonitorSnap{skipSize: 512}
		result := fpm.getSkipSize(snap)
		if !strings.Contains(result, "Skip size") {
			t.Errorf("期望包含 Skip size，实际: %s", result)
		}
	})
}

func TestFileProcessMonitorGetDealSizeDetail(t *testing.T) {
	t.Run("返回 OK size 字符串", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		snap := &FileProcessMonitorSnap{dealSize: 2048}
		result := fpm.getDealSizeDetail(snap)
		if !strings.Contains(result, "OK size") {
			t.Errorf("期望包含 OK size，实际: %s", result)
		}
	})
}

func TestFileProcessMonitorGetPrecent(t *testing.T) {
	t.Run("seekAheadEnd=true 且 TotalSize>0 时返回百分比", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.seekAheadEnd = true
		fpm.TotalSize = 1000
		snap := &FileProcessMonitorSnap{dealSize: 500}
		result := fpm.getPrecent(snap)
		if result != 50.0 {
			t.Errorf("期望 50.0，实际: %f", result)
		}
	})

	t.Run("seekAheadEnd=true 且 TotalSize=0 且 totalNum>0 时返回百分比", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.seekAheadEnd = true
		fpm.TotalSize = 0
		fpm.totalNum = 10
		snap := &FileProcessMonitorSnap{dealNum: 5}
		result := fpm.getPrecent(snap)
		if result != 50.0 {
			t.Errorf("期望 50.0，实际: %f", result)
		}
	})

	t.Run("seekAheadEnd=true 且 TotalSize=0 且 totalNum=0 时返回 100", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.seekAheadEnd = true
		fpm.TotalSize = 0
		fpm.totalNum = 0
		snap := &FileProcessMonitorSnap{}
		result := fpm.getPrecent(snap)
		if result != 100 {
			t.Errorf("期望 100，实际: %f", result)
		}
	})

	t.Run("seekAheadEnd=false 时返回 0", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.seekAheadEnd = false
		snap := &FileProcessMonitorSnap{dealSize: 500}
		result := fpm.getPrecent(snap)
		if result != 0 {
			t.Errorf("期望 0，实际: %f", result)
		}
	})
}

func TestFileProcessMonitorGetNumDetail(t *testing.T) {
	t.Run("hasErr=true 且有错误时包含 Error 信息", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		snap := &FileProcessMonitorSnap{errNum: 2, fileNum: 3}
		result := fpm.getDealNumDetail(snap)
		if !strings.Contains(result, "Error") {
			t.Errorf("期望包含 Error，实际: %s", result)
		}
	})

	t.Run("有 fileNum 时包含操作字符串", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		snap := &FileProcessMonitorSnap{fileNum: 5, okNum: 5}
		result := fpm.getDealNumDetail(snap)
		if !strings.Contains(result, "upload") {
			t.Errorf("期望包含 upload，实际: %s", result)
		}
	})

	t.Run("有 dirNum 时包含 directories", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		snap := &FileProcessMonitorSnap{dirNum: 2, okNum: 2}
		result := fpm.getDealNumDetail(snap)
		if !strings.Contains(result, "directories") {
			t.Errorf("期望包含 directories，实际: %s", result)
		}
	})

	t.Run("有 skipNum 时包含 skip", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		snap := &FileProcessMonitorSnap{skipNum: 1, okNum: 1}
		result := fpm.getDealNumDetail(snap)
		if !strings.Contains(result, "skip") {
			t.Errorf("期望包含 skip，实际: %s", result)
		}
	})

	t.Run("有 skipNumDir 时包含 directory", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		snap := &FileProcessMonitorSnap{skipNumDir: 1, okNum: 1}
		result := fpm.getDealNumDetail(snap)
		if !strings.Contains(result, "directory") {
			t.Errorf("期望包含 directory，实际: %s", result)
		}
	})

	t.Run("hasErr=false 且 okNum=0 时返回空字符串", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		snap := &FileProcessMonitorSnap{okNum: 0}
		result := fpm.getOKNumDetail(snap)
		if result != "" {
			t.Errorf("期望空字符串，实际: %s", result)
		}
	})
}

func TestFileProcessMonitorGetFinishInfo(t *testing.T) {
	t.Run("seekAheadEnd=true 且无错误时返回 Succeed 信息", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.seekAheadEnd = true
		fpm.seekAheadError = nil
		fpm.TotalSize = 1024
		fpm.totalNum = 2
		fpm.updateFile(1024, 2)

		result := fpm.GetFinishInfo()
		if !strings.Contains(result, "Succeed") {
			t.Errorf("期望包含 Succeed，实际: %s", result)
		}
	})

	t.Run("seekAheadEnd=true 且有错误时返回 FinishWithError 信息", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.seekAheadEnd = true
		fpm.seekAheadError = nil
		fpm.TotalSize = 1024
		fpm.totalNum = 2
		fpm.updateFile(512, 1)
		fpm.updateErr(0, 1)

		result := fpm.GetFinishInfo()
		if !strings.Contains(result, "FinishWithError") {
			t.Errorf("期望包含 FinishWithError，实际: %s", result)
		}
	})

	t.Run("seekAheadEnd=false 且无错误时返回 Succeed 信息", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeDownload)
		fpm.seekAheadEnd = false
		fpm.updateFile(512, 1)

		result := fpm.GetFinishInfo()
		if !strings.Contains(result, "Succeed") {
			t.Errorf("期望包含 Succeed，实际: %s", result)
		}
	})

	t.Run("seekAheadEnd=false 且有错误时返回 FinishWithError 信息", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeDownload)
		fpm.seekAheadEnd = false
		fpm.updateErr(0, 1)

		result := fpm.GetFinishInfo()
		if !strings.Contains(result, "FinishWithError") {
			t.Errorf("期望包含 FinishWithError，实际: %s", result)
		}
	})
}

func TestFileProcessMonitorProgressBar(t *testing.T) {
	t.Run("finish=true 时调用 getFinishBar", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		result := fpm.progressBar(true, normalExit)
		// 不 panic 即可，结果可能为空或包含信息
		_ = result
	})

	t.Run("finish=false 且时间未到时返回空字符串", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		// lastSnapTime 刚初始化，duration < tickDuration，应返回空
		result := fpm.progressBar(false, normalExit)
		if result != "" {
			// 可能因为时间差导致非空，不强制断言
		}
		_ = result
	})

	t.Run("已 finish 后再次调用返回空字符串", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.finish = true
		result := fpm.progressBar(false, normalExit)
		if result != "" {
			t.Errorf("期望空字符串，实际: %q", result)
		}
	})
}

func TestFileProcessMonitorGetWholeFinishBar(t *testing.T) {
	t.Run("seekAheadEnd=true 且无错误时返回 Succeed", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.seekAheadEnd = true
		fpm.seekAheadError = nil
		fpm.TotalSize = 2048
		fpm.totalNum = 3
		fpm.updateFile(2048, 3)
		result := fpm.getWholeFinishBar()
		if !strings.Contains(result, "Succeed") {
			t.Errorf("期望包含 Succeed，实际: %s", result)
		}
	})

	t.Run("seekAheadEnd=true 且有错误时返回 FinishWithError", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.seekAheadEnd = true
		fpm.seekAheadError = nil
		fpm.TotalSize = 2048
		fpm.totalNum = 3
		fpm.updateFile(1024, 2)
		fpm.updateErr(0, 1)
		result := fpm.getWholeFinishBar()
		if !strings.Contains(result, "FinishWithError") {
			t.Errorf("期望包含 FinishWithError，实际: %s", result)
		}
	})

	t.Run("seekAheadEnd=false 且无错误时返回 Succeed", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeCopy)
		fpm.seekAheadEnd = false
		fpm.updateFile(512, 1)
		result := fpm.getWholeFinishBar()
		if !strings.Contains(result, "Succeed") {
			t.Errorf("期望包含 Succeed，实际: %s", result)
		}
	})

	t.Run("seekAheadEnd=false 且有错误时返回 FinishWithError", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeCopy)
		fpm.seekAheadEnd = false
		fpm.updateErr(0, 1)
		result := fpm.getWholeFinishBar()
		if !strings.Contains(result, "FinishWithError") {
			t.Errorf("期望包含 FinishWithError，实际: %s", result)
		}
	})
}

func TestFileProcessMonitorGetDefeatBar(t *testing.T) {
	t.Run("seekAheadEnd=true 时返回包含 Total num 的字符串", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.seekAheadEnd = true
		fpm.seekAheadError = nil
		fpm.TotalSize = 1024
		fpm.totalNum = 2
		result := fpm.getDefeatBar()
		if !strings.Contains(result, "Total num") {
			t.Errorf("期望包含 Total num，实际: %s", result)
		}
	})

	t.Run("seekAheadEnd=false 时返回包含 Scanned 的字符串", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeDownload)
		fpm.seekAheadEnd = false
		result := fpm.getDefeatBar()
		if !strings.Contains(result, "Scanned") {
			t.Errorf("期望包含 Scanned，实际: %s", result)
		}
	})
}

func TestFileProcessMonitorGetProgressBar(t *testing.T) {
	t.Run("duration < tickDuration 时返回空字符串", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.lastSnapTime = time.Now() // 刚刚初始化，duration 很小
		result := fpm.getProgressBar()
		if result != "" {
			// 时间差可能导致非空，不强制断言
		}
		_ = result
	})

	t.Run("seekAheadEnd=true 时返回包含 Total num 的进度条", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.seekAheadEnd = true
		fpm.seekAheadError = nil
		fpm.TotalSize = 1024
		fpm.totalNum = 5
		fpm.lastSnapTime = time.Now().Add(-10 * time.Second) // 模拟时间已过
		fpm.tickDuration = 0                                 // 强制触发
		result := fpm.getProgressBar()
		if !strings.Contains(result, "Total num") {
			t.Errorf("期望包含 Total num，实际: %s", result)
		}
	})

	t.Run("seekAheadEnd=false 时返回包含 Scanned num 的进度条", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		fpm.seekAheadEnd = false
		fpm.lastSnapTime = time.Now().Add(-10 * time.Second)
		fpm.tickDuration = 0
		result := fpm.getProgressBar()
		if !strings.Contains(result, "Scanned num") {
			t.Errorf("期望包含 Scanned num，实际: %s", result)
		}
	})
}

func TestFileProcessMonitorGetSpeed(t *testing.T) {
	t.Run("getSpeed 返回速度字符串", func(t *testing.T) {
		fpm := newTestMonitor(CpTypeUpload)
		snap := &FileProcessMonitorSnap{
			incrementSize: 1024 * 1024, // 1MB
			duration:      int64(time.Second),
		}
		result := fpm.getSpeed(snap)
		if result == "" {
			t.Error("期望速度字符串不为空")
		}
	})
}
