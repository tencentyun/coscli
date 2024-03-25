package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func fileStatistic(localPath string, cc *CopyCommand) {
	f, err := os.Stat(localPath)
	if err != nil {
		cc.Monitor.setScanError(err)
		return
	}
	if f.IsDir() {
		if !strings.HasSuffix(localPath, string(os.PathSeparator)) {
			localPath += string(os.PathSeparator)
		}

		err := getFileListStatistic(localPath, cc)
		if err != nil {
			cc.Monitor.setScanError(err)
			return
		}
	} else {
		if filterCheckpointDir(localPath, cc.CpParams.CheckpointDir) {
			cc.Monitor.updateScanSizeNum(f.Size(), 1)
		}
	}

	cc.Monitor.setScanEnd()
	freshProgress()
}

func getFileListStatistic(dpath string, cc *CopyCommand) error {
	if cc.CpParams.OnlyCurrentDir {
		return getCurrentDirFilesStatistic(dpath, cc)
	}

	name := dpath
	symlinkDiretorys := []string{dpath}
	walkFunc := func(fpath string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}

		if !filterCheckpointDir(fpath, cc.CpParams.CheckpointDir) {
			return nil
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
				cc.Monitor.updateScanNum(1)
			}
			return nil
		}

		if cc.CpParams.DisableAllSymlink && (f.Mode()&os.ModeSymlink) != 0 {
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

			if cc.CpParams.EnableSymlinkDir && realInfo.IsDir() {
				// 软链文件夹，如果有"/"后缀，os.Lstat 将判断它是一个目录
				if !strings.HasSuffix(name, string(os.PathSeparator)) {
					name += string(os.PathSeparator)
				}
				linkDir := name + fileName + string(os.PathSeparator)
				symlinkDiretorys = append(symlinkDiretorys, linkDir)
				return nil
			}
		}
		if fileMatchPatterns(f.Name(), cc.CpParams.Filters) {
			cc.Monitor.updateScanSizeNum(realFileSize, 1)
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

func getCurrentDirFilesStatistic(dpath string, cc *CopyCommand) error {
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

			if fileMatchPatterns(fileInfo.Name(), cc.CpParams.Filters) {
				cc.Monitor.updateScanSizeNum(fileInfo.Size(), 1)
			}
		}
	}
	return nil
}

func generateFileList(localPath string, chFiles chan<- fileInfoType, chListError chan<- error, cc *CopyCommand) {
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

		err := getFileList(localPath, chFiles, cc)
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

func getFileList(dpath string, chFiles chan<- fileInfoType, cc *CopyCommand) error {
	if cc.CpParams.OnlyCurrentDir {
		return getCurrentDirFileList(dpath, chFiles, cc)
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
				if strings.HasSuffix(fileName, "\\") || strings.HasSuffix(fileName, "/") {
					chFiles <- fileInfoType{fileName, name}
				} else {
					chFiles <- fileInfoType{fileName + string(os.PathSeparator), name}
				}
			}
			return nil
		}

		if cc.CpParams.DisableAllSymlink && (f.Mode()&os.ModeSymlink) != 0 {
			return nil
		}

		if cc.CpParams.EnableSymlinkDir && (f.Mode()&os.ModeSymlink) != 0 {
			// there is difference between os.Stat and os.Lstat in filepath.Walk
			realInfo, err := os.Stat(fpath)
			if err != nil {
				return err
			}

			if realInfo.IsDir() {
				// it's symlink dir
				// if linkDir has suffix os.PathSeparator,os.Lstat determine it is a dir
				if !strings.HasSuffix(name, string(os.PathSeparator)) {
					name += string(os.PathSeparator)
				}
				linkDir := name + fileName + string(os.PathSeparator)
				symlinkDiretorys = append(symlinkDiretorys, linkDir)
				return nil
			}
		}

		if fileMatchPatterns(fileName, cc.CpParams.Filters) {
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

func getCurrentDirFileList(dpath string, chFiles chan<- fileInfoType, cc *CopyCommand) error {
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

			if fileMatchPatterns(fileInfo.Name(), cc.CpParams.Filters) {
				chFiles <- fileInfoType{fileInfo.Name(), dpath}
			}
		}
	}
	return nil
}
