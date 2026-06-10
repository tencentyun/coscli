package util

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"os"
	"path/filepath"
	"strings"
)

// IsCosPath checks if the given path is a COS path.
// It returns true if the path starts with "cos://", otherwise it returns false.
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

// ParsePath 解析URL路径，返回桶名和路径。
// 如果URL是COS路径，则解析路径；否则，如果URL以'~'开头，则返回当前用户的home目录。
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

// UploadPathFixed appends the file name to the cosPath if it is not complete or ends with a slash.
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

// DownloadPathFixed appends the necessary path separator to the input path if it does not end with one.
// It also ensures the path is compatible with Windows by replacing '/' with '\\' if necessary.
func DownloadPathFixed(relativeObject, filePath string) string {
	if strings.HasSuffix(filePath, "/") || strings.HasSuffix(filePath, "\\") {
		return filePath + relativeObject
	}
	// 兼容windows路径
	filePath = strings.Replace(filePath, "/", string(os.PathSeparator), -1)
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

// CheckPath checks if the given fileUrl is a subdirectory of the local file path.
// It returns an error if the fileUrl is invalid or if the local file path cannot be determined.
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
		if fo.Operation.FailOutput {
			path = fo.Operation.FailOutputPath
		} else {
			return nil
		}

	} else if pathType == TypeProcessLogPath {
		if fo.Operation.ProcessLog {
			path = fo.Operation.ProcessLogPath
		} else {
			return nil
		}

	} else {
		return fmt.Errorf("check path failed , invalid pathType %s", pathType)
	}

	absPath, err := getAbsPath(path)
	if err != nil {
		return err
	}

	if strings.Index(absPath, absFileDir) >= 0 {
		return fmt.Errorf("%s %s is subdirectory of %s", pathType, absPath, fileUrl.ToString())
	}
	return nil
}

// IsLocalPathWithinDir 判断目标文件路径 targetPath 解析后是否仍位于下载目标目录 baseDir 之内。
// 用于防御对象名中携带 "../" 等路径穿越片段，导致下载内容被写入到用户指定目录之外（越界写入）。
// 返回 true 表示安全（在目录内或等于目录本身），false 表示发生了路径穿越。
func IsLocalPathWithinDir(baseDir, targetPath string) (bool, error) {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return false, err
	}
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return false, err
	}

	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return false, err
	}

	// rel 为 ".." 或以 "../"（Windows 下为 "..\\"）开头，说明目标路径逃逸到了 baseDir 之外
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false, nil
	}
	return true, nil
}

func createParentDirectory(localFilePath string) error {
	dir, err := filepath.Abs(filepath.Dir(localFilePath))
	if err != nil {
		return err
	}
	dir = strings.Replace(dir, "\\", "/", -1)
	return os.MkdirAll(dir, 0755)
}
