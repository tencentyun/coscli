package util

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Meta struct {
	CacheControl       string
	ContentDisposition string
	ContentEncoding    string
	ContentType        string
	ContentMD5         string
	ContentLength      int64
	ContentLanguage    string
	Expires            string
	// 自定义的 x-cos-meta-* header
	XCosMetaXXX *http.Header
	MetaChange  bool
}

func MetaStringToHeader(meta string) (result Meta, err error) {
	if meta == "" {
		return
	}
	meta = strings.TrimSpace(meta)
	kvs := strings.Split(meta, "#")
	header := http.Header{}
	metaXXX := &http.Header{}
	var metaChange bool
	for _, kv := range kvs {
		if kv == "" {
			continue
		}
		item := strings.Split(kv, ":")
		if len(item) < 2 {
			return result, fmt.Errorf("invalid meta item %v", item)
		}

		k := strings.ToLower(item[0])
		v := strings.Join(item[1:], ":")
		if strings.HasPrefix(k, "x-cos-meta-") {
			metaXXX.Set(k, v)
			metaChange = true
		} else {
			header.Set(k, v)
		}
	}

	expires := header.Get("Expires")
	if expires != "" {
		extime, err := time.Parse(time.RFC3339, expires)
		if err != nil {
			return result, fmt.Errorf("invalid meta expires format, %v", err)
		}

		expires = extime.Format(time.RFC1123)
	}
	result = Meta{
		CacheControl:       header.Get("Cache-Control"),
		ContentDisposition: header.Get("Content-Disposition"),
		ContentEncoding:    header.Get("Content-Encoding"),
		ContentType:        header.Get("Content-Type"),
		ContentMD5:         header.Get("Content-MD5"),
		ContentLength:      0,
		ContentLanguage:    header.Get("Content-Language"),
		Expires:            expires,
		XCosMetaXXX:        metaXXX,
		MetaChange:         metaChange,
	}

	cl := header.Get("Content-Length")
	if cl != "" {
		var clInt int64
		clInt, err = strconv.ParseInt(cl, 10, 64)
		if err != nil {
			return result, fmt.Errorf("parse meta ContentLength invalid, %v", err)
		}

		result.ContentLength = clInt
	}

	return result, nil
}
