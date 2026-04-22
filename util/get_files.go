package util

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	logger "github.com/sirupsen/logrus"
)

var once sync.Once

// recordSkipSymlink 记录因 stat 失败（悬空 / 权限不足等）而被跳过的 symlink：
//   1. 终端 warning 日志
//   2. 写入 error.report（始终写入，体例与 download/delete 等保持一致）
//   3. 写入 process.log（受 --process-log 开关控制）
func recordSkipSymlink(fpath string, cause error, fo *FileOperations) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf("[%s] skip symlink %s , errMsg:%s\n", ts, fpath, cause.Error())
	logger.Warningf("skip symlink %s: %s", fpath, cause.Error())
	if fo != nil && fo.ErrOutput != nil {
		writeError(msg, fo)
	}
	if fo != nil && fo.ProcessLogger != nil {
		writeProcessLog(msg, fo)
	}
}

func fileStatistic(localPath string, fo *FileOperations) {
	f, err := os.Stat(localPath)
	if err != nil {
		fo.Monitor.setScanError(err)
		return
	}
	if f.IsDir() {
		if !strings.HasSuffix(localPath, string(os.PathSeparator)) {
			localPath += string(os.PathSeparator)
		}

		err := getFileListStatistic(localPath, fo)
		if err != nil {
			fo.Monitor.setScanError(err)
			return
		}
	} else {
		fo.Monitor.updateScanSizeNum(f.Size(), 1)
	}

	fo.Monitor.setScanEnd()
	freshProgress()
}

func getFileListStatistic(dpath string, fo *FileOperations) error {
	if fo.Operation.OnlyCurrentDir {
		return getCurrentDirFilesStatistic(dpath, fo)
	}

	name := dpath
	symlinkDiretorys := []string{dpath}
	walkFunc := func(fpath string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}

		realFileSize := f.Size()
		dpath = filepath.Clean(dpath)
		fpath = filepath.Clean(fpath)
		fileName, err := filepath.Rel(dpath, fpath)
		if err != nil {
			return fmt.Errorf("list file error: %s, info: %s", fpath, err.Error())
		}

		if f.IsDir() {
			if fpath != dpath {
				if matchPatterns(filepath.Join(dpath, fileName), fo.Operation.Filters) {
					fo.Monitor.updateScanNum(1)
				}
			}
			return nil
		}

		if fo.Operation.DisableAllSymlink && (f.Mode()&os.ModeSymlink) != 0 {
			return nil
		}

		// 处理软链文件或文件夹
		if f.Mode()&os.ModeSymlink != 0 {

			realInfo, err := os.Stat(fpath)
			if err != nil {
				// 悬空 symlink 或权限问题：降级为 warning + 跳过，不中断整个遍历；
				// 同时写入 error.report / process.log，便于排查
				recordSkipSymlink(fpath, err, fo)
				return nil
			}

			if realInfo.IsDir() {
				realFileSize = 0
			} else {
				realFileSize = realInfo.Size()
			}

			if fo.Operation.EnableSymlinkDir && realInfo.IsDir() {
				// 软链文件夹，如果有"/"后缀，os.Lstat 将判断它是一个目录
				if !strings.HasSuffix(name, string(os.PathSeparator)) {
					name += string(os.PathSeparator)
				}
				linkDir := name + fileName + string(os.PathSeparator)
				symlinkDiretorys = append(symlinkDiretorys, linkDir)
				return nil
			}
		}
		if matchPatterns(filepath.Join(dpath, fileName), fo.Operation.Filters) {
			fo.Monitor.updateScanSizeNum(realFileSize, 1)
		}
		return nil
	}

	var err error
	for {
		symlinks := symlinkDiretorys
		symlinkDiretorys = []string{}
		for _, v := range symlinks {
			err = filepath.Walk(v, walkFunc)
			if err != nil {
				return err
			}
		}
		if len(symlinkDiretorys) == 0 {
			break
		}
	}
	return err
}

func getCurrentDirFilesStatistic(dpath string, fo *FileOperations) error {
	if !strings.HasSuffix(dpath, string(os.PathSeparator)) {
		dpath += string(os.PathSeparator)
	}

	fileList, err := ioutil.ReadDir(dpath)
	if err != nil {
		return err
	}

	for _, fileInfo := range fileList {
		if !fileInfo.IsDir() {
			// P6 修复：only-current-dir 模式下也尊重 DisableAllSymlink 开关
			if fo.Operation.DisableAllSymlink && (fileInfo.Mode()&os.ModeSymlink) != 0 {
				continue
			}

			fullPath := dpath + fileInfo.Name()
			realInfo, errF := os.Stat(fullPath)
			if errF != nil {
				// 悬空 symlink 或其他 stat 错误：降级为 warning + 跳过
				if (fileInfo.Mode() & os.ModeSymlink) != 0 {
					recordSkipSymlink(fullPath, errF, fo)
					continue
				}
				// 非 symlink 的 stat 错误仍沿用原有行为（被下方统计覆盖）
			} else if realInfo.IsDir() {
				// 指向目录的 symlink：only-current-dir 语义下不展开
				continue
			}

			// P1/P2 修复：symlink 文件取真实目标文件 size
			fileSize := fileInfo.Size()
			if errF == nil && (fileInfo.Mode()&os.ModeSymlink) != 0 {
				fileSize = realInfo.Size()
			}

			if matchPatterns(filepath.Join(dpath, fileInfo.Name()), fo.Operation.Filters) {
				fo.Monitor.updateScanSizeNum(fileSize, 1)
			}
		}
	}
	return nil
}

func generateFileList(localPath string, chFiles chan<- fileInfoType, chListError chan<- error, fo *FileOperations) {
	defer close(chFiles)
	f, err := os.Stat(localPath)
	if err != nil {
		chListError <- err
		return
	}
	if f.IsDir() {
		if !strings.HasSuffix(localPath, string(os.PathSeparator)) {
			localPath += string(os.PathSeparator)
		}

		err := getFileList(localPath, chFiles, fo)
		if err != nil {
			chListError <- err
			return
		}
	} else {
		dir, fname := filepath.Split(localPath)
		chFiles <- fileInfoType{filePath: fname, dir: dir, size: f.Size(), isDir: f.IsDir()}
	}
	chListError <- nil
}

func getFileList(dpath string, chFiles chan<- fileInfoType, fo *FileOperations) error {
	if fo.Operation.OnlyCurrentDir {
		return getCurrentDirFileList(dpath, chFiles, fo)
	}

	name := dpath
	symlinkDiretorys := []string{dpath}
	walkFunc := func(fpath string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}

		realFileSize := f.Size()
		dpath = filepath.Clean(dpath)
		fpath = filepath.Clean(fpath)
		fileName, err := filepath.Rel(dpath, fpath)
		if err != nil {
			return fmt.Errorf("list file error: %s, info: %s", fpath, err.Error())
		}

		if f.IsDir() {
			if fpath != dpath {
				if matchPatterns(filepath.Join(dpath, fileName), fo.Operation.Filters) {
					if strings.HasSuffix(fileName, "\\") || strings.HasSuffix(fileName, "/") {
						chFiles <- fileInfoType{filePath: fileName, dir: name, size: 0, lastModified: f.ModTime().Unix(), isDir: f.IsDir()}
					} else {
						chFiles <- fileInfoType{filePath: fileName + string(os.PathSeparator), dir: name, size: 0, lastModified: f.ModTime().Unix(), isDir: f.IsDir()}
					}
				}
			}
			return nil
		}

		if fo.Operation.DisableAllSymlink && (f.Mode()&os.ModeSymlink) != 0 {
			return nil
		}

		// P1/P2/P5 修复：对所有 symlink 统一取真实 size，并将 stat 错误降级为 warning
		if f.Mode()&os.ModeSymlink != 0 {
			realInfo, err := os.Stat(fpath)
			if err != nil {
				// 悬空 symlink 或权限问题：降级为 warning + 跳过，不中断整个遍历；
				// 同时写入 error.report / process.log，便于排查
				recordSkipSymlink(fpath, err, fo)
				return nil
			}

			if realInfo.IsDir() {
				realFileSize = 0
				if fo.Operation.EnableSymlinkDir {
					if !strings.HasSuffix(name, string(os.PathSeparator)) {
						name += string(os.PathSeparator)
					}
					linkDir := name + fileName + string(os.PathSeparator)
					symlinkDiretorys = append(symlinkDiretorys, linkDir)
					return nil
				}
				// 未启用 EnableSymlinkDir：不展开 symlink 目录，跳过
				return nil
			}
			// symlink 文件：取真实目标文件的 size
			realFileSize = realInfo.Size()
		}

		if matchPatterns(filepath.Join(dpath, fileName), fo.Operation.Filters) {
			chFiles <- fileInfoType{filePath: fileName, dir: name, size: realFileSize, lastModified: f.ModTime().Unix(), isDir: f.IsDir()}
		}
		return nil
	}

	var err error
	for {
		symlinks := symlinkDiretorys
		symlinkDiretorys = []string{}
		for _, v := range symlinks {
			err = filepath.Walk(v, walkFunc)
			if err != nil {
				return err
			}
		}
		if len(symlinkDiretorys) == 0 {
			break
		}
	}
	return err
}

func getCurrentDirFileList(dpath string, chFiles chan<- fileInfoType, fo *FileOperations) error {
	if !strings.HasSuffix(dpath, string(os.PathSeparator)) {
		dpath += string(os.PathSeparator)
	}

	fileList, err := ioutil.ReadDir(dpath)
	if err != nil {
		return err
	}

	for _, fileInfo := range fileList {
		if !fileInfo.IsDir() {
			// P6 修复：only-current-dir 模式下也尊重 DisableAllSymlink 开关
			if fo.Operation.DisableAllSymlink && (fileInfo.Mode()&os.ModeSymlink) != 0 {
				continue
			}

			fullPath := dpath + fileInfo.Name()
			realInfo, errF := os.Stat(fullPath)
			if errF != nil {
				// P5 修复：悬空 symlink 降级为 warning + 跳过
				if (fileInfo.Mode() & os.ModeSymlink) != 0 {
					recordSkipSymlink(fullPath, errF, fo)
					continue
				}
			} else if realInfo.IsDir() {
				// for symlink 指向目录的情况：only-current-dir 语义下不展开
				continue
			}

			// P1/P2 修复：symlink 文件取真实目标文件 size
			fileSize := fileInfo.Size()
			lastMod := fileInfo.ModTime().Unix()
			if errF == nil && (fileInfo.Mode()&os.ModeSymlink) != 0 {
				fileSize = realInfo.Size()
				lastMod = realInfo.ModTime().Unix()
			}

			if matchPatterns(filepath.Join(dpath, fileInfo.Name()), fo.Operation.Filters) {
				chFiles <- fileInfoType{filePath: fileInfo.Name(), dir: dpath, size: fileSize, lastModified: lastMod, isDir: fileInfo.IsDir()}
			}
		}
	}
	return nil
}

func getLocalFileKeys(fileUrl StorageUrl, keys map[string]commonInfoType, fo *FileOperations, objType string) error {
	strPath := fileUrl.ToString()
	if !strings.HasSuffix(strPath, string(os.PathSeparator)) {
		strPath += string(os.PathSeparator)
	}

	chFiles := make(chan fileInfoType, ChannelSize)
	chFinish := make(chan error, 2)
	go ReadLocalFileKeys(chFiles, chFinish, keys, fo, objType)
	go GetFileList(strPath, chFiles, chFinish, fo)
	select {
	case err := <-chFinish:
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadLocalFileKeys 读取本地文件keys
func ReadLocalFileKeys(chFiles <-chan fileInfoType, chFinish chan<- error, keys map[string]commonInfoType, fo *FileOperations, objType string) {

	results := make(chan commonInfoType, 1000)
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU()*2; i++ { // 根据CPU核心数动态调整
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fileInfo := range chFiles {
				select {
				case results <- commonInfoType{key: fileInfo.filePath, dir: fileInfo.dir, size: fileInfo.size, lastModifiedUnix: fileInfo.lastModified, isDir: fileInfo.isDir}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// 3. 启动结果收集器
	go func() {
		defer close(done)
		totalCount := 0
		lastReport := time.Now()
		batchSize := 1000 // 批量处理大小
		batch := make([]commonInfoType, 0, batchSize)

		for res := range results {
			totalCount++
			batch = append(batch, res)

			if len(batch) >= batchSize || time.Since(lastReport) > 100*time.Millisecond {
				for _, item := range batch {
					keys[item.key] = item
				}
				batch = batch[:0] // 重置批次

				if objType == TypeSrc {
					fo.SyncDeleteObjectInfo.srcCount = totalCount
				} else {
					fo.SyncDeleteObjectInfo.destCount = totalCount
				}
				lastReport = time.Now()

				// 检查数量限制
				if len(keys) > MaxSyncNumbers {
					cancel() // 取消所有工作
					chFinish <- fmt.Errorf("over max sync numbers %d", MaxSyncNumbers)
					return
				}
			}
		}

		// 处理剩余批次
		for _, item := range batch {
			keys[item.key] = item
		}

		if objType == TypeSrc {
			fo.SyncDeleteObjectInfo.srcCount = totalCount
		} else {
			fo.SyncDeleteObjectInfo.destCount = totalCount
		}
		chFinish <- nil
	}()

	// 4. 等待所有工作器完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 5. 等待结果收集完成
	<-done
}

// GetFileList 获取文件列表
func GetFileList(strPath string, chFiles chan<- fileInfoType, chFinish chan<- error, fo *FileOperations) {
	defer close(chFiles)
	err := getFileList(strPath, chFiles, fo)
	if err != nil {
		chFinish <- err
	}
}

func generateFileListByKeys(srcKeys, uploadKeys map[string]commonInfoType, chFiles chan<- fileInfoType, chListError chan<- error, fo *FileOperations) {
	defer close(chFiles)

	// 使用 WaitGroup 等待两个任务完成
	var wg sync.WaitGroup
	wg.Add(2) // 等待两个任务

	// 任务1：扫描统计（在后台执行）
	go func() {
		defer wg.Done()
		for _, v := range srcKeys {
			fo.Monitor.updateScanSizeNum(v.size, 1)
		}
		fo.Monitor.setScanEnd()
		freshProgress()
	}()

	// 任务2：发送文件信息（在后台执行）
	go func() {
		defer wg.Done()
		for k, v := range srcKeys {
			if _, exists := uploadKeys[k]; exists {
				chFiles <- fileInfoType{v.key, v.dir, v.size, v.lastModifiedUnix, v.isDir, false}
			} else {
				chFiles <- fileInfoType{v.key, v.dir, v.size, v.lastModifiedUnix, v.isDir, true}
			}
		}
		// 发送完成信号
		chListError <- nil
	}()

	// 等待两个任务完成
	wg.Wait()

}
