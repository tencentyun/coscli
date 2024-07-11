package util

import (
	"context"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"io/fs"
	"io/ioutil"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	logger "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func UrlDecodeCosPattern(objects []cos.Object) []cos.Object {
	res := make([]cos.Object, 0)
	for _, o := range objects {
		o.Key, _ = url.QueryUnescape(o.Key)
		res = append(res, o)
	}
	return res
}

func MatchCosPattern(objects []cos.Object, pattern string, include bool) []cos.Object {
	res := make([]cos.Object, 0)
	for _, o := range objects {
		match, _ := regexp.Match(pattern, []byte(o.Key))
		if !include {
			match = !match
		}
		if match {
			res = append(res, o)
		}
	}
	return res
}

func MatchUploadPattern(uploads []UploadInfo, pattern string, include bool) []UploadInfo {
	res := make([]UploadInfo, 0)
	for _, u := range uploads {
		match, _ := regexp.Match(pattern, []byte(u.Key))
		if !include {
			match = !match
		}
		if match {
			res = append(res, u)
		}
	}
	return res
}

func MatchPattern(strs []string, pattern string, include bool) []string {
	res := make([]string, 0)
	re := regexp.MustCompile(pattern)
	for _, s := range strs {
		match := re.MatchString(s)
		if !include {
			match = !match
		}
		if match {
			res = append(res, s)
		}
	}
	return res
}

func GetObjectsListRecursive(c *cos.Client, prefix string, limit int, include string, exclude string, retryCount ...int) (objects []cos.Object,
	commonPrefixes []string) {

	retries := 0
	if len(retryCount) > 0 {
		retries = retryCount[0]
	}

	opt := &cos.BucketGetOptions{
		Prefix:       prefix,
		Delimiter:    "",
		EncodingType: "url",
		Marker:       "",
		MaxKeys:      limit,
	}

	isTruncated := true
	marker := ""
	for isTruncated {
		opt.Marker = marker

		res, err := tryGetBucket(c, opt, retries)
		if err != nil {
			logger.Fatalln(err)
			os.Exit(1)
		}

		objects = append(objects, res.Contents...)
		commonPrefixes = res.CommonPrefixes

		if limit > 0 {
			isTruncated = false
		} else {
			isTruncated = res.IsTruncated
			marker, _ = url.QueryUnescape(res.NextMarker)
		}
	}

	// 对key进行urlDecode解码
	objects = UrlDecodeCosPattern(objects)

	if len(include) > 0 {
		objects = MatchCosPattern(objects, include, true)
	}
	if len(exclude) > 0 {
		objects = MatchCosPattern(objects, exclude, false)
	}

	return objects, commonPrefixes
}

func tryGetBucket(c *cos.Client, opt *cos.BucketGetOptions, retryCount int) (*cos.BucketGetResult, error) {
	for i := 0; i <= retryCount; i++ {
		res, resp, err := c.Bucket.Get(context.Background(), opt)
		if err != nil {
			if resp != nil && resp.StatusCode == 503 {
				if i == retryCount {
					return res, err
				} else {
					fmt.Println("Error 503: Service Unavailable. Retrying...")
					waitTime := time.Duration(rand.Intn(10)+1) * time.Second
					time.Sleep(waitTime)
					continue
				}
			} else {
				return res, err
			}
		} else {
			return res, err
		}
	}
	return nil, fmt.Errorf("Retry limit exceeded")
}

func GetLocalFilesListRecursive(localPath string, include string, exclude string) (files []string) {
	// bfs遍历文件夹
	var dirs []string
	dirs = append(dirs, localPath)
	for len(dirs) > 0 {
		dirName := dirs[0]
		dirs = dirs[1:]

		fileInfos, err := ioutil.ReadDir(dirName)
		if err != nil {
			logger.Fatalln(err)
			os.Exit(1)
		}
		if len(fileInfos) == 0 {
			logger.Warningf("skip empty dir: %s", dirName)
			continue
		}

		for _, f := range fileInfos {
			fileName := filepath.Join(dirName, f.Name())
			if f.Mode().IsRegular() { // 普通文件，直接添加
				fileName = fileName[len(localPath):]
				files = append(files, fileName)
			} else if f.IsDir() { // 普通目录，添加到继续迭代
				dirs = append(dirs, fileName)
			} else if f.Mode()&os.ModeSymlink == fs.ModeSymlink { // 软链接
				linkTarget, err := os.Readlink(fileName)
				if err != nil {
					logger.Infoln(fmt.Sprintf("Failed to read symlink %s, will be excluded", fileName))
					continue
				}
				fmt.Println(dirName, linkTarget)
				linkTargetPath := filepath.Join(dirName, linkTarget)
				linkTargetInfo, err := os.Stat(linkTargetPath)
				if err != nil {
					logger.Infoln(fmt.Sprintf("Failed to stat symlink target %s, will be excluded", linkTargetPath))
					continue
				}
				if linkTargetInfo.Mode().IsRegular() {
					linkTargetPath = linkTargetPath[len(localPath):]
					files = append(files, linkTargetPath)
				} else if linkTargetInfo.IsDir() {
					dirs = append(dirs, linkTargetPath)
				} else {
					logger.Infoln(fmt.Sprintf("List %s file is not regular file, will be excluded", linkTargetPath))
					continue
				}
			} else {
				logger.Infoln(fmt.Sprintf("List %s file is not regular file, will be excluded", fileName))
				continue
			}
		}
	}

	if len(include) > 0 {
		files = MatchPattern(files, include, true)
	}
	if len(exclude) > 0 {
		files = MatchPattern(files, exclude, false)
	}

	return files
}

func GetUploadsListRecursive(c *cos.Client, prefix string, limit int, include string, exclude string) (uploads []UploadInfo) {
	opt := &cos.ListMultipartUploadsOptions{
		Delimiter:      "",
		EncodingType:   "",
		Prefix:         prefix,
		MaxUploads:     limit,
		KeyMarker:      "",
		UploadIDMarker: "",
	}

	isTruncated := true
	keyMarker := ""
	uploadIDMarker := ""
	for isTruncated {
		opt.KeyMarker = keyMarker
		opt.UploadIDMarker = uploadIDMarker

		res, _, err := c.Bucket.ListMultipartUploads(context.Background(), opt)
		if err != nil {
			logger.Fatalln(err)
			os.Exit(1)
		}

		for _, u := range res.Uploads {
			uploads = append(uploads, UploadInfo{
				Key:       u.Key,
				UploadID:  u.UploadID,
				Initiated: u.Initiated,
			})
		}

		if limit > 0 {
			isTruncated = false
		} else {
			isTruncated = res.IsTruncated
			keyMarker = res.NextKeyMarker
			uploadIDMarker = res.NextUploadIDMarker
		}
	}

	if len(include) > 0 {
		uploads = MatchUploadPattern(uploads, include, true)
	}
	if len(exclude) > 0 {
		uploads = MatchUploadPattern(uploads, exclude, false)
	}

	return uploads
}

// =====new

func ListObjects(c *cos.Client, cosUrl StorageUrl, limit int, recursive bool, filters []FilterOptionType) {
	var err error
	var objects []cos.Object
	var commonPrefixes []string
	total := 0
	isTruncated := true
	marker := ""

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Key", "Type", "Last Modified", "Etag", "Size", "RestoreStatus"})
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)

	for isTruncated && total < limit {
		table.ClearRows()
		queryLimit := 1000
		if limit-total < 1000 {
			queryLimit = limit - total
		}

		err, objects, commonPrefixes, isTruncated, marker = getCosObjectListForLs(c, cosUrl, marker, queryLimit, recursive)

		if err != nil {
			logger.Fatalf("list objects error : %v", err)
		}

		if len(commonPrefixes) > 0 {
			for _, commonPrefix := range commonPrefixes {
				if cosObjectMatchPatterns(commonPrefix, filters) {
					table.Append([]string{commonPrefix, "DIR", "", "", "", ""})
					total++
				}
			}
		}

		for _, object := range objects {
			object.Key, _ = url.QueryUnescape(object.Key)
			if cosObjectMatchPatterns(object.Key, filters) {
				utcTime, err := time.Parse(time.RFC3339, object.LastModified)
				if err != nil {
					fmt.Println("Error parsing time:", err)
					return
				}
				table.Append([]string{object.Key, object.StorageClass, utcTime.Local().Format(time.RFC3339), object.ETag, formatBytes(float64(object.Size)), object.RestoreStatus})
				total++
			}
		}

		if !isTruncated || total >= limit {
			table.SetFooter([]string{"", "", "", "", "Total Objects: ", fmt.Sprintf("%d", total)})
			table.Render()
			break
		}
		table.Render()

		// 重置表格
		table = tablewriter.NewWriter(os.Stdout)
		table.SetBorder(false)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetAutoWrapText(false)
	}

}

func ListOfsObjects(c *cos.Client, cosUrl StorageUrl, limit int, recursive bool, filters []FilterOptionType) {
	lsCounter := &LsCounter{}
	prefix := cosUrl.(*CosUrl).Object

	lsCounter.Table = tablewriter.NewWriter(os.Stdout)
	lsCounter.Table.SetHeader([]string{"Key", "Type", "Last Modified", "Etag", "Size", "RestoreStatus"})
	lsCounter.Table.SetBorder(false)
	lsCounter.Table.SetAlignment(tablewriter.ALIGN_LEFT)
	lsCounter.Table.SetAutoWrapText(false)

	getOfsObjects(c, prefix, limit, recursive, filters, "", lsCounter)

	lsCounter.Table.SetFooter([]string{"", "", "", "", "Total Objects: ", fmt.Sprintf("%d", lsCounter.TotalLimit)})
	lsCounter.Table.Render()
}

func getOfsObjects(c *cos.Client, prefix string, limit int, recursive bool, filters []FilterOptionType, marker string, lsCounter *LsCounter) {
	var err error
	var objects []cos.Object
	var commonPrefixes []string
	isTruncated := true

	for isTruncated {

		queryLimit := 1000
		if limit-lsCounter.TotalLimit < 1000 {
			queryLimit = limit - lsCounter.TotalLimit
		}

		if queryLimit <= 0 {
			return
		}

		err, objects, commonPrefixes, isTruncated, marker = getOfsObjectListForLs(c, prefix, marker, queryLimit, recursive)

		if err != nil {
			logger.Fatalf("list objects error : %v", err)
		}

		for _, object := range objects {
			object.Key, _ = url.QueryUnescape(object.Key)
			if cosObjectMatchPatterns(object.Key, filters) {
				utcTime, err := time.Parse(time.RFC3339, object.LastModified)
				if err != nil {
					fmt.Println("Error parsing time:", err)
					return
				}
				if lsCounter.TotalLimit >= limit {
					break
				}
				lsCounter.TotalLimit++
				lsCounter.RenderNum++
				lsCounter.Table.Append([]string{object.Key, object.StorageClass, utcTime.Local().Format(time.RFC3339), object.ETag, formatBytes(float64(object.Size)), object.RestoreStatus})
				tableRender(lsCounter)
			}
		}

		if len(commonPrefixes) > 0 {
			for _, commonPrefix := range commonPrefixes {
				commonPrefix, _ = url.QueryUnescape(commonPrefix)
				if cosObjectMatchPatterns(commonPrefix, filters) {
					if lsCounter.TotalLimit >= limit {
						break
					}
					lsCounter.TotalLimit++
					lsCounter.RenderNum++
					lsCounter.Table.Append([]string{commonPrefix, "DIR", "", "", "", ""})
					tableRender(lsCounter)
					// 递归目录
					getOfsObjects(c, commonPrefix, limit, recursive, filters, "", lsCounter)
				}
			}
		}
	}
}

func tableRender(lsCounter *LsCounter) {
	if lsCounter.RenderNum >= OfsMaxRenderNum {
		lsCounter.Table.Render()
		lsCounter.Table.ClearRows()
		lsCounter.RenderNum = 0
		lsCounter.Table = tablewriter.NewWriter(os.Stdout)
		lsCounter.Table.SetBorder(false)
		lsCounter.Table.SetAlignment(tablewriter.ALIGN_LEFT)
		lsCounter.Table.SetAutoWrapText(false)
	}
}

func ListBuckets(c *cos.Client, limit int) {
	var buckets []cos.Bucket
	marker := ""
	isTruncated := true
	totalNum := 0

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Bucket Name", "Region", "Create Date"})
	for isTruncated {
		buckets, marker, isTruncated = GetBucketsList(c, limit, marker)
		for _, b := range buckets {
			table.Append([]string{b.Name, b.Region, b.CreationDate})
			totalNum++
		}
		if limit > 0 {
			isTruncated = false
		}
	}

	table.SetFooter([]string{"", "Total Buckets: ", fmt.Sprintf("%d", totalNum)})
	table.SetBorder(false)
	table.Render()
}

func GetBucketsList(c *cos.Client, limit int, marker string) (buckets []cos.Bucket, nextMarker string, isTruncated bool) {
	opt := &cos.ServiceGetOptions{
		Marker:  marker,
		MaxKeys: int64(limit),
	}
	res, _, err := c.Service.Get(context.Background(), opt)

	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}

	buckets = res.Buckets
	nextMarker = res.NextMarker
	isTruncated = res.IsTruncated

	return
}
