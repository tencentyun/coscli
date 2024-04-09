package util

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"os"
	"path/filepath"
	"strings"
)

func IsCosPath(path string) bool {
	if len(path) <= 6 {
		return false
	}
	if path[:6] == "cos://" {
		return true
	} else {
		return false
	}
}

func ParsePath(url string) (bucketName string, path string) {
	if IsCosPath(url) {
		res := strings.SplitN(url[6:], "/", 2)
		if len(res) < 2 {
			return res[0], ""
		} else {
			return res[0], res[1]
		}
	} else {
		if url[0] == '~' {
			home, _ := homedir.Dir()
			path = home + url[1:]
		} else {
			path = url
		}
		return "", path
	}
}

func UploadPathFixed(file fileInfoType, cosPath string) (string, string) {
	// cos路径不全则补充文件名
	if cosPath == "" || strings.HasSuffix(cosPath, "/") {
		filePath := file.filePath
		filePath = strings.Replace(file.filePath, string(os.PathSeparator), "/", -1)
		filePath = strings.Replace(file.filePath, "\\", "/", -1)
		cosPath += filePath
	}

	localFilePath := filepath.Join(file.dir, file.filePath)

	return localFilePath, cosPath
}

func DownloadPathFixed(relativeObject, filePath string) string {
	if strings.HasSuffix(filePath, "/") || strings.HasSuffix(filePath, "\\") {
		return filePath + relativeObject
	}

	return filePath
}

func copyPathFixed(relativeObject, destPath string) string {
	if destPath == "" || strings.HasSuffix(destPath, "/") {
		return destPath + relativeObject
	}

	return destPath
}

func getAbsPath(strPath string) (string, error) {
	if filepath.IsAbs(strPath) {
		return strPath, nil
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	if !strings.HasSuffix(strPath, string(os.PathSeparator)) {
		strPath += string(os.PathSeparator)
	}

	strPath = currentDir + string(os.PathSeparator) + strPath
	absPath, err := filepath.Abs(strPath)
	if err != nil {
		return "", err
	}

	if !strings.HasSuffix(absPath, string(os.PathSeparator)) {
		absPath += string(os.PathSeparator)
	}
	return absPath, err
}

// 检查路径是否是本地文件路径的子路径
func CheckPath(fileUrl StorageUrl, fo *FileOperations, pathType string) error {
	absFileDir, err := getAbsPath(fileUrl.ToString())
	if err != nil {
		return err
	}

	var path string
	if pathType == TypeSnapshotPath {
		path = fo.Operation.SnapshotPath
	} else if pathType == TypeFailOutputPath {
		path = fo.Operation.FailOutputPath
	} else {
		return fmt.Errorf("check path failed , invalid pathType %s", pathType)
	}

	absPath, err := getAbsPath(path)
	if err != nil {
		return err
	}

	if strings.Index(absPath, absFileDir) >= 0 {
		return fmt.Errorf("%s %s is subdirectory of %s", pathType, fo.Operation.SnapshotPath, fileUrl.ToString())
	}
	return nil
}

func createParentDirectory(localFilePath string) error {
	dir, err := filepath.Abs(filepath.Dir(localFilePath))
	if err != nil {
		return err
	}
	dir = strings.Replace(dir, "\\", "/", -1)
	return os.MkdirAll(dir, 0755)
}
