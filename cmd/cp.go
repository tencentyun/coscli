package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"os"
	"time"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var cpCmd = &cobra.Command{
	Use:   "cp",
	Short: "Upload, download or cper objects",
	Long: `Upload, download or cper objects

Format:
  ./coscli cp <source_path> <destination_path> [flags]

Example: 
  Upload:
    ./coscli cp ~/example.txt cos://examplebucket/example.txt
  Download:
    ./coscli cp cos://examplebucket/example.txt ~/example.txt
  Copy:
    ./coscli cp cos://examplebucket1/example1.txt cos://examplebucket2/example2.txt`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(2)(cmd, args); err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		recursive, _ := cmd.Flags().GetBool("recursive")
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		storageClass, _ := cmd.Flags().GetString("storage-class")
		rateLimiting, _ := cmd.Flags().GetFloat32("rate-limiting")
		partSize, _ := cmd.Flags().GetInt64("part-size")
		threadNum, _ := cmd.Flags().GetInt("thread-num")
		routines, _ := cmd.Flags().GetInt("routines")
		failOutput, _ := cmd.Flags().GetBool("fail-output")
		failOutputPath, _ := cmd.Flags().GetString("fail-output-path")
		metaString, _ := cmd.Flags().GetString("meta")
		retryNum, _ := cmd.Flags().GetInt("retry-num")
		errRetryNum, _ := cmd.Flags().GetInt("err-retry-num")
		errRetryInterval, _ := cmd.Flags().GetInt("err-retry-interval")
		onlyCurrentDir, _ := cmd.Flags().GetBool("only-current-dir")
		disableAllSymlink, _ := cmd.Flags().GetBool("disable-all-symlink")
		enableSymlinkDir, _ := cmd.Flags().GetBool("enable-symlink-dir")
		disableCrc64, _ := cmd.Flags().GetBool("disable-crc64")
		disableChecksum, _ := cmd.Flags().GetBool("disable-checksum")
		disableLongLinks, _ := cmd.Flags().GetBool("disable-long-links")
		longLinksNums, _ := cmd.Flags().GetInt("long-links-nums")
		versionId, _ := cmd.Flags().GetString("version-id")
		move, _ := cmd.Flags().GetBool("move")

		meta, err := util.MetaStringToHeader(metaString)
		if err != nil {
			return fmt.Errorf("Copy invalid meta " + err.Error())
		}

		if retryNum < 0 || retryNum > 10 {
			return fmt.Errorf("retry-num must be between 0 and 10 (inclusive)")
		}

		if errRetryNum < 0 || errRetryNum > 10 {
			return fmt.Errorf("err-retry-num must be between 0 and 10 (inclusive)")
		}

		if errRetryInterval < 0 || errRetryInterval > 10 {
			return fmt.Errorf("err-retry-interval must be between 0 and 10 (inclusive)")
		}

		srcUrl, err := util.FormatUrl(args[0])
		if err != nil {
			return fmt.Errorf("format srcURL error,%v", err)
		}

		destUrl, err := util.FormatUrl(args[1])
		if err != nil {
			return fmt.Errorf("format destURL error,%v", err)
		}

		if srcUrl.IsFileUrl() && destUrl.IsFileUrl() {
			return fmt.Errorf("not support cp between local directory")
		}

		if move && !(srcUrl.IsCosUrl() && destUrl.IsCosUrl()) {
			return fmt.Errorf("move only supports cp between cos paths")
		}

		_, filters := util.GetFilter(include, exclude)

		fo := &util.FileOperations{
			Operation: util.Operation{
				Recursive:         recursive,
				Filters:           filters,
				StorageClass:      storageClass,
				RateLimiting:      rateLimiting,
				PartSize:          partSize,
				ThreadNum:         threadNum,
				Routines:          routines,
				FailOutput:        failOutput,
				FailOutputPath:    failOutputPath,
				Meta:              meta,
				RetryNum:          retryNum,
				ErrRetryNum:       errRetryNum,
				ErrRetryInterval:  errRetryInterval,
				OnlyCurrentDir:    onlyCurrentDir,
				DisableAllSymlink: disableAllSymlink,
				EnableSymlinkDir:  enableSymlinkDir,
				DisableCrc64:      disableCrc64,
				DisableChecksum:   disableChecksum,
				DisableLongLinks:  disableLongLinks,
				LongLinksNums:     longLinksNums,
				VersionId:         versionId,
				Move:              move,
			},
			Monitor:    &util.FileProcessMonitor{},
			Config:     &config,
			Param:      &param,
			ErrOutput:  &util.ErrOutput{},
			CpType:     getCommandType(srcUrl, destUrl),
			Command:    util.CommandCP,
			BucketType: "COS",
		}

		if !fo.Operation.Recursive && len(fo.Operation.Filters) > 0 {
			return fmt.Errorf("--include or --exclude only work with --recursive")
		}

		srcPath := srcUrl.ToString()
		destPath := destUrl.ToString()

		var operate string
		startT := time.Now().UnixNano() / 1000 / 1000
		if srcUrl.IsFileUrl() && destUrl.IsCosUrl() {
			operate = "Upload"
			logger.Infof("Upload %s to %s start", srcPath, destPath)
			// 检查错误输出日志是否是本地路径的子集
			err = util.CheckPath(srcUrl, fo, util.TypeFailOutputPath)
			if err != nil {
				return err
			}
			// 实例化cos client
			bucketName := destUrl.(*util.CosUrl).Bucket
			c, err := util.NewClient(fo.Config, fo.Param, bucketName, fo)
			if err != nil {
				return err
			}
			// 是否关闭crc64
			if fo.Operation.DisableCrc64 {
				c.Conf.EnableCRC = false
			}
			// 格式化上传路径
			err = util.FormatUploadPath(srcUrl, destUrl, fo)
			if err != nil {
				return err
			}
			// 上传
			util.Upload(c, srcUrl, destUrl, fo)
		} else if srcUrl.IsCosUrl() && destUrl.IsFileUrl() {
			operate = "Download"
			logger.Infof("Download %s to %s start", srcPath, destPath)
			if storageClass != "" {
				return fmt.Errorf("--storage-class can not use in download")
			}
			// 检查错误输出日志是否是本地路径的子集
			err = util.CheckPath(destUrl, fo, util.TypeFailOutputPath)
			if err != nil {
				return err
			}
			bucketName := srcUrl.(*util.CosUrl).Bucket
			c, err := util.NewClient(fo.Config, fo.Param, bucketName, fo)
			if err != nil {
				return err
			}

			if versionId != "" {
				res, _, err := util.GetBucketVersioning(c)
				if err != nil {
					return err
				}

				if res.Status != util.VersionStatusEnabled {
					return fmt.Errorf("versioning is not enabled on the current bucket")
				}
			}

			// 判断桶是否是ofs桶
			s, err := c.Bucket.Head(context.Background())
			if err != nil {
				return err
			}
			// 根据s.Header判断是否是融合桶或者普通桶
			if s.Header.Get("X-Cos-Bucket-Arch") == "OFS" {
				fo.BucketType = "OFS"
			}
			// 是否关闭crc64
			if fo.Operation.DisableCrc64 {
				c.Conf.EnableCRC = false
			}
			// 格式化下载路径
			err = util.FormatDownloadPath(srcUrl, destUrl, fo, c)
			if err != nil {
				return err
			}
			// 下载
			err = util.Download(c, srcUrl, destUrl, fo)
			if err != nil {
				return err
			}
		} else if srcUrl.IsCosUrl() && destUrl.IsCosUrl() {
			operate = "Copy"
			if move {
				operate = "Move"
			}
			logger.Infof("%s %s to %s start", operate, srcPath, destPath)
			// 实例化来源 cos client
			srcBucketName := srcUrl.(*util.CosUrl).Bucket
			srcClient, err := util.NewClient(fo.Config, fo.Param, srcBucketName)
			if err != nil {
				return err
			}

			if versionId != "" {
				res, _, err := util.GetBucketVersioning(srcClient)
				if err != nil {
					return err
				}

				if res.Status != util.VersionStatusEnabled {
					return fmt.Errorf("versioning is not enabled on the src bucket")
				}
			}

			// 实例化目标 cos client
			destBucketName := destUrl.(*util.CosUrl).Bucket
			destClient, err := util.NewClient(fo.Config, fo.Param, destBucketName, fo)
			if err != nil {
				return err
			}

			// 判断桶是否是ofs桶
			s, _ := srcClient.Bucket.Head(context.Background())
			// 根据s.Header判断是否是融合桶或者普通桶
			if s.Header.Get("X-Cos-Bucket-Arch") == "OFS" {
				fo.BucketType = "OFS"
			}

			// 是否关闭crc64
			if fo.Operation.DisableCrc64 {
				destClient.Conf.EnableCRC = false
			}

			// 格式化copy路径
			err = util.FormatCopyPath(srcUrl, destUrl, fo, srcClient)
			if err != nil {
				return err
			}

			if move && (srcUrl.(*util.CosUrl).Object == destUrl.(*util.CosUrl).Object) {
				return fmt.Errorf("using --move is not allowed when the target path and source path for cos are the same")
			}

			// 拷贝
			err = util.CosCopy(srcClient, destClient, srcUrl, destUrl, fo)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cospath needs to contain %s", util.SchemePrefix)
		}
		util.CloseErrorOutputFile(fo)
		endT := time.Now().UnixNano() / 1000 / 1000
		util.PrintCostTime(startT, endT)

		if fo.Monitor.ErrNum > 0 {
			logger.Warningf("%s %s to %s %s", operate, srcPath, destPath, fo.Monitor.GetFinishInfo())
			os.Exit(2)
		} else {
			logger.Infof("%s %s to %s %s", operate, srcPath, destPath, fo.Monitor.GetFinishInfo())
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cpCmd)

	cpCmd.Flags().BoolP("recursive", "r", false, "Copy objects recursively")
	cpCmd.Flags().String("include", "", "Include files that meet the specified criteria")
	cpCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
	cpCmd.Flags().String("storage-class", "", "Specifying a storage class")
	cpCmd.Flags().Float32("rate-limiting", 0, "Upload or download speed limit(MB/s)")
	cpCmd.Flags().Int64("part-size", 32, "Specifies the block size(MB)")
	cpCmd.Flags().Int("thread-num", 5, "Specifies the number of partition concurrent upload or download threads")
	cpCmd.Flags().Int("routines", 3, "Specifies the number of files concurrent upload or download threads")
	cpCmd.Flags().Bool("fail-output", true, "This option determines whether the error output for failed file uploads or downloads is enabled. If enabled, the error messages for any failed file transfers will be recorded in a file within the specified directory (if not specified, the default is coscli_output). If disabled, only the number of error files will be output to the console.")
	cpCmd.Flags().String("fail-output-path", "coscli_output", "This option specifies the designated error output folder where the error messages for failed file uploads or downloads will be recorded. By providing a custom folder path, you can control the location and name of the error output folder. If this option is not set, the default error log folder (coscli_output) will be used.")
	cpCmd.Flags().String("meta", "",
		"Set the meta information of the file, "+
			"the format is header:value#header:value, the example is Cache-Control:no-cache#Content-Encoding:gzip")
	cpCmd.Flags().Int("retry-num", 0, "Rate-limited retry. Specify 1-10 times. When multiple machines concurrently execute download operations on the same COS directory, rate-limited retry can be performed by specifying this parameter.")
	cpCmd.Flags().Int("err-retry-num", 0, "Error retry attempts. Specify 1-10 times, or 0 for no retry.")
	cpCmd.Flags().Int("err-retry-interval", 0, "Retry interval (available only when specifying error retry attempts 1-10). Specify an interval of 1-10 seconds, or if not specified or set to 0, a random interval within 1-10 seconds will be used for each retry.")
	cpCmd.Flags().Bool("only-current-dir", false, "Upload only the files in the current directory, ignoring subdirectories and their contents")
	cpCmd.Flags().Bool("disable-all-symlink", true, "Ignore all symbolic link subfiles and symbolic link subdirectories when uploading, not uploaded by default")
	cpCmd.Flags().Bool("enable-symlink-dir", false, "Upload linked subdirectories, not uploaded by default")
	cpCmd.Flags().Bool("disable-crc64", false, "Disable CRC64 data validation. By default, coscli enables CRC64 validation for data transfer")
	cpCmd.Flags().Bool("disable-checksum", false, "Disable overall CRC64 checksum, only validate fragments")
	cpCmd.Flags().Bool("disable-long-links", false, "Disable long links, use short links")
	cpCmd.Flags().Int("long-links-nums", 0, "The long connection quantity parameter, if 0 or not provided, defaults to the concurrent file count.")
	cpCmd.Flags().String("version-id", "", "Downloading a specified version of a file , only available if bucket versioning is enabled.")
	cpCmd.Flags().Bool("move", false, "Enable migration mode (only available between COS paths), which will delete the source file after it has been successfully copied to the destination path.")
}

func getCommandType(srcUrl util.StorageUrl, destUrl util.StorageUrl) util.CpType {
	if srcUrl.IsCosUrl() {
		if destUrl.IsFileUrl() {
			return util.CpTypeDownload
		}
		return util.CpTypeCopy
	}
	return util.CpTypeUpload
}
