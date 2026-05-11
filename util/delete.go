package util

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	logger "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

var fileRemoveCount int
var totalDeleteErrCount int

func getDeleteKeys(srcClient, destClient *cos.Client, srcUrl StorageUrl, destUrl StorageUrl, fo *FileOperations) (map[string]commonInfoType, map[string]commonInfoType, map[string]commonInfoType, error) {
	// 创建结果通道
	srcKeysChan := make(chan map[string]commonInfoType)
	destKeysChan := make(chan map[string]commonInfoType)
	errChan := make(chan error, 2) // 缓冲通道避免阻塞

	// 启动进度打印协程
	progressCtx, progressCancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fmt.Printf("\rProcessing source num: %d,destination num: %d", fo.SyncDeleteObjectInfo.srcCount, fo.SyncDeleteObjectInfo.destCount)
			case <-progressCtx.Done():
				return
			}
		}
	}()

	// 并发获取源端键列表
	go func() {
		keys := make(map[string]commonInfoType)
		var err error

		if srcUrl.IsFileUrl() {
			err = getLocalFileKeys(srcUrl, keys, fo, TypeSrc)
		} else {
			if fo.BucketType == BucketTypeOfs {
				err = GetOfsKeys(srcClient, srcUrl, keys, fo, TypeSrc)
			} else {
				err = GetCosKeys(srcClient, srcUrl, keys, fo, TypeSrc)
			}
		}

		if err != nil {
			errChan <- err
			return
		}
		srcKeysChan <- keys
	}()

	// 并发获取目标端键列表
	go func() {
		keys := make(map[string]commonInfoType)
		var err error

		if destUrl.IsFileUrl() {
			err = getLocalFileKeys(destUrl, keys, fo, TypeDest)
		} else {
			if fo.BucketType == BucketTypeOfs {
				err = GetOfsKeys(destClient, destUrl, keys, fo, TypeDest)
			} else {
				err = GetCosKeys(destClient, destUrl, keys, fo, TypeDest)
			}
		}

		if err != nil {
			errChan <- err
			return
		}
		destKeysChan <- keys
	}()

	// 等待结果
	var srcKeys, destKeys map[string]commonInfoType
	var err error

	// 收集结果和错误
	for i := 0; i < 2; i++ {
		select {
		case keys := <-srcKeysChan:
			srcKeys = keys
		case keys := <-destKeysChan:
			destKeys = keys
		case e := <-errChan:
			if err == nil {
				err = e
			}
		}
	}

	// 如果有错误，提前返回
	if err != nil {
		return nil, nil, nil, err
	}

	// 取完列表后终止进度打印，并输出最终结果
	progressCancel()
	fmt.Printf("\r\033[KTotal source num: %d,destination num: %d", fo.SyncDeleteObjectInfo.srcCount, fo.SyncDeleteObjectInfo.destCount)

	delKeys := make(map[string]commonInfoType)
	for k, v := range destKeys {
		delKeys[k] = v
	}

	transferKeys := make(map[string]commonInfoType)
	for k, v := range srcKeys {
		transferKeys[k] = v
	}

	// 根据操作系统和操作类型筛选出需要删除的对象或文件
	isLinux := (string(os.PathSeparator) == "/")
	for k := range srcKeys {
		if isLinux || fo.CpType == CpTypeCopy {
			delete(delKeys, k)
		} else if fo.CpType == CpTypeUpload {
			delete(delKeys, strings.Replace(k, "\\", "/", -1))
		} else {
			delete(delKeys, strings.Replace(k, "/", "\\", -1))
		}
	}

	// sync 语义：本地目录只要在 COS 端有任意文件以其为前缀，就视为"一致"，不应被列为待删。
	// 由于 COS list 不会返回隐式目录条目，这里需要根据 srcKeys 中每个文件 key 的所有
	// 父目录前缀，再次从 delKeys 中剔除对应的本地目录条目（destKeys 中目录 key 以分隔符结尾）。
	// 仅对目的端为本地的场景（download/copy 到本地）有意义。
	if destUrl.IsFileUrl() {
		pruneParentDirsFromDelKeys(srcKeys, delKeys, fo.CpType, isLinux)
	}

	// 根据操作系统和操作类型筛选出需要传输的对象或文件
	if fo.Operation.IgnoreExisting || fo.Operation.Update {
		for k, v := range destKeys {
			// 源端不存在，目的端存在
			if _, exists := transferKeys[k]; !exists {
				continue
			}
			shouldDelete := false

			if fo.Operation.IgnoreExisting {
				// 启用跳过已存在的文件
				shouldDelete = true
			} else if fo.Operation.Update && transferKeys[k].lastModifiedUnix <= v.lastModifiedUnix {
				// 未启用跳过但启用更新时间检查，且源文件不新于目标文件
				shouldDelete = true
			}

			if shouldDelete {
				if isLinux || fo.CpType == CpTypeCopy {
					delete(transferKeys, k)
				} else if fo.CpType == CpTypeUpload {
					delete(transferKeys, strings.Replace(k, "/", "\\", -1))
				} else {
					delete(transferKeys, strings.Replace(k, "\\", "/", -1))
				}
			}

		}
	}

	// 输出统计信息
	if destUrl.IsFileUrl() {
		fmt.Printf("\nfile(directory) will be removed count:%d\n", len(delKeys))
	} else {
		fmt.Printf("\nobject will be deleted count:%d\n", len(delKeys))
	}

	return srcKeys, delKeys, transferKeys, nil
}

// pruneParentDirsFromDelKeys 根据 srcKeys 中每个文件 key 的所有父目录前缀，
// 从 delKeys 中删除对应的目录条目。
// 用于 sync --delete 下载/拷贝到本地的场景：本地的目录条目只要在 COS 端有任意
// 对象以其为前缀，就视为目录"一致"，不应被列为待删（避免每次 sync 都误报本地目录待删）。
//
// destSep 选取规则：
//   - Linux 或 COS 之间拷贝（CpTypeCopy）：本地/目的端 key 使用 '/'
//   - Windows 下 CpTypeDownload：本地 key 使用 '\\'，需将 srcKeys 中的 '/' 转为 '\\'
func pruneParentDirsFromDelKeys(srcKeys, delKeys map[string]commonInfoType, cpType CpType, isLinux bool) {
	var destSep string
	useNativeSep := !isLinux && cpType != CpTypeCopy
	if useNativeSep {
		destSep = "\\"
	} else {
		destSep = "/"
	}

	for k := range srcKeys {
		localKey := k
		if useNativeSep {
			localKey = strings.Replace(k, "/", destSep, -1)
		}
		// 逐级剥离父目录前缀（保留末尾分隔符），从 delKeys 中删除
		idx := strings.LastIndex(localKey, destSep)
		for idx > 0 {
			dirKey := localKey[:idx+1]
			delete(delKeys, dirKey)
			idx = strings.LastIndex(localKey[:idx], destSep)
		}
	}
}

func deleteKeys(c *cos.Client, keysToDelete map[string]commonInfoType, destUrl StorageUrl, fo *FileOperations) error {
	// 根据类型区分删除cos上的对象还是本地文件
	if fo.CpType == CpTypeCopy || fo.CpType == CpTypeUpload {
		err := DeleteCosObjects(c, keysToDelete, destUrl, fo)
		return err
	} else {
		err := DeleteLocalFiles(keysToDelete, destUrl, fo)
		return err
	}

	return nil
}

// DeleteCosObjects deletes multiple COS objects based on the provided keysToDelete map.
// It returns an error if any of the operations fail.
func DeleteCosObjects(c *cos.Client, keysToDelete map[string]commonInfoType, cosUrl StorageUrl, fo *FileOperations) error {

	errCount := 0
	objects := []cos.Object{}
	for k, v := range keysToDelete {
		if len(objects) >= MaxDeleteBatchCount {
			if confirm(objects, fo, cosUrl) {
				opt := &cos.ObjectDeleteMultiOptions{
					Objects: objects,
					// 布尔值，这个值决定了是否启动 Quiet 模式
					// 值为 true 启动 Quiet 模式，值为 false 则启动 Verbose 模式，默认值为 false
					Quiet: true,
				}
				res, _, err := c.Object.DeleteMulti(context.Background(), opt)
				if err != nil {
					return err
				}
				// 删除失败的记录写入错误日志
				if fo.Operation.FailOutput {
					for _, delErr := range res.Errors {
						fo.DeleteCount--
						errCount++
						totalDeleteErrCount++
						writeError(fmt.Sprintf("delete %s failed , code:%s,errMsg:%s\n", delErr.Key, delErr.Code, delErr.Message), fo)
					}
				}
			}
			objects = []cos.Object{}
			fo.DeleteCount += MaxDeleteBatchCount
			if errCount > 0 {
				fmt.Printf("\rdelete object count:%d, err count:%d", fo.DeleteCount, errCount)
			} else {
				fmt.Printf("\rdelete object count:%d", fo.DeleteCount)
			}

		}

		objects = append(objects, cos.Object{Key: v.dir + k})
	}

	if len(objects) > 0 && confirm(objects, fo, cosUrl) {
		opt := &cos.ObjectDeleteMultiOptions{
			Objects: objects,
			// 布尔值，这个值决定了是否启动 Quiet 模式
			// 值为 true 启动 Quiet 模式，值为 false 则启动 Verbose 模式，默认值为 false
			Quiet: true,
		}
		res, _, err := c.Object.DeleteMulti(context.Background(), opt)
		if err != nil {
			return err
		}
		// 删除失败的记录写入错误日志
		if fo.Operation.FailOutput {
			for _, delErr := range res.Errors {
				fo.DeleteCount--
				errCount++
				totalDeleteErrCount++
				writeError(fmt.Sprintf("delete %s failed , code:%s,errMsg:%s\n", delErr.Key, delErr.Code, delErr.Message), fo)
			}
		}

		fo.DeleteCount += len(objects)
		if errCount > 0 {
			fmt.Printf("\rdelete object count:%d, err count:%d", fo.DeleteCount, errCount)
		} else {
			fmt.Printf("\rdelete object count:%d", fo.DeleteCount)
		}
	}
	return nil
}

// DeleteCosObjectVersions deletes multiple object versions in a COS bucket.
//
// Parameters:
// - c: *cos.Client - the COS client to use
// - keysToDelete: []cos.Object - the keys of the object versions to delete
// - cosUrl: StorageUrl - the COS bucket URL
// - fo: *FileOperations - the file operations object
//
// Returns:
// - error: an error if the deletion fails, otherwise nil
func DeleteCosObjectVersions(c *cos.Client, keysToDelete []cos.Object, cosUrl StorageUrl, fo *FileOperations) error {

	errCount := 0
	objects := []cos.Object{}
	for _, v := range keysToDelete {
		if len(objects) >= MaxDeleteBatchCount {
			if confirm(objects, fo, cosUrl) {
				opt := &cos.ObjectDeleteMultiOptions{
					Objects: objects,
					// 布尔值，这个值决定了是否启动 Quiet 模式
					// 值为 true 启动 Quiet 模式，值为 false 则启动 Verbose 模式，默认值为 false
					Quiet: true,
				}
				res, _, err := c.Object.DeleteMulti(context.Background(), opt)
				if err != nil {
					return err
				}
				// 删除失败的记录写入错误日志
				if fo.Operation.FailOutput {
					for _, delErr := range res.Errors {
						fo.DeleteCount--
						errCount++
						totalDeleteErrCount++
						writeError(fmt.Sprintf("delete version %s of object %s failed , code:%s,errMsg:%s\n", delErr.VersionId, delErr.Key, delErr.Code, delErr.Message), fo)
					}
				}
			}
			objects = []cos.Object{}
			fo.DeleteCount += MaxDeleteBatchCount
			if errCount > 0 {
				fmt.Printf("\rdelete object versions count:%d, err count:%d", fo.DeleteCount, errCount)
			} else {
				fmt.Printf("\rdelete object versions count:%d", fo.DeleteCount)
			}

		}

		objects = append(objects, v)
	}

	if len(objects) > 0 && confirm(objects, fo, cosUrl) {
		opt := &cos.ObjectDeleteMultiOptions{
			Objects: objects,
			// 布尔值，这个值决定了是否启动 Quiet 模式
			// 值为 true 启动 Quiet 模式，值为 false 则启动 Verbose 模式，默认值为 false
			Quiet: true,
		}
		res, _, err := c.Object.DeleteMulti(context.Background(), opt)
		if err != nil {
			return err
		}
		// 删除失败的记录写入错误日志
		if fo.Operation.FailOutput {
			for _, delErr := range res.Errors {
				fo.DeleteCount--
				errCount++
				totalDeleteErrCount++
				writeError(fmt.Sprintf("delete version %s of object %s failed , code:%s,errMsg:%s\n", delErr.VersionId, delErr.Key, delErr.Code, delErr.Message), fo)
			}
		}

		fo.DeleteCount += len(objects)
		if errCount > 0 {
			fmt.Printf("\rdelete object versions count:%d, err count:%d", fo.DeleteCount, errCount)
		} else {
			fmt.Printf("\rdelete object versions count:%d", fo.DeleteCount)
		}
	}
	return nil
}

func confirm(objects []cos.Object, fo *FileOperations, cosUrl StorageUrl) bool {
	if fo.Operation.Force {
		return true
	}

	var logBuffer bytes.Buffer
	logBuffer.WriteString("\n")
	for _, v := range objects {
		if fo.Command == CommandRm && fo.Operation.AllVersions {
			logBuffer.WriteString(fmt.Sprintf("version %s of %s\n", v.VersionId, SchemePrefix+cosUrl.(*CosUrl).Bucket+CosSeparator+v.Key))
		} else {
			logBuffer.WriteString(fmt.Sprintf("%s\n", SchemePrefix+cosUrl.(*CosUrl).Bucket+CosSeparator+v.Key))
		}

	}
	if fo.Command == CommandSync {
		logBuffer.WriteString(fmt.Sprintf("sync:delete above objects(Y or N)? "))
	} else {
		if fo.Command == CommandRm && fo.Operation.AllVersions {
			logBuffer.WriteString(fmt.Sprintf("delete above object versions(Y or N)? "))
		} else {
			logBuffer.WriteString(fmt.Sprintf("delete above objects(Y or N)? "))
		}

	}
	fmt.Printf(logBuffer.String())

	var val string
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func confirmOfs(prefix string, fo *FileOperations, cosUrl StorageUrl) bool {
	if fo.Operation.Force {
		return true
	}

	var logBuffer bytes.Buffer
	logBuffer.WriteString("\n")
	if prefix == "" {
		logBuffer.WriteString(fmt.Sprintf("Do you want to delete all the objects in the %s bucket? ", cosUrl.(*CosUrl).Bucket))
	} else {
		logBuffer.WriteString(fmt.Sprintf("Do you want to delete all the objects under the %s path? ", prefix))
	}

	fmt.Printf(logBuffer.String())

	var val string
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

// DeleteLocalFiles 删除本地文件
func DeleteLocalFiles(keysToDelete map[string]commonInfoType, fileUrl StorageUrl, fo *FileOperations) error {
	var sortList []string
	for key, _ := range keysToDelete {
		sortList = append(sortList, key)
	}
	// 排序，先删除文件后删除文件夹
	sort.Sort(sort.Reverse(sort.StringSlice(sortList)))

	absDirName, err := getAbsPath(fileUrl.ToString())
	if err != nil {
		return err
	}

	nowFatherDirName := ""
	for _, key := range sortList {
		if strings.HasSuffix(key, string(os.PathSeparator)) {
			dirName := key[0 : len(key)-1]
			readerInfos, _ := getDirFiles(absDirName+dirName, 10)

			if len(readerInfos) > 0 {
				continue
			} else {
				// 获取备份路径
				f, err := os.Stat(fo.Operation.BackupDir + dirName)
				if err != nil {
					// 嵌套目录场景下，BackupDir 中对应的父目录可能尚未创建，
					// 直接 os.Rename 会失败（"no such file or directory"），
					// 因此先确保父目录存在再 move。
					backupParent := fo.Operation.BackupDir + dirName
					if idx := strings.LastIndex(dirName, string(os.PathSeparator)); idx >= 0 {
						backupParent = fo.Operation.BackupDir + dirName[:idx]
					} else {
						// dirName 没有分隔符（顶层目录），父目录就是 BackupDir 本身
						backupParent = strings.TrimRight(fo.Operation.BackupDir, string(os.PathSeparator))
					}
					if mkErr := os.MkdirAll(backupParent, 0755); mkErr != nil {
						return fmt.Errorf("create backup parent dir %s error: %s", backupParent, mkErr.Error())
					}
					if mvErr := movePath(absDirName+dirName, fo.Operation.BackupDir+dirName); mvErr != nil {
						return mvErr
					}
				} else {
					if !f.IsDir() {
						return fmt.Errorf("backup %s is already exist,but is file", fo.Operation.BackupDir+dirName)
					} else {
						// 文件夹里面内容已被删完，则删除文件夹
						os.RemoveAll(absDirName + dirName)
					}
				}
			}
		} else {
			fatherDir := absDirName
			index := strings.LastIndex(key, string(os.PathSeparator))
			if index >= 0 {
				fatherDir = key[:index]
			}

			if fatherDir != nowFatherDirName && fatherDir != absDirName {
				os.MkdirAll(fo.Operation.BackupDir+fatherDir, 0755)
				nowFatherDirName = fatherDir
			}

			err := movePath(absDirName+key, fo.Operation.BackupDir+key)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CheckBackupDir todo
func CheckBackupDir(fileUrl StorageUrl, fo *FileOperations) error {
	createDir := false
	f, err := os.Stat(fileUrl.ToString())
	if err != nil {
		if err := os.MkdirAll(fileUrl.ToString(), 0755); err != nil {
			return err
		}
		createDir = true
	} else if !f.IsDir() {
		return fmt.Errorf("dest dir %s is file,is not directory", fileUrl.ToString())
	}

	if createDir && fo.Operation.BackupDir == "" {
		return nil
	}

	if fo.Operation.BackupDir == "" {
		return fmt.Errorf("files backup dir is empty string,please use --backup-dir")
	}

	if !strings.HasSuffix(fo.Operation.BackupDir, string(os.PathSeparator)) {
		fo.Operation.BackupDir += string(os.PathSeparator)
	}

	// 检查备份路径是否是目标文件路径的子路径
	absFileDir, err := getAbsPath(fileUrl.ToString())
	if err != nil {
		return err
	}

	absBackupDir, err := getAbsPath(fo.Operation.BackupDir)
	if err != nil {
		return err
	}

	if strings.Index(absBackupDir, absFileDir) >= 0 {
		return fmt.Errorf("files backup dir %s is subdirectory of %s", fo.Operation.BackupDir, fileUrl.ToString())
	}

	f, err = os.Stat(fo.Operation.BackupDir)
	if err != nil {
		if err := os.MkdirAll(fo.Operation.BackupDir, 0755); err != nil {
			return err
		}
	} else if !f.IsDir() {
		return fmt.Errorf("files backup dir %s is file,is not directory", fo.Operation.BackupDir)
	}
	return nil
}

func getDirFiles(dirName string, limitCount int) ([]os.FileInfo, error) {
	f, err := os.Open(dirName)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(limitCount)
	f.Close()
	if err != nil {
		return nil, err
	}
	return list, nil
}

func movePath(srcName, destName string) error {
	err := moveFileToPath(srcName, destName)
	if err != nil {
		return fmt.Errorf("rename %s %s error,%s\n", srcName, destName, err.Error())
	} else {
		fileRemoveCount += 1
		fmt.Printf("\rremove file(directory) count:%d", fileRemoveCount)
	}
	return err
}

func moveFileToPath(srcName, destName string) error {
	err := os.Rename(srcName, destName)
	if err == nil {
		return nil
	}

	// Rename 失败时（例如 Windows 上跨卷移动）回退到 copy + remove。
	// 注意：不能用 defer 延迟关闭，否则在 Windows 上 os.Remove(srcName) 会因为
	// 源文件仍被当前进程打开而失败："The process cannot access the file because
	// it is being used by another process."。必须在 Remove 之前显式 Close。
	inputFile, err := os.Open(srcName)
	if err != nil {
		return err
	}

	outputFile, err := os.Create(destName)
	if err != nil {
		inputFile.Close()
		return err
	}

	if _, err = io.Copy(outputFile, inputFile); err != nil {
		inputFile.Close()
		outputFile.Close()
		// 拷贝失败时清理可能已生成的目标文件，避免残留半成品
		_ = os.Remove(destName)
		return err
	}

	// 显式关闭源/目标文件，确保后续 Remove 在 Windows 上不会被自身句柄占用
	if err = inputFile.Close(); err != nil {
		outputFile.Close()
		return err
	}
	if err = outputFile.Close(); err != nil {
		return err
	}

	return os.Remove(srcName)
}

// RemoveObjects 删除cos对象
func RemoveObjects(args []string, fo *FileOperations) error {
	for _, arg := range args {

		cosUrl, err := FormatUrl(arg)
		if err != nil {
			return fmt.Errorf("format cosUrl error,%v", err)
		}

		bucketName := cosUrl.(*CosUrl).Bucket

		c, err := NewClient(fo.Config, fo.Param, bucketName)
		if err != nil {
			return err
		}

		if fo.Operation.AllVersions {
			res, _, err := GetBucketVersioning(c)
			if err != nil {
				return err
			}
			if res.Status != VersionStatusEnabled {
				return fmt.Errorf("versioning is not enabled on the src bucket")
			}
			logger.Infof("Start remove prefix %s all versions", getCosUrl(cosUrl.(*CosUrl).Bucket, cosUrl.(*CosUrl).Object))
		} else {
			logger.Infof("Start remove prefix %s", getCosUrl(cosUrl.(*CosUrl).Bucket, cosUrl.(*CosUrl).Object))
		}

		bucketType, err := GetBucketType(c, fo.Param, fo.Config, bucketName)
		if err != nil {
			return err
		}

		// 打印一个空行
		fmt.Println()

		if bucketType == BucketTypeOfs {
			prefix := cosUrl.(*CosUrl).Object

			if len(fo.Operation.Filters) == 0 {
				if confirmOfs(prefix, fo, cosUrl) {
					// 若不筛选路径，则直接使用?recursive 方式直接删除路径下所有内容
					err = RemoveOfsObjectsRecursive(c, prefix)
				} else {
					logger.Info("Cancel deletion")
				}
			} else {
				err = RemoveOfsObjects("", c, cosUrl, prefix, fo)
			}
		} else {
			if fo.Operation.AllVersions {
				err = RemoveCosObjectVersions(c, cosUrl, fo)
			} else {
				err = RemoveCosObjects("", c, cosUrl, fo)
			}

		}

		if err != nil {
			return err
		}
		// 打印一个空行
		fmt.Println()

		if fo.Operation.AllVersions {
			logger.Infof("Remove prefix %s all versions completed", getCosUrl(cosUrl.(*CosUrl).Bucket, cosUrl.(*CosUrl).Object))
		} else {
			logger.Infof("Remove prefix %s completed", getCosUrl(cosUrl.(*CosUrl).Bucket, cosUrl.(*CosUrl).Object))
		}

	}

	if totalDeleteErrCount > 0 && fo.Operation.FailOutput {
		absErrOutputPath, _ := filepath.Abs(fo.ErrOutput.Path)

		if fo.Operation.AllVersions {
			logger.Infof("Some object versions remove failed, please check the detailed information in dir %s.\n", absErrOutputPath)
		} else {
			logger.Infof("Some objects remove failed, please check the detailed information in dir %s.\n", absErrOutputPath)
		}
	}
	// 打印一个空行
	fmt.Println()

	return nil
}

// RemoveOfsObjectsRecursive 删除ofs对象
func RemoveOfsObjectsRecursive(c *cos.Client, prefix string) error {
	query := &url.Values{}
	query.Add("recursive", "")
	opt := &cos.ObjectDeleteOptions{
		XOptionQuery: query,
	}
	isTruncated := true
	marker := ""
	var err error
	var objects []cos.Object
	if prefix == "" {
		for isTruncated {
			var commonPrefixes []string
			err, objects, commonPrefixes, isTruncated, marker = getOfsObjectListForLs(c, prefix, marker, 0, false)

			if err != nil {
				return fmt.Errorf("list objects error : %v", err)
			}

			for _, object := range objects {
				key, _ := url.QueryUnescape(object.Key)
				_, err = c.Object.Delete(context.Background(), key)
			}

			if len(commonPrefixes) > 0 {
				for _, commonPrefix := range commonPrefixes {
					commonPrefix, _ = url.QueryUnescape(commonPrefix)
					_, err = c.Object.Delete(context.Background(), commonPrefix, opt)
				}
			}
		}
	} else {
		_, err = c.Object.Delete(context.Background(), prefix, opt)
	}
	return err
}

// RemoveOfsObjects 删除ofs对象
func RemoveOfsObjects(marker string, c *cos.Client, cosUrl StorageUrl, prefix string, fo *FileOperations) error {
	var err error
	isTruncated := true
	var objects []cos.Object
	var keysToDelete map[string]commonInfoType

	for isTruncated {
		var commonPrefixes []string
		err, objects, commonPrefixes, isTruncated, marker = getOfsObjectListForLs(c, prefix, marker, 0, true)

		if err != nil {
			return fmt.Errorf("list objects error : %v", err)
		}

		keysToDelete = make(map[string]commonInfoType)
		for _, object := range objects {
			key, _ := url.QueryUnescape(object.Key)
			if cosObjectMatchPatterns(key, fo.Operation.Filters) {
				objPrefix := ""
				objKey := key
				index := strings.LastIndex(cosUrl.(*CosUrl).Object, "/")
				if index > 0 {
					objPrefix = key[:index+1]
					objKey = key[index+1:]
				}
				keysToDelete[objKey] = commonInfoType{key: objKey, dir: objPrefix}
			}
		}
		err = DeleteCosObjects(c, keysToDelete, cosUrl, fo)
		if err != nil {
			return err
		}

		if len(commonPrefixes) > 0 {
			for _, commonPrefix := range commonPrefixes {
				commonPrefix, _ = url.QueryUnescape(commonPrefix)
				err = RemoveOfsObjects("", c, cosUrl, commonPrefix, fo)
				if err != nil {
					return err
				}
			}

			keysToDelete = make(map[string]commonInfoType)
			for _, commonPrefix := range commonPrefixes {
				key, _ := url.QueryUnescape(commonPrefix)
				if cosObjectMatchPatterns(key, fo.Operation.Filters) {
					objPrefix := ""
					objKey := key
					index := strings.LastIndex(cosUrl.(*CosUrl).Object, "/")
					if index > 0 {
						objPrefix = key[:index+1]
						objKey = key[index+1:]
					}
					keysToDelete[objKey] = commonInfoType{key: objKey, dir: objPrefix}
				}
			}
			err = DeleteCosObjects(c, keysToDelete, cosUrl, fo)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// RemoveCosObjects 删除cos对象
func RemoveCosObjects(marker string, c *cos.Client, cosUrl StorageUrl, fo *FileOperations) error {
	var err error
	var objects []cos.Object
	isTruncated := true
	for isTruncated {
		err, objects, _, isTruncated, marker = getCosObjectListForLs(c, cosUrl, marker, 0, true)

		if err != nil {
			return fmt.Errorf("list objects error : %v", err)
		}

		keysToDelete := make(map[string]commonInfoType)
		for _, object := range objects {
			object.Key, _ = url.QueryUnescape(object.Key)
			if cosObjectMatchPatterns(object.Key, fo.Operation.Filters) {
				objPrefix := ""
				objKey := object.Key
				index := strings.LastIndex(cosUrl.(*CosUrl).Object, "/")
				if index > 0 {
					objPrefix = object.Key[:index+1]
					objKey = object.Key[index+1:]
				}
				keysToDelete[objKey] = commonInfoType{key: objKey, dir: objPrefix}
			}
		}

		err = DeleteCosObjects(c, keysToDelete, cosUrl, fo)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveCosObjectVersions 删除cos对象历史版本
func RemoveCosObjectVersions(c *cos.Client, cosUrl StorageUrl, fo *FileOperations) error {
	var err error
	var versions []cos.ListVersionsResultVersion
	var deleteMarkers []cos.ListVersionsResultDeleteMarker
	isTruncated := true
	var keyMarker, versionIdMarker string

	for isTruncated {
		err, versions, deleteMarkers, _, isTruncated, versionIdMarker, keyMarker = getCosObjectVersionListForLs(c, cosUrl, versionIdMarker, keyMarker, 0, true)

		if err != nil {
			return fmt.Errorf("list object versions error : %v", err)
		}

		keysToDelete := []cos.Object{}
		for _, object := range versions {
			object.Key, _ = url.QueryUnescape(object.Key)
			if cosObjectMatchPatterns(object.Key, fo.Operation.Filters) {
				keysToDelete = append(keysToDelete, cos.Object{Key: object.Key, VersionId: object.VersionId})
			}
		}

		for _, object := range deleteMarkers {
			object.Key, _ = url.QueryUnescape(object.Key)
			if cosObjectMatchPatterns(object.Key, fo.Operation.Filters) {
				keysToDelete = append(keysToDelete, cos.Object{Key: object.Key, VersionId: object.VersionId})
			}
		}

		err = DeleteCosObjectVersions(c, keysToDelete, cosUrl, fo)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveObject 删除单个对象
func RemoveObject(args []string, fo *FileOperations) error {
	for _, arg := range args {

		cosUrl, err := FormatUrl(arg)
		if err != nil {
			return fmt.Errorf("format cosUrl error,%v", err)
		}
		bucketName := cosUrl.(*CosUrl).Bucket
		cosPath := cosUrl.(*CosUrl).Object

		if cosPath == "" || strings.HasSuffix(cosPath, CosSeparator) {
			return fmt.Errorf("cosPath:%v is dir, please use --recursive option", cosPath)
		}

		c, err := NewClient(fo.Config, fo.Param, bucketName)
		if err != nil {
			return err
		}

		if fo.Operation.VersionId != "" {
			res, _, err := GetBucketVersioning(c)
			if err != nil {
				return err
			}
			if res.Status != VersionStatusEnabled {
				return fmt.Errorf("versioning is not enabled on the src bucket")
			}
		}

		// 查询对象是否存在
		fileExist, err := CheckCosObjectExist(c, cosPath, fo.Operation.VersionId)
		if err != nil {
			return err
		}
		if !fileExist {
			if fo.Operation.VersionId != "" {
				deleteMarkerExist, err := CheckDeleteMarkerExist(c, cosUrl, fo.Operation.VersionId)
				if err != nil {
					return err
				}
				if !deleteMarkerExist {
					return fmt.Errorf("cos object or version not found:%s", cosPath)
				}
			} else {
				return fmt.Errorf("cos object or version not found:%s", cosPath)
			}

		}

		// 删除指定object或其指定版本
		RemoveObjectOrVersion(c, cosUrl, fo)

	}
	return nil
}

// RemoveObjectOrVersion 删除对象单个版本
func RemoveObjectOrVersion(c *cos.Client, cosUrl StorageUrl, fo *FileOperations) error {
	var err error
	cosPath := getCosUrl(cosUrl.(*CosUrl).Bucket, cosUrl.(*CosUrl).Object)
	if fo.Operation.VersionId == "" {
		logger.Infof("Start Delete object %s", cosPath)
	} else {
		logger.Infof("Start Delete version %s of the object %s", fo.Operation.VersionId, cosPath)
	}

	opt := &cos.ObjectDeleteOptions{
		XCosSSECustomerAglo:   "",
		XCosSSECustomerKey:    "",
		XCosSSECustomerKeyMD5: "",
		XOptionHeader:         nil,
		VersionId:             fo.Operation.VersionId,
	}

	if !fo.Operation.Force {
		if fo.Operation.VersionId == "" {
			logger.Infof("Are you sure you want to Delete object %s? (y/n)", cosPath)
		} else {
			logger.Infof("Are you sure you want to Delete version %s of the object %s? (y/n)", fo.Operation.VersionId, cosPath)
		}

		var choice string
		_, _ = fmt.Scanf("%s\n", &choice)
		if choice == "" || choice == "y" || choice == "Y" || choice == "yes" || choice == "Yes" || choice == "YES" {
			_, err = c.Object.Delete(context.Background(), cosUrl.(*CosUrl).Object, opt)
			if err != nil {
				return err
			}
			if fo.Operation.VersionId == "" {
				logger.Infof("Delete object %s successfully!", cosPath)
			} else {
				logger.Infof("Delete version %s of the object %s successfully!", fo.Operation.VersionId, cosPath)
			}
		} else {
			if fo.Operation.VersionId == "" {
				logger.Infof("Cancel Delete object %s", cosPath)
			} else {
				logger.Infof("Cancel Delete version %s of the object %s", fo.Operation.VersionId, cosPath)
			}
		}
	} else {
		_, err = c.Object.Delete(context.Background(), cosUrl.(*CosUrl).Object, opt)
		if err != nil {
			return err
		}
		if fo.Operation.VersionId == "" {
			logger.Infof("Delete object %s successfully!", cosPath)
		} else {
			logger.Infof("Delete version %s of the object %s successfully!", fo.Operation.VersionId, cosPath)
		}
	}

	if fo.Operation.VersionId == "" {
		logger.Infof("Delete object %s Completed", cosPath)
	} else {
		logger.Infof("Delete version %s of the object %s Completed", fo.Operation.VersionId, cosPath)
	}

	return nil
}

// RemoveBucket 删除cos桶
func RemoveBucket(bucketIDName string, c *cos.Client) error {

	_, err := c.Bucket.Delete(context.Background())
	if err != nil {
		return err
	}
	logger.Infof("Delete a empty bucket! name: %s\n", bucketIDName)
	return nil
}
