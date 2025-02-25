package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var once sync.Once

func fileStatistic(localPath string, fo *FileOperations) {
	f, err := os.Stat(localPath)
	if err != nil {
		fo.Monitor.setScanError(err)
		return
	}
	if f.IsDir() {
		if !strings.HasSuffix(localPath, string(os.PathSeparator)) {
			localPath += string(os.PathSeparator)
		}

		err := getFileListStatistic(localPath, fo)
		if err != nil {
			fo.Monitor.setScanError(err)
			return
		}
	} else {
		fo.Monitor.updateScanSizeNum(f.Size(), 1)
	}

	fo.Monitor.setScanEnd()
	freshProgress()
}

func getFileListStatistic(dpath string, fo *FileOperations) error {
	if fo.Operation.OnlyCurrentDir {
		return getCurrentDirFilesStatistic(dpath, fo)
	}

	name := dpath
	symlinkDiretorys := []string{dpath}
	walkFunc := func(fpath string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}

		realFileSize := f.Size()
		dpath = filepath.Clean(dpath)
		fpath = filepath.Clean(fpath)
		fileName, err := filepath.Rel(dpath, fpath)
		if err != nil {
			return fmt.Errorf("list file error: %s, info: %s", fpath, err.Error())
		}

		if f.IsDir() {
			if fpath != dpath {
				if matchPatterns(filepath.Join(dpath, fileName), fo.Operation.Filters) {
					fo.Monitor.updateScanNum(1)
				}
			}
			return nil
		}

		if fo.Operation.DisableAllSymlink && (f.Mode()&os.ModeSymlink) != 0 {
			return nil
		}

		// 处理软链文件或文件夹
		if f.Mode()&os.ModeSymlink != 0 {

			realInfo, err := os.Stat(fpath)
			if err != nil {
				return err
			}

			if realInfo.IsDir() {
				realFileSize = 0
			} else {
				realFileSize = realInfo.Size()
			}

			if fo.Operation.EnableSymlinkDir && realInfo.IsDir() {
				// 软链文件夹，如果有"/"后缀，os.Lstat 将判断它是一个目录
				if !strings.HasSuffix(name, string(os.PathSeparator)) {
					name += string(os.PathSeparator)
				}
				linkDir := name + fileName + string(os.PathSeparator)
				symlinkDiretorys = append(symlinkDiretorys, linkDir)
				return nil
			}
		}
		if matchPatterns(filepath.Join(dpath, fileName), fo.Operation.Filters) {
			fo.Monitor.updateScanSizeNum(realFileSize, 1)
		}
		return nil
	}

	var err error
	for {
		symlinks := symlinkDiretorys
		symlinkDiretorys = []string{}
		for _, v := range symlinks {
			err = filepath.Walk(v, walkFunc)
			if err != nil {
				return err
			}
		}
		if len(symlinkDiretorys) == 0 {
			break
		}
	}
	return err
}

func getCurrentDirFilesStatistic(dpath string, fo *FileOperations) error {
	if !strings.HasSuffix(dpath, string(os.PathSeparator)) {
		dpath += string(os.PathSeparator)
	}

	fileList, err := ioutil.ReadDir(dpath)
	if err != nil {
		return err
	}

	for _, fileInfo := range fileList {
		if !fileInfo.IsDir() {
			realInfo, errF := os.Stat(dpath + fileInfo.Name())
			if errF == nil && realInfo.IsDir() {
				// for symlink
				continue
			}
			if matchPatterns(filepath.Join(dpath, fileInfo.Name()), fo.Operation.Filters) {
				fo.Monitor.updateScanSizeNum(fileInfo.Size(), 1)
			}
		}
	}
	return nil
}

func generateFileList(localPath string, chFiles chan<- fileInfoType, chListError chan<- error, fo *FileOperations) {
	defer close(chFiles)
	f, err := os.Stat(localPath)
	if err != nil {
		chListError <- err
		return
	}
	if f.IsDir() {
		if !strings.HasSuffix(localPath, string(os.PathSeparator)) {
			localPath += string(os.PathSeparator)
		}

		err := getFileList(localPath, chFiles, fo)
		if err != nil {
			chListError <- err
			return
		}
	} else {
		dir, fname := filepath.Split(localPath)
		chFiles <- fileInfoType{fname, dir}
	}
	chListError <- nil
}

func getFileList(dpath string, chFiles chan<- fileInfoType, fo *FileOperations) error {
	if fo.Operation.OnlyCurrentDir {
		return getCurrentDirFileList(dpath, chFiles, fo)
	}

	name := dpath
	symlinkDiretorys := []string{dpath}
	walkFunc := func(fpath string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}

		dpath = filepath.Clean(dpath)
		fpath = filepath.Clean(fpath)

		fileName, err := filepath.Rel(dpath, fpath)
		if err != nil {
			return fmt.Errorf("list file error: %s, info: %s", fpath, err.Error())
		}

		if f.IsDir() {
			if fpath != dpath {
				if matchPatterns(filepath.Join(dpath, fileName), fo.Operation.Filters) {
					if strings.HasSuffix(fileName, "\\") || strings.HasSuffix(fileName, "/") {
						chFiles <- fileInfoType{fileName, name}
					} else {
						chFiles <- fileInfoType{fileName + string(os.PathSeparator), name}
					}
				}
			}
			return nil
		}

		if fo.Operation.DisableAllSymlink && (f.Mode()&os.ModeSymlink) != 0 {
			return nil
		}

		if fo.Operation.EnableSymlinkDir && (f.Mode()&os.ModeSymlink) != 0 {
			realInfo, err := os.Stat(fpath)
			if err != nil {
				return err
			}

			if realInfo.IsDir() {
				if !strings.HasSuffix(name, string(os.PathSeparator)) {
					name += string(os.PathSeparator)
				}
				linkDir := name + fileName + string(os.PathSeparator)
				symlinkDiretorys = append(symlinkDiretorys, linkDir)
				return nil
			}
		}

		if matchPatterns(filepath.Join(dpath, fileName), fo.Operation.Filters) {
			chFiles <- fileInfoType{fileName, name}
		}
		return nil
	}

	var err error
	for {
		symlinks := symlinkDiretorys
		symlinkDiretorys = []string{}
		for _, v := range symlinks {
			err = filepath.Walk(v, walkFunc)
			if err != nil {
				return err
			}
		}
		if len(symlinkDiretorys) == 0 {
			break
		}
	}
	return err
}

func getCurrentDirFileList(dpath string, chFiles chan<- fileInfoType, fo *FileOperations) error {
	if !strings.HasSuffix(dpath, string(os.PathSeparator)) {
		dpath += string(os.PathSeparator)
	}

	fileList, err := ioutil.ReadDir(dpath)
	if err != nil {
		return err
	}

	for _, fileInfo := range fileList {
		if !fileInfo.IsDir() {
			realInfo, errF := os.Stat(dpath + fileInfo.Name())
			if errF == nil && realInfo.IsDir() {
				// for symlink
				continue
			}

			if matchPatterns(filepath.Join(dpath, fileInfo.Name()), fo.Operation.Filters) {
				chFiles <- fileInfoType{fileInfo.Name(), dpath}
			}
		}
	}
	return nil
}

func getLocalFileKeys(fileUrl StorageUrl, keys map[string]string, fo *FileOperations) error {
	strPath := fileUrl.ToString()
	if !strings.HasSuffix(strPath, string(os.PathSeparator)) {
		strPath += string(os.PathSeparator)
	}

	chFiles := make(chan fileInfoType, ChannelSize)
	chFinish := make(chan error, 2)
	go ReadLocalFileKeys(chFiles, chFinish, keys, fo)
	go GetFileList(strPath, chFiles, chFinish, fo)
	select {
	case err := <-chFinish:
		if err != nil {
			return err
		}
	}
	return nil
}

func ReadLocalFileKeys(chFiles <-chan fileInfoType, chFinish chan<- error, keys map[string]string, fo *FileOperations) {
	totalCount := 0
	fmt.Printf("\n")
	for fileInfo := range chFiles {
		totalCount++
		fmt.Printf("\rtotal file(directory) count:%d", totalCount)
		keys[fileInfo.filePath] = ""
		if len(keys) > MaxSyncNumbers {
			fmt.Printf("\n")
			chFinish <- fmt.Errorf("over max sync numbers %d", MaxSyncNumbers)
			break
		}
	}
	fmt.Printf("\rtotal file(directory) count:%d", totalCount)
	chFinish <- nil
}

func GetFileList(strPath string, chFiles chan<- fileInfoType, chFinish chan<- error, fo *FileOperations) {
	defer close(chFiles)
	err := getFileList(strPath, chFiles, fo)
	if err != nil {
		chFinish <- err
	}
}
