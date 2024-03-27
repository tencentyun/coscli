package util

import (
	"fmt"
	logger "github.com/sirupsen/logrus"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

type StorageUrl interface {
	IsCosUrl() bool
	IsFileUrl() bool
	ToString() string
	UpdateUrlStr(newUrlStr string)
}

type CosUrl struct {
	urlStr string
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

func (cu *CosUrl) UpdateUrlStr(urlStr string) {
	cu.urlStr = urlStr
	cu.parseBucketObject()
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

func (fu *FileUrl) UpdateUrlStr(urlStr string) {
	fu.urlStr = urlStr
}

func (cu *CosUrl) Init(urlStr string) error {
	cu.urlStr = urlStr
	if err := cu.parseBucketObject(); err != nil {
		return err
	}
	if err := cu.checkBucketObject(); err != nil {
		return err
	}
	return nil
}

func (cu *CosUrl) parseBucketObject() error {
	path := cu.urlStr
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
		return fmt.Errorf("invalid cos url: %s, miss bucket", cu.urlStr)
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

func FormatUrl(urlStr string) (StorageUrl, error) {
	if strings.HasPrefix(strings.ToLower(urlStr), SchemePrefix) {
		var CosUrl CosUrl
		if err := CosUrl.Init(urlStr); err != nil {
			return nil, err
		}
		return &CosUrl, nil
	}
	var FileUrl FileUrl
	if err := FileUrl.Init(urlStr); err != nil {
		return nil, err
	}
	return &FileUrl, nil
}

func getCosUrl(bucket string, object string) string {
	cosUrl := CosUrl{
		Bucket: bucket,
		Object: object,
	}
	return cosUrl.ToString()
}

// 格式化上传操作cos路径及local路径
func FormatUploadPath(fileUrl StorageUrl, cosUrl StorageUrl, fo *FileOperations) {
	localPath := fileUrl.ToString()
	if localPath == "" {
		logger.Fatalln("localPath is empty")
	}

	// 获取本地文件/文件夹信息
	localPathInfo, err := os.Stat(localPath)
	if err != nil {
		logger.Fatalln(err)
	}

	if localPathInfo.IsDir() && !fo.Operation.Recursive {
		logger.Fatalf("localPath:%v is dir, please use --recursive option", localPath)
	}

	cosPath := cosUrl.ToString()
	// 若local路径若不以路径分隔符结尾 且 cos路径以路径分隔符结尾，则需将local路径的最终文件夹拼接至cos路径最后，并给local路径补充路径分隔符
	if !strings.HasSuffix(localPath, string(filepath.Separator)) && strings.HasSuffix(cosPath, "/") {
		fileName := filepath.Base(localPath)
		localPath += string(filepath.Separator)
		// cos路径拼接文件夹名
		cosPath += fileName + CosSeparator
	} else {
		// cos路径格式化
		if cosPath != "" && !strings.HasSuffix(cosPath, CosSeparator) {
			cosPath += CosSeparator
		}
	}

	fileUrl.UpdateUrlStr(localPath)
	cosUrl.UpdateUrlStr(cosPath)
}
