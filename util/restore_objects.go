package util

import (
	"context"
	"encoding/xml"
	"fmt"
	"math/rand"
	"net/url"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	logger "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

// restoreCounter 封装 restore 过程中的原子计数器，替代原包级全局变量，
// 避免多次调用间的状态残留，同时保证并发安全。
// total/done 用于进度展示：total 为需要回热的对象总数（进度分母），
// done 为已处理完成的对象数（进度分子）；scanEnd 标记 list 阶段是否结束。
type restoreCounter struct {
	succeed int64
	failed  int64
	errType int64
	total   int64
	done    int64
	scanEnd int32
}

func (c *restoreCounter) incSuccess() { atomic.AddInt64(&c.succeed, 1) }
func (c *restoreCounter) incFailed()  { atomic.AddInt64(&c.failed, 1) }
func (c *restoreCounter) incErrType() { atomic.AddInt64(&c.errType, 1) }
func (c *restoreCounter) incTotal()   { atomic.AddInt64(&c.total, 1) }
func (c *restoreCounter) incDone()    { atomic.AddInt64(&c.done, 1) }
func (c *restoreCounter) setScanEnd() { atomic.StoreInt32(&c.scanEnd, 1) }
func (c *restoreCounter) snapshot() (succeed, failed, errType int64) {
	return atomic.LoadInt64(&c.succeed), atomic.LoadInt64(&c.failed), atomic.LoadInt64(&c.errType)
}

// progressSnapshot 返回进度展示所需的总数/完成数及 list 是否结束
func (c *restoreCounter) progressSnapshot() (total, done int64, scanEnd bool) {
	return atomic.LoadInt64(&c.total), atomic.LoadInt64(&c.done), atomic.LoadInt32(&c.scanEnd) == 1
}

// restoreTask 是投递给 worker 的单个回热任务
type restoreTask struct {
	bucket string
	key    string
}

// RestoreObjects 取回cos对象
func RestoreObjects(c *cos.Client, cosUrl StorageUrl, fo *FileOperations, bucketType string) error {
	logger.Infof("Start Restore %s", cosUrl.(*CosUrl).Bucket+cosUrl.(*CosUrl).Object)

	routines := fo.Operation.Routines
	if routines <= 0 {
		routines = 1
	}

	counter := &restoreCounter{}
	taskCh := make(chan restoreTask, routines*2)
	var wg sync.WaitGroup

	// 启动进度显示 goroutine（单行刷新，类似 cp 的进度体验）
	target := fmt.Sprintf("cos://%s/%s", cosUrl.(*CosUrl).Bucket, cosUrl.(*CosUrl).Object)
	stopProgress := make(chan struct{})
	progressStopped := make(chan struct{})
	go func() {
		restoreProgressLoop(counter, target, stopProgress)
		close(progressStopped)
	}()

	// 启动 worker 池
	for i := 0; i < routines; i++ {
		wg.Add(1)
		go restoreWorker(c, taskCh, counter, fo, &wg)
	}

	var err error
	if bucketType == BucketTypeOfs {
		bucketName := cosUrl.(*CosUrl).Bucket
		prefix := cosUrl.(*CosUrl).Object
		err = produceOfsRestoreTasks(c, bucketName, prefix, fo, "", counter, taskCh)
	} else {
		err = produceCosRestoreTasks(c, cosUrl, fo, counter, taskCh)
	}

	// list 阶段结束，回热任务总数已确定，进度可切换为百分比显示
	counter.setScanEnd()

	// 无论 list 是否失败，都关闭通道让 worker 收尾，避免 goroutine 泄漏
	close(taskCh)
	wg.Wait()

	// 停止进度显示并等待其打印最终行（保证换行在汇总日志之前）
	close(stopProgress)
	<-progressStopped

	if err != nil {
		return err
	}

	absErrOutputPath, _ := filepath.Abs(fo.ErrOutput.Path)
	succeedNum, failedNum, errTypeNum := counter.snapshot()
	totalNum := succeedNum + failedNum + errTypeNum

	if failedNum > 0 {
		logger.Warningf("Restore %s completed, total num: %d,success num: %d,restore error num: %d,error type num: %d,Some objects restore failed, please check the detailed information in dir %s.\n", cosUrl.(*CosUrl).Bucket+cosUrl.(*CosUrl).Object, totalNum, succeedNum, failedNum, errTypeNum, absErrOutputPath)
	} else {
		logger.Infof("Restore %s completed,total num: %d,success num: %d,restore error num: %d,error type num: %d", cosUrl.(*CosUrl).Bucket+cosUrl.(*CosUrl).Object, totalNum, succeedNum, failedNum, errTypeNum)
	}

	return nil
}

// restoreWorker 消费 taskCh 中的对象并发起回热请求
func restoreWorker(c *cos.Client, taskCh <-chan restoreTask, counter *restoreCounter, fo *FileOperations, wg *sync.WaitGroup) {
	defer wg.Done()
	for t := range taskCh {
		resp, err := TryRestoreObject(c, t.bucket, t.key, fo.Operation.Days, fo.Operation.RestoreMode)
		if err != nil {
			if resp != nil && resp.StatusCode == 409 {
				// 对象已在回热中或已回热完成，视为成功
				counter.incSuccess()
			} else {
				counter.incFailed()
				writeError(fmt.Sprintf("restore %s failed , errMsg:%v\n", t.key, err), fo)
			}
		} else {
			counter.incSuccess()
		}
		// 完成一个回热任务，推进进度
		counter.incDone()
	}
}

// produceCosRestoreTasks list COS 对象并将需要回热的对象投递到 taskCh
func produceCosRestoreTasks(c *cos.Client, cosUrl StorageUrl, fo *FileOperations, counter *restoreCounter, taskCh chan<- restoreTask) error {
	var objects []cos.Object
	marker := ""
	isTruncated := true
	bucket := cosUrl.(*CosUrl).Bucket

	for isTruncated {
		var err error
		err, objects, _, isTruncated, marker = getCosObjectListForLs(c, cosUrl, marker, 0, true)
		if err != nil {
			return fmt.Errorf("list objects error : %v", err)
		}

		for _, object := range objects {
			if !isRestoreType(object) {
				counter.incErrType()
				continue
			}
			object.Key, _ = url.QueryUnescape(object.Key)
			if !cosObjectMatchPatterns(object.Key, fo.Operation.Filters) {
				continue
			}
			// 需要回热的对象计入进度分母
			counter.incTotal()
			if object.RestoreStatus == "ONGOING" || object.RestoreStatus == "ONGING" {
				// 已在回热中，视为已完成
				counter.incSuccess()
				counter.incDone()
				continue
			}
			taskCh <- restoreTask{bucket: bucket, key: object.Key}
		}
	}
	return nil
}

// TryRestoreObject 重试回热对象
func TryRestoreObject(c *cos.Client, bucketName, objectKey string, days int, mode string) (resp *cos.Response, err error) {

	logger.Debugf("Restore cos://%s/%s\n", bucketName, objectKey)
	opt := &cos.ObjectRestoreOptions{
		XMLName:       xml.Name{},
		Days:          days,
		Tier:          &cos.CASJobParameters{Tier: mode},
		XOptionHeader: nil,
	}

	for i := 0; i <= 10; i++ {
		resp, err = c.Object.PostRestore(context.Background(), objectKey, opt)
		if err != nil {
			if resp != nil && resp.StatusCode == 503 {
				if i == 10 {
					return resp, err
				} else {
					logger.Debugf("Error 503: Service rate limiting. Retrying...")
					waitTime := time.Duration(rand.Intn(10)+1) * time.Second
					time.Sleep(waitTime)
					continue
				}
			} else {
				return resp, err
			}
		} else {
			return resp, err
		}
	}
	return resp, err
}

// produceOfsRestoreTasks list OFS 对象（含递归子目录）并投递回热任务
func produceOfsRestoreTasks(c *cos.Client, bucketName, prefix string, fo *FileOperations, marker string, counter *restoreCounter, taskCh chan<- restoreTask) error {
	var objects []cos.Object
	var commonPrefixes []string
	isTruncated := true

	for isTruncated {
		var err error
		err, objects, commonPrefixes, isTruncated, marker = getOfsObjectListForLs(c, prefix, marker, 0, true)
		if err != nil {
			return fmt.Errorf("list objects error : %v", err)
		}

		for _, object := range objects {
			if !isRestoreType(object) {
				counter.incErrType()
				continue
			}
			object.Key, _ = url.QueryUnescape(object.Key)
			if !cosObjectMatchPatterns(object.Key, fo.Operation.Filters) {
				continue
			}
			// 需要回热的对象计入进度分母
			counter.incTotal()
			if object.RestoreStatus == "ONGOING" || object.RestoreStatus == "ONGING" {
				// 已在回热中，视为已完成
				counter.incSuccess()
				counter.incDone()
				continue
			}
			taskCh <- restoreTask{bucket: bucketName, key: object.Key}
		}

		if len(commonPrefixes) > 0 {
			for _, commonPrefix := range commonPrefixes {
				commonPrefix, _ = url.QueryUnescape(commonPrefix)
				// 递归子目录
				if err := produceOfsRestoreTasks(c, bucketName, commonPrefix, fo, "", counter, taskCh); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// 判断是否是需要回热的文件类型
func isRestoreType(object cos.Object) bool {
	if object.StorageClass == Archive || object.StorageClass == MAZArchive || object.StorageClass == DeepArchive {
		return true
	}

	// 智能分层类型需要在归档层和深度归档层才可以回热
	if object.StorageClass == IntelligentTiering || object.StorageClass == MAZIntelligentTiering {
		if object.StorageTier == StorageTierArchive || object.StorageTier == StorageTierDeepArchive {
			return true
		}
	}

	return false

}

// restoreProgressLoop 定时刷新单行回热进度，收到 stop 信号后打印最终行并返回。
// 由单个 goroutine 独占调用，打印本身无并发。
func restoreProgressLoop(c *restoreCounter, target string, stop <-chan struct{}) {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			printRestoreProgress(c, target, true)
			return
		case <-ticker.C:
			printRestoreProgress(c, target, false)
		}
	}
}

// printRestoreProgress 打印一行回热进度（final 为 true 时补换行）。
// list 未结束时总数仍在增长，展示已发现/已回热数；结束后展示百分比进度。
func printRestoreProgress(c *restoreCounter, target string, final bool) {
	total, done, scanEnd := c.progressSnapshot()
	succeed, failed, errType := c.snapshot()

	var line string
	if !scanEnd {
		line = fmt.Sprintf("Restoring %s ... found: %d, restored: %d (success: %d, failed: %d)",
			target, total, done, succeed, failed)
	} else {
		percent := 100.0
		if total > 0 {
			percent = float64(done) * 100.0 / float64(total)
		}
		line = fmt.Sprintf("Restoring %s ... total: %d, restored: %d, progress: %.1f%% (success: %d, failed: %d, skip-non-archive: %d)",
			target, total, done, percent, succeed, failed, errType)
	}

	fmt.Print(getClearStr(line))
	if final {
		fmt.Println()
	}
}
