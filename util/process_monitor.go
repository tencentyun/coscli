package util

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	normalExit = iota
	errExit
)

var (
	pmMu             sync.RWMutex
	chProgressSignal chan chProgressSignalType
	signalNum        = 0
)

type chProgressSignalType struct {
	finish   bool
	exitStat int
}

var processTickInterval int64 = 5
var clearStrLen int = 0
var clearStr string = strings.Repeat(" ", clearStrLen)

type FileProcessMonitor struct {
	TotalSize      int64
	totalNum       int64
	TransferSize   int64
	skipSize       int64
	dealSize       int64
	fileNum        int64
	dirNum         int64
	skipNum        int64
	skipNumDir     int64
	ErrNum         int64
	lastSnapSize   int64
	tickDuration   int64
	seekAheadError error
	op             CpType
	seekAheadEnd   bool
	finish         bool
	_              uint32 // 占位符 用于确保下一个数据64位对齐
	lastSnapTime   time.Time
}

type FileProcessMonitorSnap struct {
	transferSize  int64
	skipSize      int64
	dealSize      int64
	fileNum       int64
	dirNum        int64
	skipNum       int64
	skipNumDir    int64
	errNum        int64
	okNum         int64
	dealNum       int64
	duration      int64
	incrementSize int64
}

func (fpm *FileProcessMonitor) init(op CpType) {
	fpm.op = op
	fpm.TotalSize = 0
	fpm.totalNum = 0
	fpm.seekAheadEnd = false
	fpm.seekAheadError = nil
	fpm.TransferSize = 0
	fpm.skipSize = 0
	fpm.dealSize = 0
	fpm.fileNum = 0
	fpm.dirNum = 0
	fpm.skipNum = 0
	fpm.ErrNum = 0
	fpm.finish = false
	fpm.lastSnapSize = 0
	fpm.lastSnapTime = time.Now()
	fpm.tickDuration = processTickInterval * int64(time.Second)
}

func (fpm *FileProcessMonitor) setScanError(err error) {
	fpm.seekAheadError = err
	fpm.seekAheadEnd = true
}
func (fpm *FileProcessMonitor) updateScanNum(num int64) {
	fpm.totalNum = fpm.totalNum + num
}
func (fpm *FileProcessMonitor) updateScanSizeNum(size, num int64) {
	fpm.TotalSize = fpm.TotalSize + size
	fpm.totalNum = fpm.totalNum + num
}

func (fpm *FileProcessMonitor) updateTransferSize(size int64) {
	atomic.AddInt64(&fpm.TransferSize, size)
}

func (fpm *FileProcessMonitor) updateDealSize(size int64) {
	atomic.AddInt64(&fpm.dealSize, size)
}

func (fpm *FileProcessMonitor) updateFile(size, num int64) {
	atomic.AddInt64(&fpm.fileNum, num)
	atomic.AddInt64(&fpm.TransferSize, size)
	atomic.AddInt64(&fpm.dealSize, size)
}

func (fpm *FileProcessMonitor) updateDir(size, num int64) {
	atomic.AddInt64(&fpm.dirNum, num)
	atomic.AddInt64(&fpm.TransferSize, size)
	atomic.AddInt64(&fpm.dealSize, size)
}

func (fpm *FileProcessMonitor) updateSkip(size, num int64) {
	atomic.AddInt64(&fpm.skipNum, num)
	atomic.AddInt64(&fpm.skipSize, size)
}

func (fpm *FileProcessMonitor) updateSkipDir(num int64) {
	atomic.AddInt64(&fpm.skipNumDir, num)
}

func (fpm *FileProcessMonitor) updateErr(size, num int64) {
	atomic.AddInt64(&fpm.ErrNum, num)
	//atomic.AddInt64(&fpm.TransferSize, size)
}

func (fpm *FileProcessMonitor) updateMonitor(skip bool, err error, isDir bool, size int64) {
	if err != nil {
		fpm.updateErr(0, 1)
	} else if skip {
		if !isDir {
			fpm.updateSkip(size, 1)
		} else {
			fpm.updateSkipDir(1)
		}
	} else if isDir {
		fpm.updateDir(size, 1)
	} else {
		fpm.updateFile(size, 1)
	}
	freshProgress()
}

func (fpm *FileProcessMonitor) setScanEnd() {
	fpm.seekAheadEnd = true
}

func (fpm *FileProcessMonitor) progressBar(finish bool, exitStat int) string {
	if fpm.finish {
		return ""
	}
	fpm.finish = fpm.finish || finish
	if !finish {
		return fpm.getProgressBar()
	}
	return fpm.getFinishBar(exitStat)
}

func (fpm *FileProcessMonitor) getProgressBar() string {
	pmMu.RLock()
	defer pmMu.RUnlock()

	snap := fpm.getSnapshot()
	if snap.duration < fpm.tickDuration {
		return ""
	} else {
		fpm.lastSnapTime = time.Now()
		snap.incrementSize = fpm.TransferSize - fpm.lastSnapSize
		fpm.lastSnapSize = snap.transferSize
	}

	if fpm.seekAheadEnd && fpm.seekAheadError == nil {
		return getClearStr(fmt.Sprintf("Total num: %d, size: %s. Processed num: %d%s%s, Progress: %.3f%s, Speed: %s/s", fpm.totalNum, getSizeString(fpm.TotalSize), snap.dealNum, fpm.getDealNumDetail(snap), fpm.getDealSizeDetail(snap), fpm.getPrecent(snap), "%%", fpm.getSpeed(snap)))
	}
	scanNum := max(fpm.totalNum, snap.dealNum)
	scanSize := max(fpm.TotalSize, snap.dealSize)
	return getClearStr(fmt.Sprintf("Scanned num: %d, size: %s. Processed num: %d%s%s, Speed: %s/s.", scanNum, getSizeString(scanSize), snap.dealNum, fpm.getDealNumDetail(snap), fpm.getDealSizeDetail(snap), fpm.getSpeed(snap)))
}

func (fpm *FileProcessMonitor) getFinishBar(exitStat int) string {
	if exitStat == normalExit {
		return fpm.getWholeFinishBar()
	}
	return fpm.getDefeatBar()
}

func (fpm *FileProcessMonitor) getWholeFinishBar() string {
	snap := fpm.getSnapshot()
	if fpm.seekAheadEnd && fpm.seekAheadError == nil {
		if snap.errNum == 0 {
			return getClearStr(fmt.Sprintf("Succeed: Total num: %d, size: %s. OK num: %d%s%s.\n", fpm.totalNum, getSizeString(fpm.TotalSize), snap.okNum, fpm.getDealNumDetail(snap), fpm.getSkipSize(snap)))
		}
		return getClearStr(fmt.Sprintf("FinishWithError: Total num: %d, size: %s. Error num: %d. OK num: %d%s%s.\n", fpm.totalNum, getSizeString(fpm.TotalSize), snap.errNum, snap.okNum, fpm.getOKNumDetail(snap), fpm.getSizeDetail(snap)))
	}
	scanNum := max(fpm.totalNum, snap.dealNum)
	if snap.errNum == 0 {
		return getClearStr(fmt.Sprintf("Succeed: Total num: %d, size: %s. OK num: %d%s%s.\n", scanNum, getSizeString(snap.dealSize), snap.okNum, fpm.getDealNumDetail(snap), fpm.getSkipSize(snap)))
	}
	return getClearStr(fmt.Sprintf("FinishWithError: Scanned %d %s. Error num: %d. OK num: %d%s%s.\n", scanNum, fpm.getSubject(), snap.errNum, snap.okNum, fpm.getOKNumDetail(snap), fpm.getSizeDetail(snap)))
}

func (fpm *FileProcessMonitor) getDefeatBar() string {
	snap := fpm.getSnapshot()
	if fpm.seekAheadEnd && fpm.seekAheadError == nil {
		return getClearStr(fmt.Sprintf("Total num: %d, size: %s. Processed num: %d%s%s. When error happens.\n", fpm.totalNum, getSizeString(fpm.TotalSize), snap.okNum, fpm.getOKNumDetail(snap), fpm.getSizeDetail(snap)))
	}
	scanNum := max(fpm.totalNum, snap.dealNum)
	return getClearStr(fmt.Sprintf("Scanned %d %s. Processed num: %d%s%s. When error happens.\n", scanNum, fpm.getSubject(), snap.okNum, fpm.getOKNumDetail(snap), fpm.getSizeDetail(snap)))
}

func (fpm *FileProcessMonitor) getSnapshot() *FileProcessMonitorSnap {
	var snap FileProcessMonitorSnap
	snap.transferSize = fpm.TransferSize
	snap.skipSize = fpm.skipSize
	snap.dealSize = fpm.dealSize + snap.skipSize
	snap.fileNum = fpm.fileNum
	snap.dirNum = fpm.dirNum
	snap.skipNum = fpm.skipNum
	snap.errNum = fpm.ErrNum
	snap.okNum = snap.fileNum + snap.dirNum + snap.skipNum
	snap.dealNum = snap.okNum + snap.errNum
	snap.skipNumDir = fpm.skipNumDir
	now := time.Now()
	snap.duration = now.Sub(fpm.lastSnapTime).Nanoseconds()

	return &snap
}

func getClearStr(str string) string {
	if clearStrLen <= len(str) {
		clearStrLen = len(str)
		return fmt.Sprintf("\r%s", str)
	}
	clearStr = strings.Repeat(" ", clearStrLen)
	return fmt.Sprintf("\r%s\r%s", clearStr, str)
}

func (fpm *FileProcessMonitor) getDealNumDetail(snap *FileProcessMonitorSnap) string {
	return fpm.getNumDetail(snap, true)
}

func (fpm *FileProcessMonitor) getOKNumDetail(snap *FileProcessMonitorSnap) string {
	return fpm.getNumDetail(snap, false)
}

func (fpm *FileProcessMonitor) getNumDetail(snap *FileProcessMonitorSnap, hasErr bool) string {
	if !hasErr && snap.okNum == 0 {
		return ""
	}
	strList := []string{}
	if hasErr && snap.errNum != 0 {
		strList = append(strList, fmt.Sprintf("Error %d %s", snap.errNum, fpm.getSubject()))
	}
	if snap.fileNum != 0 {
		strList = append(strList, fmt.Sprintf("%s %d %s", fpm.getOPStr(), snap.fileNum, fpm.getSubject()))
	}
	if snap.dirNum != 0 {
		str := fmt.Sprintf("%d directories", snap.dirNum)
		if snap.fileNum == 0 {
			str = fmt.Sprintf("%s %d directories", fpm.getOPStr(), snap.dirNum)
		}
		strList = append(strList, str)
	}
	if snap.skipNum != 0 {
		strList = append(strList, fmt.Sprintf("skip %d %s", snap.skipNum, fpm.getSubject()))
	}
	if snap.skipNumDir != 0 {
		strList = append(strList, fmt.Sprintf("skip %d directory", snap.skipNumDir))
	}

	if len(strList) == 0 {
		return ""
	}
	return fmt.Sprintf("(%s)", strings.Join(strList, ", "))
}

func (fpm *FileProcessMonitor) getSizeDetail(snap *FileProcessMonitorSnap) string {
	if snap.skipSize == 0 {
		return fmt.Sprintf(", Transfer size: %s", getSizeString(snap.transferSize))
	}
	if snap.transferSize == 0 {
		return fmt.Sprintf(", Skip size: %s", getSizeString(snap.skipSize))
	}
	return fmt.Sprintf(", OK size: %s(transfer: %s, skip: %s)", getSizeString(snap.transferSize+snap.skipSize), getSizeString(snap.transferSize), getSizeString(snap.skipSize))
}

func (fpm *FileProcessMonitor) getSkipSize(snap *FileProcessMonitorSnap) string {
	if snap.skipSize != 0 {
		return fmt.Sprintf(", Skip size: %s", getSizeString(snap.skipSize))
	}
	return ""
}

func (fpm *FileProcessMonitor) getDealSizeDetail(snap *FileProcessMonitorSnap) string {
	return fmt.Sprintf(", OK size: %s", getSizeString(snap.dealSize))
}

func (fpm *FileProcessMonitor) getSpeed(snap *FileProcessMonitorSnap) string {
	speed := (float64(snap.incrementSize)) / (float64(snap.duration) * 1e-9)
	return formatBytes(speed)
}

func (fpm *FileProcessMonitor) getPrecent(snap *FileProcessMonitorSnap) float64 {
	if fpm.seekAheadEnd && fpm.seekAheadError == nil {
		if fpm.TotalSize != 0 {
			return float64((snap.dealSize)*100.0) / float64(fpm.TotalSize)
		}
		if fpm.totalNum != 0 {
			return float64((snap.dealNum)*100.0) / float64(fpm.totalNum)
		}
		return 100
	}
	return 0
}

func (fpm *FileProcessMonitor) getOPStr() string {
	switch fpm.op {
	case CpTypeUpload:
		return "upload"
	case CpTypeDownload:
		return "download"
	default:
		return "copy"
	}
}

func (fpm *FileProcessMonitor) getSubject() string {
	switch fpm.op {
	case CpTypeUpload:
		return "files"
	default:
		return "objects"
	}
}

func (fpm *FileProcessMonitor) GetFinishInfo() string {
	snap := fpm.getSnapshot()
	if fpm.seekAheadEnd && fpm.seekAheadError == nil {
		if snap.errNum == 0 {
			return fmt.Sprintf("Succeed: Total num: %d, size: %s. OK num: %d%s%s.\n", fpm.totalNum, getSizeString(fpm.TotalSize), snap.okNum, fpm.getDealNumDetail(snap), fpm.getSkipSize(snap))
		}
		return fmt.Sprintf("FinishWithError: Total num: %d, size: %s. Error num: %d. OK num: %d%s%s.\n", fpm.totalNum, getSizeString(fpm.TotalSize), snap.errNum, snap.okNum, fpm.getOKNumDetail(snap), fpm.getSizeDetail(snap))
	}
	scanNum := max(fpm.totalNum, snap.dealNum)
	if snap.errNum == 0 {
		return fmt.Sprintf("Succeed: Total num: %d, size: %s. OK num: %d%s%s.\n", scanNum, getSizeString(snap.dealSize), snap.okNum, fpm.getDealNumDetail(snap), fpm.getSkipSize(snap))
	}
	return fmt.Sprintf("FinishWithError: Scanned %d %s. Error num: %d. OK num: %d%s%s.\n", scanNum, fpm.getSubject(), snap.errNum, snap.okNum, fpm.getOKNumDetail(snap), fpm.getSizeDetail(snap))
}
