package util

import (
	"context"
	"net/http"
	"strings"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// ObjectStatInfo 对象元数据信息
type ObjectStatInfo struct {
	ETag               string
	ContentType        string
	ContentLength      string
	LastModified       string
	CacheControl       string
	ContentDisposition string
	ContentEncoding    string
	ContentLanguage    string
	Expires            string
	StorageClass       string
	VersionId          string
	ObjectType         string
	CRC64              string
	CustomMeta         map[string]string
}

// StatObject 查询对象元数据，通过 HEAD Object 接口获取
// bucketType 用于判断是否携带 versionId：OFS 桶不接受 versionId。
func StatObject(c *cos.Client, objectKey string, versionId string, bucketType string) (*ObjectStatInfo, error) {
	opt := &cos.ObjectHeadOptions{
		XOptionHeader: &http.Header{},
	}

	var resp *cos.Response
	var err error
	if needCarryVersionId(bucketType, versionId) {
		resp, err = c.Object.Head(context.Background(), objectKey, opt, versionId)
	} else {
		resp, err = c.Object.Head(context.Background(), objectKey, opt)
	}
	if err != nil {
		return nil, err
	}

	header := resp.Header
	info := &ObjectStatInfo{
		ETag:               header.Get("ETag"),
		ContentType:        header.Get("Content-Type"),
		ContentLength:      header.Get("Content-Length"),
		LastModified:       header.Get("Last-Modified"),
		CacheControl:       header.Get("Cache-Control"),
		ContentDisposition: header.Get("Content-Disposition"),
		ContentEncoding:    header.Get("Content-Encoding"),
		ContentLanguage:    header.Get("Content-Language"),
		Expires:            header.Get("Expires"),
		StorageClass:       header.Get("x-cos-storage-class"),
		VersionId:          header.Get("x-cos-version-id"),
		ObjectType:         header.Get("x-cos-object-type"),
		CRC64:              header.Get("x-cos-hash-crc64ecma"),
		CustomMeta:         make(map[string]string),
	}

	// 提取自定义元数据 x-cos-meta-*
	for k, v := range header {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "x-cos-meta-") {
			info.CustomMeta[lk] = strings.Join(v, ",")
		}
	}

	return info, nil
}
