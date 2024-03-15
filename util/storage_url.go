package util

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"
)

const SchemePrefix string = "cos://"

type StorageUrl interface {
	IsCosUrl() bool
	IsFileUrl() bool
	ToString() string
}

type CosUrl struct {
	UrlStr string
	Bucket string
	Object string
}

type FileUrl struct {
	urlStr string
}

func (cu CosUrl) IsCosUrl() bool {
	return true
}

func (cu CosUrl) IsFileUrl() bool {
	return false
}

func (cu CosUrl) ToString() string {
	if cu.Object == "" {
		return fmt.Sprintf("%s%s", SchemePrefix, cu.Bucket)
	}
	return fmt.Sprintf("%s%s/%s", SchemePrefix, cu.Bucket, cu.Object)
}

func (fu FileUrl) IsCosUrl() bool {
	return false
}

func (fu FileUrl) IsFileUrl() bool {
	return true
}

func (fu FileUrl) ToString() string {
	return fu.urlStr
}

func (cu *CosUrl) Init(urlStr string) error {
	cu.UrlStr = urlStr
	if err := cu.parseBucketObject(); err != nil {
		return err
	}
	if err := cu.checkBucketObject(); err != nil {
		return err
	}
	return nil
}

func (cu *CosUrl) parseBucketObject() error {
	path := cu.UrlStr
	if strings.HasPrefix(strings.ToLower(path), SchemePrefix) {
		path = string(path[len(SchemePrefix):])
	} else {
		if strings.HasPrefix(path, "/") {
			path = string(path[1:])
		}
	}
	sli := strings.SplitN(path, "/", 2)
	cu.Bucket = sli[0]
	if len(sli) > 1 {
		cu.Object = sli[1]
	}
	return nil
}

func (cu *CosUrl) checkBucketObject() error {
	if cu.Bucket == "" && cu.Object != "" {
		return fmt.Errorf("invalid cos url: %s, miss bucket", cu.UrlStr)
	}
	return nil
}

func (fu *FileUrl) Init(urlStr string) error {
	if len(urlStr) >= 2 && urlStr[:2] == "~"+string(os.PathSeparator) {
		homeDir := currentHomeDir()
		if homeDir != "" {
			urlStr = strings.Replace(urlStr, "~", homeDir, 1)
		} else {
			return fmt.Errorf("current home dir is empty")
		}
	}
	fu.urlStr = urlStr
	return nil
}

func currentHomeDir() string {
	homeDir := ""
	homeDrive := os.Getenv("HOMEDRIVE")
	homePath := os.Getenv("HOMEPATH")
	if runtime.GOOS == "windows" && homeDrive != "" && homePath != "" {
		homeDir = homeDrive + string(os.PathSeparator) + homePath
	}

	if homeDir != "" {
		return homeDir
	}

	usr, _ := user.Current()
	if usr != nil {
		homeDir = usr.HomeDir
	} else {
		homeDir = os.Getenv("HOME")
	}
	return homeDir
}

func StorageUrlFromString(urlStr string) (StorageUrl, error) {
	if strings.HasPrefix(strings.ToLower(urlStr), SchemePrefix) {
		var CosUrl CosUrl
		if err := CosUrl.Init(urlStr); err != nil {
			return nil, err
		}
		return CosUrl, nil
	}
	var FileUrl FileUrl
	if err := FileUrl.Init(urlStr); err != nil {
		return nil, err
	}
	return FileUrl, nil
}
