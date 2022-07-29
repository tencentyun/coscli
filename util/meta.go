package util

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
		if len(item) != 2 {
			return result, fmt.Errorf("invalid meta item %v", item)
		}

		k := strings.ToLower(item[0])
		v := item[1]
		if strings.HasPrefix(k, "x-cos-meta-") {
			metaXXX.Set(k, v)
			metaChange = true
		} else {
			header.Set(k, v)
		}
	}

	result = Meta{
		CacheControl:       header.Get("CacheControl"),
		ContentDisposition: header.Get("ContentDisposition"),
		ContentEncoding:    header.Get("ContentEncoding"),
		ContentType:        header.Get("ContentType"),
		ContentMD5:         header.Get("ContentMD5"),
		ContentLength:      0,
		ContentLanguage:    header.Get("ContentLanguage"),
		Expires:            header.Get("Expires"),
		XCosMetaXXX:        metaXXX,
		MetaChange:         metaChange,
	}

	cl := header.Get("ContentLength")
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
