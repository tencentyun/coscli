package util

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"
)

// GetCosKeys reads keys from a COS (Cloud Object Storage) URL and processes them.
//
// c: *cos.Client - the COS client
// cosUrl: StorageUrl - the COS URL to read keys from
// keys: map[string]string - the keys to process
// fo: *FileOperations - the file operations object
//
// Returns an error if any of the operations fail.
func GetCosKeys(c *cos.Client, cosUrl StorageUrl, keys map[string]commonInfoType, fo *FileOperations, TypeDest string) error {

	chFiles := make(chan objectInfoType, ChannelSize)
	chFinish := make(chan error, 2)
	go ReadCosKeys(keys, cosUrl, chFiles, chFinish, fo, TypeDest)
	go getCosObjectList(c, cosUrl, chFiles, chFinish, fo, false, false)
	select {
	case err := <-chFinish:
		if err != nil {
			return err
		}
	}

	return nil
}

// ReadCosKeys reads keys from a COS bucket and sends them to the provided channels.
//
// keys: A map to store the keys read from COS.
// cosUrl: The URL of the COS bucket.
// chObjects: A channel to receive object info types.
// chFinish: A channel to send errors if the number of keys exceeds the maximum allowed.
func ReadCosKeys(keys map[string]commonInfoType, cosUrl StorageUrl, chObjects <-chan objectInfoType, chFinish chan<- error, fo *FileOperations, objType string) {

	// 2. 创建通道
	results := make(chan commonInfoType, ChannelSize) // 缓冲通道提高性能
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. 启动工作池
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU()*2; i++ { // 根据CPU核心数动态调整
		wg.Add(1)
		go func() {
			defer wg.Done()
			for objectInfo := range chObjects {
				// 解析时间字符串
				var lastModifiedTime time.Time
				var lastModifiedTimeUnix int64
				var err error
				if objectInfo.lastModified == "" {
					lastModifiedTimeUnix = int64(0)
				} else {
					lastModifiedTime, err = time.Parse(time.RFC3339, objectInfo.lastModified)
					if err != nil {
						lastModifiedTime, err = time.Parse(time.RFC1123, objectInfo.lastModified)
						if err != nil {
							chFinish <- err
						}
					}
					lastModifiedTimeUnix = lastModifiedTime.Unix()
				}

				select {
				case results <- commonInfoType{
					key:              objectInfo.relativeKey,
					dir:              objectInfo.prefix,
					size:             objectInfo.size,
					lastModifiedUnix: lastModifiedTimeUnix,
					lastModified:     objectInfo.lastModified,
				}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// 4. 启动结果收集器
	go func() {
		defer close(done)
		totalCount := 0
		lastReport := time.Now()
		batchSize := 1000 // 批量处理大小
		batch := make([]commonInfoType, 0, batchSize)

		for res := range results {
			totalCount++
			batch = append(batch, res)

			// 批量处理
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
					fmt.Printf("\n")
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

	// 5. 等待所有工作完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 6. 等待结果收集完成
	<-done
}

// CheckCosPathType checks if the given path is a directory or not.
// It takes a *cos.Client, a prefix string, a limit int, and a *FileOperations as parameters.
// It returns a bool indicating whether the path is a directory and an error if any occurs.
func CheckCosPathType(c *cos.Client, prefix string, limit int, fo *FileOperations) (isDir bool, err error) {
	if prefix == "" {
		return true, nil
	}

	// cos路径若不以路径分隔符结尾，则添加
	if !strings.HasSuffix(prefix, CosSeparator) && prefix != "" {
		prefix += CosSeparator
	}

	opt := &cos.BucketGetOptions{
		Prefix:       prefix,
		Delimiter:    "",
		EncodingType: "url",
		Marker:       "",
		MaxKeys:      limit,
	}

	res, err := tryGetObjects(c, opt)
	if err != nil {
		return isDir, err
	}

	isDir = false
	if len(res.Contents) > 0 {
		isDir = true
	}
	if fo.BucketType == BucketTypeOfs && len(res.CommonPrefixes) > 0 {
		isDir = true
	}

	return isDir, nil
}

// CheckCosObjectExist checks if an object exists in the COS bucket with the given prefix and IDs.
// c: *cos.Client, the COS client to use.
// prefix: string, the prefix of the object to check.
// id: ...string, the IDs of the object to check.
// Returns: (exist bool, err error), whether the object exists and an error if any.
func CheckCosObjectExist(c *cos.Client, prefix string, id ...string) (exist bool, err error) {
	if prefix == "" {
		return false, nil
	}
	exist, err = c.Object.IsExist(context.Background(), prefix, id...)
	if err != nil {
		return exist, err
	}

	return exist, nil
}

// CheckUploadExist checks if the upload with the given ID exists in the COS client.
// c: *cos.Client - the COS client instance.
// cosUrl: StorageUrl - the COS storage URL.
// uploadId: string - the upload ID to check for.
// Returns: (exist bool, err error) - whether the upload exists and any error encountered.
func CheckUploadExist(c *cos.Client, cosUrl StorageUrl, uploadId string) (exist bool, err error) {
	var uploads []struct {
		Key          string
		UploadID     string `xml:"UploadId"`
		StorageClass string
		Initiator    *cos.Initiator
		Owner        *cos.Owner
		Initiated    string
	}
	isTruncated := true
	var keyMarker, uploadIDMarker string
	for isTruncated {
		err, uploads, isTruncated, uploadIDMarker, keyMarker = GetUploadsListForLs(c, cosUrl, uploadIDMarker, keyMarker, 0, false)
		for _, object := range uploads {
			if uploadId == object.UploadID {
				return true, nil
			}
		}
	}
	return false, nil
}

// CheckDeleteMarkerExist checks if a delete marker exists for the given version ID.
// c: *cos.Client - the COS client to use for the request.
// cosUrl: StorageUrl - the COS storage URL to check.
// versionId: string - the version ID to check for the delete marker.
func CheckDeleteMarkerExist(c *cos.Client, cosUrl StorageUrl, versionId string) (exist bool, err error) {

	isTruncated := true
	var versionIdMarker, keyMarker string
	var deleteMarkers []cos.ListVersionsResultDeleteMarker

	for isTruncated {
		err, _, deleteMarkers, _, isTruncated, versionIdMarker, keyMarker = getCosObjectVersionListForLs(c, cosUrl, versionIdMarker, keyMarker, 0, false)

		for _, object := range deleteMarkers {
			if versionId == object.VersionId {
				return true, nil
			}
		}
	}

	if err != nil {
		return false, err
	}
	return false, nil
}

func getCosObjectList(c *cos.Client, cosUrl StorageUrl, chObjects chan<- objectInfoType, chError chan<- error, fo *FileOperations, scanSizeNum bool, withFinishSignal bool) {
	//startTime := time.Now().UnixNano() / 1000 / 1000
	//defer func() {
	//	endTime := time.Now().UnixNano() / 1000 / 1000
	//	costTime := int(endTime - startTime)
	//	logger.Info(fmt.Sprintf("get cos list cost %ds", costTime/1000))
	//}()
	if chObjects != nil {
		defer close(chObjects)
	}

	prefix := cosUrl.(*CosUrl).Object
	marker := ""
	limit := 0
	delimiter := ""
	if fo.Operation.OnlyCurrentDir {
		delimiter = "/"
	}

	isTruncated := true
	for isTruncated {
		// 实例化请求参数
		opt := &cos.BucketGetOptions{
			Prefix:       prefix,
			Delimiter:    delimiter,
			EncodingType: "url",
			Marker:       marker,
			MaxKeys:      limit,
		}
		res, err := tryGetObjects(c, opt)
		if err != nil {
			if scanSizeNum {
				fo.Monitor.setScanError(err)
				return
			} else {
				chError <- err
				return
			}

		}
		for _, object := range res.Contents {
			object.Key, _ = url.QueryUnescape(object.Key)
			if cosObjectMatchPatterns(object.Key, fo.Operation.Filters) {
				if scanSizeNum {
					fo.Monitor.updateScanSizeNum(object.Size, 1)
				} else {
					objPrefix := ""
					objKey := object.Key
					index := strings.LastIndex(cosUrl.(*CosUrl).Object, "/")
					if index > 0 {
						objPrefix = object.Key[:index+1]
						objKey = object.Key[index+1:]
					}
					chObjects <- objectInfoType{prefix: objPrefix, relativeKey: objKey, size: object.Size, lastModified: object.LastModified}
				}
			}
		}

		isTruncated = res.IsTruncated
		marker, _ = url.QueryUnescape(res.NextMarker)
	}

	if scanSizeNum {
		fo.Monitor.setScanEnd()
		freshProgress()
	}

	if withFinishSignal {
		chError <- nil
	}
}

func getObjectListByKeys(srcKeys, transferKeys map[string]commonInfoType, chObjects chan<- objectInfoType, chListError chan<- error, fo *FileOperations) {
	defer close(chObjects)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for _, v := range srcKeys {
			fo.Monitor.updateScanSizeNum(v.size, 1)
		}
		fo.Monitor.setScanEnd()
		freshProgress()
	}()

	go func() {
		defer wg.Done()
		for k, v := range srcKeys {
			if _, exists := transferKeys[k]; exists {
				chObjects <- objectInfoType{v.dir, v.key, v.size, v.lastModified, false}
			} else {
				chObjects <- objectInfoType{v.dir, v.key, v.size, v.lastModified, true}
			}
		}
		// 发送完成信号
		chListError <- nil
	}()

	// 等待两个任务完成
	wg.Wait()
}

func getCosObjectListForLs(c *cos.Client, cosUrl StorageUrl, marker string, limit int, recursive bool) (err error, objects []cos.Object, commonPrefixes []string, isTruncated bool, nextMarker string) {

	prefix := cosUrl.(*CosUrl).Object
	delimiter := ""
	if !recursive {
		delimiter = "/"
	}

	// 实例化请求参数
	opt := &cos.BucketGetOptions{
		Prefix:       prefix,
		Delimiter:    delimiter,
		EncodingType: "url",
		Marker:       marker,
		MaxKeys:      limit,
	}
	res, err := tryGetObjects(c, opt)
	if err != nil {
		return
	}

	objects = res.Contents
	commonPrefixes = res.CommonPrefixes
	isTruncated = res.IsTruncated
	nextMarker, _ = url.QueryUnescape(res.NextMarker)
	return
}

func getCosObjectVersionListForLs(c *cos.Client, cosUrl StorageUrl, versionIdMarker, keyMarker string, limit int, recursive bool) (err error, versions []cos.ListVersionsResultVersion, deleteMarkers []cos.ListVersionsResultDeleteMarker, commonPrefixes []string, isTruncated bool, nextVersionIdMarker, nextKeyMarker string) {

	prefix := cosUrl.(*CosUrl).Object
	delimiter := ""
	if !recursive {
		delimiter = "/"
	}

	// 实例化请求参数
	opt := &cos.BucketGetObjectVersionsOptions{
		Prefix:          prefix,
		Delimiter:       delimiter,
		EncodingType:    "url",
		VersionIdMarker: versionIdMarker,
		KeyMarker:       keyMarker,
		MaxKeys:         limit,
	}
	res, err := tryGetObjectVersions(c, opt)
	if err != nil {
		return
	}

	versions = res.Version
	deleteMarkers = res.DeleteMarker
	commonPrefixes = res.CommonPrefixes
	isTruncated = res.IsTruncated
	nextVersionIdMarker, _ = url.QueryUnescape(res.NextVersionIdMarker)
	nextKeyMarker, _ = url.QueryUnescape(res.NextKeyMarker)
	return
}

// GetFilesAndDirs 获取所有文件和目录
func GetFilesAndDirs(c *cos.Client, cosDir string, nextMarker string, include string, exclude string) (files []string, err error) {
	objects, _, _, commonPrefixes, err := GetObjectsListIterator(c, cosDir, nextMarker, include, exclude)
	if err != nil {
		return files, err
	}
	tempFiles := make([]string, 0)
	tempFiles = append(tempFiles, cosDir)
	for _, v := range objects {
		files = append(files, v.Key)
	}
	if len(commonPrefixes) > 0 {
		for _, v := range commonPrefixes {
			subFiles, err := GetFilesAndDirs(c, v, nextMarker, include, exclude)
			if err != nil {
				return files, err
			}
			tempFiles = append(tempFiles, subFiles...)
		}
	}
	files = append(files, tempFiles...)
	return files, nil
}
