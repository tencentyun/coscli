package util

import (
	"bytes"
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io"
	"os"
	"sort"
	"strings"
)

var fileRemoveCount int

func getDeleteKeys(srcClient, destClient *cos.Client, srcUrl StorageUrl, destUrl StorageUrl, fo *FileOperations) (map[string]string, error) {
	var err error
	srcKeys := make(map[string]string)
	destKeys := make(map[string]string)
	if srcUrl.IsFileUrl() {
		err = getLocalFileKeys(srcUrl, srcKeys, fo)
	} else {
		err = GetCosKeys(srcClient, srcUrl, srcKeys, fo)
	}

	if err != nil {
		return nil, err
	}

	if destUrl.IsFileUrl() {
		err = getLocalFileKeys(destUrl, destKeys, fo)
	} else {
		err = GetCosKeys(destClient, destUrl, destKeys, fo)
	}

	if err != nil {
		return nil, err
	}

	// 根据操作系统和操作类型筛选出需要删除的对象或文件
	isLinux := (string(os.PathSeparator) == "/")
	for k, _ := range srcKeys {
		if isLinux || fo.CpType == CpTypeCopy {
			delete(destKeys, k)
		} else if fo.CpType == CpTypeUpload {
			delete(destKeys, strings.Replace(k, "\\", "/", -1))
		} else {
			delete(destKeys, strings.Replace(k, "/", "\\", -1))
		}
	}

	if destUrl.IsFileUrl() {
		fmt.Printf("\nfile(directory) will be removed count:%d\n", len(destKeys))
	} else {
		fmt.Printf("\nobject will be deleted count:%d\n", len(destKeys))
	}

	return destKeys, nil
}

func deleteKeys(c *cos.Client, keysToDelete map[string]string, destUrl StorageUrl, fo *FileOperations) error {
	// 根据类型区分删除cos上的对象还是本地文件
	if fo.CpType == CpTypeCopy || fo.CpType == CpTypeUpload {
		err := DeleteCosObjects(c, keysToDelete, destUrl, fo)
		return err
	} else {
		err := DeleteLocalFiles(keysToDelete, destUrl, fo)
		return err
	}

	return nil
}

func DeleteCosObjects(c *cos.Client, keysToDelete map[string]string, cosUrl StorageUrl, fo *FileOperations) error {
	deleteCount := 0
	objects := []cos.Object{}
	for k, v := range keysToDelete {
		if len(objects) >= MaxDeleteBatchCount {
			if confirm(objects, fo, cosUrl) {
				opt := &cos.ObjectDeleteMultiOptions{
					Objects: objects,
					// 布尔值，这个值决定了是否启动 Quiet 模式
					// 值为 true 启动 Quiet 模式，值为 false 则启动 Verbose 模式，默认值为 false
					Quiet: true,
				}
				_, _, err := c.Object.DeleteMulti(context.Background(), opt)
				if err != nil {
					return err
				}
			}
			objects = []cos.Object{}
			deleteCount += MaxDeleteBatchCount
			fmt.Printf("\rdelete object count:%d", deleteCount)
		}

		objects = append(objects, cos.Object{Key: v + k})
	}

	if len(objects) > 0 && confirm(objects, fo, cosUrl) {
		opt := &cos.ObjectDeleteMultiOptions{
			Objects: objects,
			// 布尔值，这个值决定了是否启动 Quiet 模式
			// 值为 true 启动 Quiet 模式，值为 false 则启动 Verbose 模式，默认值为 false
			Quiet: true,
		}
		_, _, err := c.Object.DeleteMulti(context.Background(), opt)
		if err != nil {
			return err
		}
		deleteCount += len(objects)
		fmt.Printf("\rdelete object count:%d", deleteCount)
	}
	return nil
}

func confirm(objects []cos.Object, fo *FileOperations, cosUrl StorageUrl) bool {
	if fo.Operation.Force {
		return true
	}

	var logBuffer bytes.Buffer
	logBuffer.WriteString("\n")
	for _, v := range objects {
		logBuffer.WriteString(fmt.Sprintf("%s\n", SchemePrefix+cosUrl.(*CosUrl).Bucket+CosSeparator+v.Key))
	}
	logBuffer.WriteString(fmt.Sprintf("sync:delete above objects(Y or N)? "))
	fmt.Printf(logBuffer.String())

	var val string
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func DeleteLocalFiles(keysToDelete map[string]string, fileUrl StorageUrl, fo *FileOperations) error {
	// 检查备份路径
	err := checkBackupDir(fileUrl, fo)
	if err != nil {
		return err
	}
	var sortList []string
	for key, _ := range keysToDelete {
		sortList = append(sortList, key)
	}
	// 排序，先删除文件后删除文件夹
	sort.Sort(sort.Reverse(sort.StringSlice(sortList)))

	absDirName, err := getAbsPath(fileUrl.ToString())
	if err != nil {
		return err
	}

	nowFatherDirName := ""
	for _, key := range sortList {
		if strings.HasSuffix(key, string(os.PathSeparator)) {
			dirName := key[0 : len(key)-1]
			readerInfos, _ := getDirFiles(absDirName+dirName, 10)

			if len(readerInfos) > 0 {
				continue
			} else {
				// 获取备份路径
				f, err := os.Stat(fo.Operation.BackupDir + dirName)
				if err != nil {
					movePath(absDirName+dirName, fo.Operation.BackupDir+dirName)
				} else {
					if !f.IsDir() {
						return fmt.Errorf("backup %s is already exist,but is file", fo.Operation.BackupDir+dirName)
					} else {
						// 文件夹里面内容已被删完，则删除文件夹
						os.RemoveAll(absDirName + dirName)
					}
				}
			}
		} else {
			fatherDir := absDirName
			index := strings.LastIndex(key, string(os.PathSeparator))
			if index >= 0 {
				fatherDir = key[:index]
			}

			if fatherDir != nowFatherDirName && fatherDir != absDirName {
				os.MkdirAll(fo.Operation.BackupDir+fatherDir, 0755)
				nowFatherDirName = fatherDir
			}

			err := movePath(absDirName+key, fo.Operation.BackupDir+key)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func checkBackupDir(fileUrl StorageUrl, fo *FileOperations) error {
	createDir := false
	f, err := os.Stat(fileUrl.ToString())
	if err != nil {
		if err := os.MkdirAll(fileUrl.ToString(), 0755); err != nil {
			return err
		}
		createDir = true
	} else if !f.IsDir() {
		return fmt.Errorf("dest dir %s is file,is not directory", fileUrl.ToString())
	}

	if createDir && fo.Operation.BackupDir == "" {
		return nil
	}

	if fo.Operation.BackupDir == "" {
		return fmt.Errorf("files backup dir is empty string,please use --backup-dir")
	}

	if !strings.HasSuffix(fo.Operation.BackupDir, string(os.PathSeparator)) {
		fo.Operation.BackupDir += string(os.PathSeparator)
	}

	// 检查备份路径是否是目标文件路径的子路径
	absFileDir, err := getAbsPath(fileUrl.ToString())
	if err != nil {
		return err
	}

	absBackupDir, err := getAbsPath(fo.Operation.BackupDir)
	if err != nil {
		return err
	}

	if strings.Index(absBackupDir, absFileDir) >= 0 {
		return fmt.Errorf("files backup dir %s is subdirectory of %s", fo.Operation.BackupDir, fileUrl.ToString())
	}

	f, err = os.Stat(fo.Operation.BackupDir)
	if err != nil {
		if err := os.MkdirAll(fo.Operation.BackupDir, 0755); err != nil {
			return err
		}
	} else if !f.IsDir() {
		return fmt.Errorf("files backup dir %s is file,is not directory", fo.Operation.BackupDir)
	}
	return nil
}

func getDirFiles(dirName string, limitCount int) ([]os.FileInfo, error) {
	f, err := os.Open(dirName)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(limitCount)
	f.Close()
	if err != nil {
		return nil, err
	}
	return list, nil
}

func movePath(srcName, destName string) error {
	err := moveFileToPath(srcName, destName)
	if err != nil {
		return fmt.Errorf("rename %s %s error,%s\n", srcName, destName, err.Error())
	} else {
		fileRemoveCount += 1
		fmt.Printf("\rremove file(directory) count:%d", fileRemoveCount)
	}
	return err
}

func moveFileToPath(srcName, destName string) error {
	err := os.Rename(srcName, destName)
	if err == nil {
		return nil
	} else {
		inputFile, err := os.Open(srcName)
		defer inputFile.Close()
		if err != nil {
			return err
		}
		outputFile, err := os.Create(destName)
		defer outputFile.Close()
		if err != nil {
			return err
		}
		_, err = io.Copy(outputFile, inputFile)
		if err != nil {
			return err
		}
		err = os.Remove(srcName)
		if err != nil {
			return err
		}
		return nil
	}
}