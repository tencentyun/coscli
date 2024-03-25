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

type CpProcessMonitor struct {
	TotalSize      int64
	totalNum       int64
	TransferSize   int64
	skipSize       int64
	dealSize       int64
	fileNum        int64
	dirNum         int64
	skipNum        int64
	skipNumDir     int64
	errNum         int64
	lastSnapSize   int64
	tickDuration   int64
	seekAheadError error
	op             CpType
	seekAheadEnd   bool
	finish         bool
	_              uint32 // 占位符 用于确保下一个数据64位对齐
	lastSnapTime   time.Time
}

type CpProcessMonitorSnap struct {
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

func (cppm *CpProcessMonitor) init(op CpType) {
	cppm.op = op
	cppm.TotalSize = 0
	cppm.totalNum = 0
	cppm.seekAheadEnd = false
	cppm.seekAheadError = nil
	cppm.TransferSize = 0
	cppm.skipSize = 0
	cppm.dealSize = 0
	cppm.fileNum = 0
	cppm.dirNum = 0
	cppm.skipNum = 0
	cppm.errNum = 0
	cppm.finish = false
	cppm.lastSnapSize = 0
	cppm.lastSnapTime = time.Now()
	cppm.tickDuration = processTickInterval * int64(time.Second)
}

func (cppm *CpProcessMonitor) setScanError(err error) {
	cppm.seekAheadError = err
	cppm.seekAheadEnd = true
}
func (cppm *CpProcessMonitor) updateScanNum(num int64) {
	cppm.totalNum = cppm.totalNum + num
}
func (cppm *CpProcessMonitor) updateScanSizeNum(size, num int64) {
	cppm.TotalSize = cppm.TotalSize + size
	cppm.totalNum = cppm.totalNum + num
}

func (cppm *CpProcessMonitor) updateTransferSize(size int64) {
	atomic.AddInt64(&cppm.TransferSize, size)
}

func (cppm *CpProcessMonitor) updateDealSize(size int64) {
	atomic.AddInt64(&cppm.dealSize, size)
}

func (cppm *CpProcessMonitor) updateFile(size, num int64) {
	atomic.AddInt64(&cppm.fileNum, num)
	//atomic.AddInt64(&cppm.TransferSize, size)
	atomic.AddInt64(&cppm.dealSize, size)
}

func (cppm *CpProcessMonitor) updateDir(size, num int64) {
	atomic.AddInt64(&cppm.dirNum, num)
	//atomic.AddInt64(&cppm.TransferSize, size)
	atomic.AddInt64(&cppm.dealSize, size)
}

func (cppm *CpProcessMonitor) updateSkip(size, num int64) {
	atomic.AddInt64(&cppm.skipNum, num)
	atomic.AddInt64(&cppm.skipSize, size)
}

func (cppm *CpProcessMonitor) updateSkipDir(num int64) {
	atomic.AddInt64(&cppm.skipNumDir, num)
}

func (cppm *CpProcessMonitor) updateErr(size, num int64) {
	atomic.AddInt64(&cppm.errNum, num)
	//atomic.AddInt64(&cppm.TransferSize, size)
}

func (cppm *CpProcessMonitor) updateMonitor(skip bool, err error, isDir bool, size int64) {
	if err != nil {
		cppm.updateErr(0, 1)
	} else if skip {
		if !isDir {
			cppm.updateSkip(size, 1)
		} else {
			cppm.updateSkipDir(1)
		}
	} else if isDir {
		cppm.updateDir(size, 1)
	} else {
		cppm.updateFile(size, 1)
	}
	freshProgress()
}

func (cppm *CpProcessMonitor) setScanEnd() {
	cppm.seekAheadEnd = true
}

func (cppm *CpProcessMonitor) progressBar(finish bool, exitStat int) string {
	if cppm.finish {
		return ""
	}
	cppm.finish = cppm.finish || finish
	if !finish {
		return cppm.getProgressBar()
	}
	return cppm.getFinishBar(exitStat)
}

func (cppm *CpProcessMonitor) getProgressBar() string {
	pmMu.RLock()
	defer pmMu.RUnlock()

	snap := cppm.getSnapshot()
	if snap.duration < cppm.tickDuration {
		return ""
	} else {
		cppm.lastSnapTime = time.Now()
		snap.incrementSize = cppm.TransferSize - cppm.lastSnapSize
		cppm.lastSnapSize = snap.transferSize
	}

	if cppm.seekAheadEnd && cppm.seekAheadError == nil {
		return getClearStr(fmt.Sprintf("Total num: %d, size: %s. Processed num: %d%s%s, Progress: %.3f%s, Speed: %s/s", cppm.totalNum, getSizeString(cppm.TotalSize), snap.dealNum, cppm.getDealNumDetail(snap), cppm.getDealSizeDetail(snap), cppm.getPrecent(snap), "%%", cppm.getSpeed(snap)))
	}
	scanNum := max(cppm.totalNum, snap.dealNum)
	scanSize := max(cppm.TotalSize, snap.dealSize)
	return getClearStr(fmt.Sprintf("Scanned num: %d, size: %s. Processed num: %d%s%s, Speed: %s/s.", scanNum, getSizeString(scanSize), snap.dealNum, cppm.getDealNumDetail(snap), cppm.getDealSizeDetail(snap), cppm.getSpeed(snap)))
}

func (cppm *CpProcessMonitor) getFinishBar(exitStat int) string {
	if exitStat == normalExit {
		return cppm.getWholeFinishBar()
	}
	return cppm.getDefeatBar()
}

func (cppm *CpProcessMonitor) getWholeFinishBar() string {
	snap := cppm.getSnapshot()
	if cppm.seekAheadEnd && cppm.seekAheadError == nil {
		if snap.errNum == 0 {
			return getClearStr(fmt.Sprintf("Succeed: Total num: %d, size: %s. OK num: %d%s%s.\n", cppm.totalNum, getSizeString(cppm.TotalSize), snap.okNum, cppm.getDealNumDetail(snap), cppm.getSkipSize(snap)))
		}
		return getClearStr(fmt.Sprintf("FinishWithError: Total num: %d, size: %s. Error num: %d. OK num: %d%s%s.\n", cppm.totalNum, getSizeString(cppm.TotalSize), snap.errNum, snap.okNum, cppm.getOKNumDetail(snap), cppm.getSizeDetail(snap)))
	}
	scanNum := max(cppm.totalNum, snap.dealNum)
	if snap.errNum == 0 {
		return getClearStr(fmt.Sprintf("Succeed: Total num: %d, size: %s. OK num: %d%s%s.\n", scanNum, getSizeString(snap.dealSize), snap.okNum, cppm.getDealNumDetail(snap), cppm.getSkipSize(snap)))
	}
	return getClearStr(fmt.Sprintf("FinishWithError: Scanned %d %s. Error num: %d. OK num: %d%s%s.\n", scanNum, cppm.getSubject(), snap.errNum, snap.okNum, cppm.getOKNumDetail(snap), cppm.getSizeDetail(snap)))
}

func (cppm *CpProcessMonitor) getDefeatBar() string {
	snap := cppm.getSnapshot()
	if cppm.seekAheadEnd && cppm.seekAheadError == nil {
		return getClearStr(fmt.Sprintf("Total num: %d, size: %s. Processed num: %d%s%s. When error happens.\n", cppm.totalNum, getSizeString(cppm.TotalSize), snap.okNum, cppm.getOKNumDetail(snap), cppm.getSizeDetail(snap)))
	}
	scanNum := max(cppm.totalNum, snap.dealNum)
	return getClearStr(fmt.Sprintf("Scanned %d %s. Processed num: %d%s%s. When error happens.\n", scanNum, cppm.getSubject(), snap.okNum, cppm.getOKNumDetail(snap), cppm.getSizeDetail(snap)))
}

func (cppm *CpProcessMonitor) getSnapshot() *CpProcessMonitorSnap {
	var snap CpProcessMonitorSnap
	snap.transferSize = cppm.TransferSize
	snap.skipSize = cppm.skipSize
	snap.dealSize = cppm.dealSize + snap.skipSize
	snap.fileNum = cppm.fileNum
	snap.dirNum = cppm.dirNum
	snap.skipNum = cppm.skipNum
	snap.errNum = cppm.errNum
	snap.okNum = snap.fileNum + snap.dirNum + snap.skipNum
	snap.dealNum = snap.okNum + snap.errNum
	snap.skipNumDir = cppm.skipNumDir
	now := time.Now()
	snap.duration = now.Sub(cppm.lastSnapTime).Nanoseconds()

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

func (cppm *CpProcessMonitor) getDealNumDetail(snap *CpProcessMonitorSnap) string {
	return cppm.getNumDetail(snap, true)
}

func (cppm *CpProcessMonitor) getOKNumDetail(snap *CpProcessMonitorSnap) string {
	return cppm.getNumDetail(snap, false)
}

func (cppm *CpProcessMonitor) getNumDetail(snap *CpProcessMonitorSnap, hasErr bool) string {
	if !hasErr && snap.okNum == 0 {
		return ""
	}
	strList := []string{}
	if hasErr && snap.errNum != 0 {
		strList = append(strList, fmt.Sprintf("Error %d %s", snap.errNum, cppm.getSubject()))
	}
	if snap.fileNum != 0 {
		strList = append(strList, fmt.Sprintf("%s %d %s", cppm.getOPStr(), snap.fileNum, cppm.getSubject()))
	}
	if snap.dirNum != 0 {
		str := fmt.Sprintf("%d directories", snap.dirNum)
		if snap.fileNum == 0 {
			str = fmt.Sprintf("%s %d directories", cppm.getOPStr(), snap.dirNum)
		}
		strList = append(strList, str)
	}
	if snap.skipNum != 0 {
		strList = append(strList, fmt.Sprintf("skip %d %s", snap.skipNum, cppm.getSubject()))
	}
	if snap.skipNumDir != 0 {
		strList = append(strList, fmt.Sprintf("skip %d directory", snap.skipNumDir))
	}

	if len(strList) == 0 {
		return ""
	}
	return fmt.Sprintf("(%s)", strings.Join(strList, ", "))
}

func (cppm *CpProcessMonitor) getSizeDetail(snap *CpProcessMonitorSnap) string {
	if snap.skipSize == 0 {
		return fmt.Sprintf(", Transfer size: %s", getSizeString(snap.transferSize))
	}
	if snap.transferSize == 0 {
		return fmt.Sprintf(", Skip size: %s", getSizeString(snap.skipSize))
	}
	return fmt.Sprintf(", OK size: %s(transfer: %s, skip: %s)", getSizeString(snap.transferSize+snap.skipSize), getSizeString(snap.transferSize), getSizeString(snap.skipSize))
}

func (cppm *CpProcessMonitor) getSkipSize(snap *CpProcessMonitorSnap) string {
	if snap.skipSize != 0 {
		return fmt.Sprintf(", Skip size: %s", getSizeString(snap.skipSize))
	}
	return ""
}

func (cppm *CpProcessMonitor) getDealSizeDetail(snap *CpProcessMonitorSnap) string {
	return fmt.Sprintf(", OK size: %s", getSizeString(snap.dealSize))
}

func (cppm *CpProcessMonitor) getSpeed(snap *CpProcessMonitorSnap) string {
	speed := (float64(snap.incrementSize)) / (float64(snap.duration) * 1e-9)
	return formatBytes(speed)
}

func (cppm *CpProcessMonitor) getPrecent(snap *CpProcessMonitorSnap) float64 {
	if cppm.seekAheadEnd && cppm.seekAheadError == nil {
		if cppm.TotalSize != 0 {
			return float64((snap.dealSize)*100.0) / float64(cppm.TotalSize)
		}
		if cppm.totalNum != 0 {
			return float64((snap.dealNum)*100.0) / float64(cppm.totalNum)
		}
		return 100
	}
	return 0
}

func (cppm *CpProcessMonitor) getOPStr() string {
	switch cppm.op {
	case CpTypeUpload:
		return "upload"
	case CpTypeDownload:
		return "download"
	default:
		return "cper"
	}
}

func (cppm *CpProcessMonitor) getSubject() string {
	switch cppm.op {
	case CpTypeUpload:
		return "files"
	default:
		return "objects"
	}
}
