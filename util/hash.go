package util

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc64"
	"io"
	"os"

	logger "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func ShowHash(c *cos.Client, path string, hashType string) (h string, b string) {
	opt := &cos.ObjectHeadOptions{
		IfModifiedSince:       "",
		XCosSSECustomerAglo:   "",
		XCosSSECustomerKey:    "",
		XCosSSECustomerKeyMD5: "",
		XOptionHeader:         nil,
	}

	resp, err := c.Object.Head(context.Background(), path, opt)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}

	switch hashType {
	case "crc64":
		h = resp.Header.Get("x-cos-hash-crc64ecma")
	case "md5":
		m := resp.Header.Get("etag")
		h = m[1 : len(m)-1]

		encode, _ := hex.DecodeString(h)
		b = base64.StdEncoding.EncodeToString(encode)
	default:
		logger.Infoln("Wrong args!")
	}
	return h, b
}

func CalculateHash(path string, hashType string) (h string, b string) {
	f, err := os.Open(path)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}
	defer f.Close()
	_, _ = f.Seek(0, 0)

	switch hashType {
	case "crc64":
		ecma := crc64.New(crc64.MakeTable(crc64.ECMA))
		w, _ := ecma.(hash.Hash)
		if _, err := io.Copy(w, f); err != nil {
			logger.Fatalln(err)
			os.Exit(1)
		}

		res := ecma.Sum64()
		h = fmt.Sprintf("%d", res)
	case "md5":
		m := md5.New()
		w, _ := m.(hash.Hash)
		if _, err := io.Copy(w, f); err != nil {
			logger.Fatalln(err)
			os.Exit(1)
		}

		res := m.Sum(nil)
		h = fmt.Sprintf("%x", res)
		b = base64.StdEncoding.EncodeToString(res)
	default:
		return "", ""
	}
	return h, b
}
