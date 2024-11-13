package cmd

import (
	"context"
	"fmt"
	"time"

	"coscli/util"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize objects",
	Long: `Synchronize objects

Format:
  ./coscli sync <source_path> <destination_path> [flags]

Example:
  Sync Upload:
    ./coscli sync ~/example.txt cos://examplebucket/example.txt
  Sync Download:
    ./coscli sync cos://examplebucket/example.txt ~/example.txt
  Sync Copy:
    ./coscli sync cos://examplebucket1/example1.txt cos://examplebucket2/example2.txt`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(2)(cmd, args); err != nil {
			return err
		}
		storageClass, _ := cmd.Flags().GetString("storage-class")
		if storageClass != "" && util.IsCosPath(args[0]) {
			return fmt.Errorf("--storage-class can only use in upload")
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
		metaString, _ := cmd.Flags().GetString("meta")
		retryNum, _ := cmd.Flags().GetInt("retry-num")
		errRetryNum, _ := cmd.Flags().GetInt("err-retry-num")
		errRetryInterval, _ := cmd.Flags().GetInt("err-retry-interval")
		snapshotPath, _ := cmd.Flags().GetString("snapshot-path")
		delete, _ := cmd.Flags().GetBool("delete")
		routines, _ := cmd.Flags().GetInt("routines")
		failOutput, _ := cmd.Flags().GetBool("fail-output")
		failOutputPath, _ := cmd.Flags().GetString("fail-output-path")
		onlyCurrentDir, _ := cmd.Flags().GetBool("only-current-dir")
		disableAllSymlink, _ := cmd.Flags().GetBool("disable-all-symlink")
		enableSymlinkDir, _ := cmd.Flags().GetBool("enable-symlink-dir")
		disableCrc64, _ := cmd.Flags().GetBool("disable-crc64")
		disableChecksum, _ := cmd.Flags().GetBool("disable-checksum")
		disableLongLinks, _ := cmd.Flags().GetBool("disable-long-links")
		longLinksNums, _ := cmd.Flags().GetInt("long-links-nums")
		backupDir, _ := cmd.Flags().GetString("backup-dir")
		force, _ := cmd.Flags().GetBool("force")

		meta, err := util.MetaStringToHeader(metaString)
		if err != nil {
			return fmt.Errorf("Sync invalid meta, reason: " + err.Error())
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
				SnapshotPath:      snapshotPath,
				Delete:            delete,
				BackupDir:         backupDir,
				Force:             force,
			},
			Monitor:   &util.FileProcessMonitor{},
			Config:    &config,
			Param:     &param,
			ErrOutput: &util.ErrOutput{},
			CpType:    getCommandType(srcUrl, destUrl),
			Command:   util.CommandSync,
		}

		// 快照db实例化
		err = util.InitSnapshotDb(srcUrl, destUrl, fo)
		if err != nil {
			return err
		}
		startT := time.Now().UnixNano() / 1000 / 1000
		if srcUrl.IsFileUrl() && destUrl.IsCosUrl() {
			// 检查错误输出日志是否是本地路径的子集
			err = util.CheckPath(srcUrl, fo, util.TypeFailOutputPath)
			if err != nil {
				return err
			}
			// 格式化上传路径
			err = util.FormatUploadPath(srcUrl, destUrl, fo)
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
			// 上传
			err = util.SyncUpload(c, srcUrl, destUrl, fo)
			if err != nil {
				return err
			}
		} else if srcUrl.IsCosUrl() && destUrl.IsFileUrl() {
			// 检查错误输出日志是否是本地路径的子集
			err = util.CheckPath(destUrl, fo, util.TypeFailOutputPath)
			if err != nil {
				return err
			}

			if fo.Operation.Delete {
				// 检查备份路径
				err = util.CheckBackupDir(destUrl, fo)
				if err != nil {
					return err
				}
			}

			bucketName := srcUrl.(*util.CosUrl).Bucket
			c, err := util.NewClient(fo.Config, fo.Param, bucketName, fo)
			if err != nil {
				return err
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
			err = util.SyncDownload(c, srcUrl, destUrl, fo)
			if err != nil {
				return err
			}
		} else if srcUrl.IsCosUrl() && destUrl.IsCosUrl() {
			// 实例化来源 cos client
			srcBucketName := srcUrl.(*util.CosUrl).Bucket
			srcClient, err := util.NewClient(fo.Config, fo.Param, srcBucketName)
			if err != nil {
				return err
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
			// 拷贝
			err = util.SyncCosCopy(srcClient, destClient, srcUrl, destUrl, fo)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cospath needs to contain cos://")
		}
		util.CloseErrorOutputFile(fo)
		endT := time.Now().UnixNano() / 1000 / 1000
		util.PrintCostTime(startT, endT)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().BoolP("recursive", "r", false, "Synchronize objects recursively")
	syncCmd.Flags().String("include", "", "List files that meet the specified criteria")
	syncCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
	syncCmd.Flags().String("storage-class", "", "Specifying a storage class")
	syncCmd.Flags().Float32("rate-limiting", 0, "Upload or download speed limit(MB/s)")
	syncCmd.Flags().Int64("part-size", 32, "Specifies the block size(MB)")
	syncCmd.Flags().Int("thread-num", 5, "Specifies the number of concurrent upload or download threads")
	syncCmd.Flags().String("meta", "",
		"Set the meta information of the file, "+
			"the format is header:value#header:value, the example is Cache-Control:no-cache#Content-Encoding:gzip")
	syncCmd.Flags().String("snapshot-path", "", "This option is used to accelerate the incremental"+
		" upload of batch files or download objects in certain scenarios."+
		" If you use the option when upload files or download objects,"+
		" coscli will generate files to record the snapshot information in the specified directory."+
		" When the next time you upload files or download objects with the option, "+
		"coscli will read the snapshot information under the specified directory for incremental upload or incremental download. "+
		"The snapshot-path you specified must be a local file system directory can be written in, "+
		"if the directory does not exist, coscli creates the files for recording snapshot information, "+
		"else coscli will read snapshot information from the path for "+
		"incremental upload(coscli will only upload the files which haven't not been successfully uploaded to oss or"+
		" been locally modified) or incremental download(coscli will only download the objects which have not"+
		" been successfully downloaded or have been modified),"+
		" and update the snapshot information to the directory. "+
		"Note: The option record the lastModifiedTime of local files "+
		"which have been successfully uploaded in local file system or lastModifiedTime of objects which have been successfully"+
		" downloaded, and compare the lastModifiedTime of local files or objects in the next cp to decided whether to"+
		" skip the file or object. "+
		"In addition, coscli does not automatically delete snapshot-path snapshot information, "+
		"in order to avoid too much snapshot information, when the snapshot information is useless, "+
		"please clean up your own snapshot-path on your own immediately.")
	syncCmd.Flags().Bool("delete", false, "Delete any other files in the specified destination path, only keeping the files synced this time. It is recommended to enable version control before using the --delete option to prevent accidental data deletion.")
	syncCmd.Flags().Int("retry-num", 0, "Rate-limited retry. Specify 1-10 times. When multiple machines concurrently execute download operations on the same COS directory, rate-limited retry can be performed by specifying this parameter.")
	syncCmd.Flags().Int("err-retry-num", 0, "Error retry attempts. Specify 1-10 times, or 0 for no retry.")
	syncCmd.Flags().Int("err-retry-interval", 0, "Retry interval (available only when specifying error retry attempts 1-10). Specify an interval of 1-10 seconds, or if not specified or set to 0, a random interval within 1-10 seconds will be used for each retry.")
	syncCmd.Flags().Int("routines", 3, "Specifies the number of files concurrent upload or download threads")
	syncCmd.Flags().Bool("fail-output", true, "This option determines whether the error output for failed file uploads or downloads is enabled. If enabled, the error messages for any failed file transfers will be recorded in a file within the specified directory (if not specified, the default is coscli_output). If disabled, only the number of error files will be output to the console.")
	syncCmd.Flags().String("fail-output-path", "coscli_output", "This option specifies the designated error output folder where the error messages for failed file uploads or downloads will be recorded. By providing a custom folder path, you can control the location and name of the error output folder. If this option is not set, the default error log folder (coscli_output) will be used.")
	syncCmd.Flags().Bool("only-current-dir", false, "Upload only the files in the current directory, ignoring subdirectories and their contents")
	syncCmd.Flags().Bool("disable-all-symlink", true, "Ignore all symbolic link subfiles and symbolic link subdirectories when uploading, not uploaded by default")
	syncCmd.Flags().Bool("enable-symlink-dir", false, "Upload linked subdirectories, not uploaded by default")
	syncCmd.Flags().Bool("disable-crc64", false, "Disable CRC64 data validation. By default, coscli enables CRC64 validation for data transfer")
	syncCmd.Flags().Bool("disable-checksum", false, "Disable overall CRC64 checksum, only validate fragments")
	syncCmd.Flags().Bool("disable-long-links", false, "Disable long links, use short links")
	syncCmd.Flags().Bool("long-links-nums", false, "The long connection quantity parameter, if 0 or not provided, defaults to the concurrent file count.")
	syncCmd.Flags().String("backup-dir", "", "Synchronize deleted file backups, used to save the destination-side files that have been deleted but do not exist on the source side.")
	syncCmd.Flags().Bool("force", false, "Force the operation without prompting for confirmation")
}
