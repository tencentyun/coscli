package cmd

import (
	"fmt"
	logger "github.com/sirupsen/logrus"
	"os"
	"strings"
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
		maxThreadNum, _ := cmd.Flags().GetInt("max-thread-num")
		metaString, _ := cmd.Flags().GetString("meta")
		retryNum, _ := cmd.Flags().GetInt("retry-num")
		errRetryNum, _ := cmd.Flags().GetInt("err-retry-num")
		errRetryInterval, _ := cmd.Flags().GetInt("err-retry-interval")
		snapshotPath, _ := cmd.Flags().GetString("snapshot-path")
		delete, _ := cmd.Flags().GetBool("delete")
		routines, _ := cmd.Flags().GetInt("routines")
		failOutput, _ := cmd.Flags().GetBool("fail-output")
		failOutputPath, _ := cmd.Flags().GetString("fail-output-path")
		processLog, _ := cmd.Flags().GetBool("process-log")
		processLogPath, _ := cmd.Flags().GetString("process-log-path")
		onlyCurrentDir, _ := cmd.Flags().GetBool("only-current-dir")
		disableAllSymlink, _ := cmd.Flags().GetBool("disable-all-symlink")
		enableSymlinkDir, _ := cmd.Flags().GetBool("enable-symlink-dir")
		disableCrc64, _ := cmd.Flags().GetBool("disable-crc64")
		disableChecksum, _ := cmd.Flags().GetBool("disable-checksum")
		disableLongLinks, _ := cmd.Flags().GetBool("disable-long-links")
		longLinksNums, _ := cmd.Flags().GetInt("long-links-nums")
		backupDir, _ := cmd.Flags().GetString("backup-dir")
		force, _ := cmd.Flags().GetBool("force")
		skipDir, _ := cmd.Flags().GetBool("skip-dir")
		update, _ := cmd.Flags().GetBool("update")
		ignoreExisting, _ := cmd.Flags().GetBool("ignore-existing")
		acl, _ := cmd.Flags().GetString("acl")
		grantRead, _ := cmd.Flags().GetString("grant-read")
		//grantWrite, _ := cmd.Flags().GetString("grant-write")
		grantReadAcp, _ := cmd.Flags().GetString("grant-read-acp")
		grantWriteAcp, _ := cmd.Flags().GetString("grant-write-acp")
		grantFullControl, _ := cmd.Flags().GetString("grant-full-control")
		tags, _ := cmd.Flags().GetString("tags")
		forbidOverwrite, _ := cmd.Flags().GetBool("forbid-overwrite")
		encryptionType, _ := cmd.Flags().GetString("encryption-type")
		serverSideEncryption, _ := cmd.Flags().GetString("server-side-encryption")
		sseCustomerAlgo, _ := cmd.Flags().GetString("sse-customer-algo")
		sseCustomerKey, _ := cmd.Flags().GetString("sse-customer-key")
		sseCustomerKeyMD5, _ := cmd.Flags().GetString("sse-customer-key-md5")
		checkPoint, _ := cmd.Flags().GetBool("check-point")
		ignoreEmptyFile, _ := cmd.Flags().GetBool("ignore-empty-file")

		// 服务端加密参数验证
		encryptionType = strings.ToUpper(encryptionType)
		if encryptionType == "SSE-COS" {
			// 当 encryptionType 为 SSE-COS 时，将 SSE-C 的参数设为空
			sseCustomerAlgo = ""
			sseCustomerKey = ""
			sseCustomerKeyMD5 = ""
		} else if encryptionType == "SSE-C" {
			// 当 encryptionType 为 SSE-C 时，将 ServerSideEncryption 参数设为空
			serverSideEncryption = ""
		} else if encryptionType == "" {
			// 当 encryptionType 为空时，将所有加密相关参数设置为空
			serverSideEncryption = ""
			sseCustomerAlgo = ""
			sseCustomerKey = ""
			sseCustomerKeyMD5 = ""
		} else {
			// 如果 encryptionType 为其他非法值，报错并退出
			return fmt.Errorf("error: encryptionType must be either 'SSE-COS' or 'SSE-C'")
		}

		meta, err := util.MetaStringToHeader(metaString)
		if err != nil {
			return fmt.Errorf("Sync invalid meta, reason: " + err.Error())
		}

		if retryNum < 0 || retryNum > 100 {
			return fmt.Errorf("retry-num must be between 0 and 100 (inclusive)")
		}

		if errRetryNum < 0 || errRetryNum > 100 {
			return fmt.Errorf("err-retry-num must be between 0 and 100 (inclusive)")
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

		// 解析tags
		tags, err = util.EncodeTagging(tags)
		if err != nil {
			return err
		}

		_, filters := util.GetFilter(include, exclude)

		if delete && !recursive {
			return fmt.Errorf("delete can only use with --recursive option")
		}

		fo := &util.FileOperations{
			Operation: util.Operation{
				Recursive:         recursive,
				Filters:           filters,
				StorageClass:      storageClass,
				RateLimiting:      rateLimiting,
				PartSize:          partSize,
				ThreadNum:         threadNum,
				MaxThreadNum:      maxThreadNum,
				Routines:          routines,
				FailOutput:        failOutput,
				FailOutputPath:    failOutputPath,
				Meta:              meta,
				RetryNum:          retryNum,
				ErrRetryNum:       errRetryNum,
				ErrRetryInterval:  errRetryInterval,
				ProcessLog:        processLog,
				ProcessLogPath:    processLogPath,
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
				SkipDir:           skipDir,
				Update:            update,
				IgnoreExisting:    ignoreExisting,
				Acl:               acl,
				GrantRead:         grantRead,
				//GrantWrite:        grantWrite,
				GrantReadAcp:         grantReadAcp,
				GrantWriteAcp:        grantWriteAcp,
				GrantFullControl:     grantFullControl,
				Tags:                 tags,
				ForbidOverWrite:      forbidOverwrite,
				ServerSideEncryption: serverSideEncryption,
				SSECustomerAlgo:      sseCustomerAlgo,
				SSECustomerKey:       sseCustomerKey,
				SSECustomerKeyMD5:    sseCustomerKeyMD5,
				CheckPoint:           checkPoint,
				IgnoreEmptyFile:      ignoreEmptyFile,
			},
			Monitor:       &util.FileProcessMonitor{},
			Config:        &config,
			Param:         &param,
			ErrOutput:     &util.ErrOutput{},
			ProcessLogger: &util.ProcessLogger{},
			CpType:        getCommandType(srcUrl, destUrl),
			Command:       util.CommandSync,
			OutPutDirName: time.Now().Format("20060102_150405"),
		}

		// 快照db实例化
		err = util.InitSnapshotDb(srcUrl, destUrl, fo)
		if err != nil {
			return err
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
			// 检查进程日志是否是本地路径的子集
			err = util.CheckPath(srcUrl, fo, util.TypeProcessLogPath)
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
			// 获取桶类型
			fo.BucketType, err = util.GetBucketType(c, fo.Param, fo.Config, bucketName)
			if err != nil {
				return err
			}
			// 上传
			err = util.SyncUpload(c, srcUrl, destUrl, fo)
			if err != nil {
				return err
			}
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
			// 检查进程日志是否是本地路径的子集
			err = util.CheckPath(destUrl, fo, util.TypeProcessLogPath)
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

			// 获取桶类型
			fo.BucketType, err = util.GetBucketType(c, fo.Param, fo.Config, bucketName)
			if err != nil {
				return err
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
			operate = "Copy"
			logger.Infof("Copy %s to %s start", srcPath, destPath)
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

			// 获取桶类型
			fo.BucketType, err = util.GetBucketType(srcClient, fo.Param, fo.Config, srcBucketName)
			if err != nil {
				return err
			}

			// 获取目标桶类型（copy 请求发往目标桶，是否可带 versionId 由目标桶决定）
			fo.DstBucketType, err = util.GetBucketType(destClient, fo.Param, fo.Config, destBucketName)
			if err != nil {
				return err
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
		util.CloseProcessLoggerFile(fo)
		endT := time.Now().UnixNano() / 1000 / 1000
		util.PrintCostTime(startT, endT)

		if fo.Monitor.ErrNum > 0 || fo.Monitor.ListErrNum > 0 {
			logger.Warningf("%s %s to %s %s", operate, srcPath, destPath, fo.Monitor.GetFinishInfo())
			os.Exit(2)
		} else {
			logger.Infof("%s %s to %s %s", operate, srcPath, destPath, fo.Monitor.GetFinishInfo())
		}
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
	syncCmd.Flags().Int("thread-num", 0, "Specifies the number of concurrent upload or download threads. When set (>0), it overrides auto-derivation by file size and ignores --max-thread-num.")
	syncCmd.Flags().Int("max-thread-num", 32, "Upper bound for auto-derived partition concurrency when --thread-num is 0. Has no effect if --thread-num is explicitly set.")
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
	syncCmd.Flags().Int("retry-num", 0, "Rate-limited retry. Specify 1-100 times. When multiple machines concurrently execute download operations on the same COS directory, rate-limited retry can be performed by specifying this parameter.")
	syncCmd.Flags().Int("err-retry-num", 5, "Error retry attempts. Specify 1-100 times, or 0 for no retry.")
	syncCmd.Flags().Int("err-retry-interval", 0, "Retry interval (available only when specifying error retry attempts 1-10). Specify an interval of 1-10 seconds, or if not specified or set to 0, a random interval within 1-10 seconds will be used for each retry.")
	syncCmd.Flags().Int("routines", 3, "Specifies the number of files concurrent upload or download threads")
	syncCmd.Flags().Bool("fail-output", true, "This option determines whether the error output for failed file uploads or downloads is enabled. If enabled, the error messages for any failed file transfers will be recorded in a file within the specified directory (if not specified, the default is coscli_output). If disabled, only the number of error files will be output to the console.")
	syncCmd.Flags().String("fail-output-path", "coscli_output", "This option specifies the designated error output folder where the error messages for failed file uploads or downloads will be recorded. By providing a custom folder path, you can control the location and name of the error output folder. If this option is not set, the default error log folder (coscli_output) will be used.")
	syncCmd.Flags().Bool("process-log", true, "This option determines whether process log recording is enabled. If enabled, information related to file upload or download processes (including error details) will be recorded in a log file within the specified directory (if not specified, the default is coscli_output). If disabled, only the number of error files and basic process information will be output to the console.")
	syncCmd.Flags().String("process-log-path", "coscli_output", "This option is used to specify a dedicated output folder for process logs. The logs will record information related to file uploads or downloads, including errors and process details. By providing a custom folder path, you can control the location and name of the log output folder. If this option is not set, the default log folder (coscli_output) will be used.")
	syncCmd.Flags().Bool("only-current-dir", false, "Upload only the files in the current directory, ignoring subdirectories and their contents")
	syncCmd.Flags().Bool("disable-all-symlink", true, "Ignore all symbolic link subfiles and symbolic link subdirectories when uploading, not uploaded by default")
	syncCmd.Flags().Bool("enable-symlink-dir", false, "Upload linked subdirectories, not uploaded by default")
	syncCmd.Flags().Bool("disable-crc64", false, "Disable CRC64 data validation. By default, coscli enables CRC64 validation for data transfer")
	syncCmd.Flags().Bool("disable-checksum", true, "Disable overall CRC64 checksum, only validate fragments")
	syncCmd.Flags().Bool("disable-long-links", false, "Disable long links, use short links")
	syncCmd.Flags().Int("long-links-nums", 0, "The long connection quantity parameter, if 0 or not provided, defaults to the concurrent file count.")
	syncCmd.Flags().String("backup-dir", "", "Synchronize deleted file backups, used to save the destination-side files that have been deleted but do not exist on the source side.")
	syncCmd.Flags().Bool("force", false, "Force the operation without prompting for confirmation")
	syncCmd.Flags().Bool("skip-dir", false, "Skip folders during upload.")
	syncCmd.Flags().Bool("update", false, "Only when the target file does not exist will the operation be executed, or if the source file's last modified time is later than the target file's.")
	syncCmd.Flags().Bool("ignore-existing", false, "Only when the target file does not exist will the operation be executed.")
	syncCmd.Flags().String("acl", "", "Defines the Access Control List (ACL) property of an object. The default value is default.")
	syncCmd.Flags().String("grant-read", "", "Grants the grantee permission to read the object. The format is id=\"[OwnerUin]\", for example, id=\"100000000001\". Multiple grantees can be specified using commas (,), for example, id=\"100000000001\",id=\"100000000002\".")
	//syncCmd.Flags().String("grant-write", "", "Grants the grantee permission to write the object. The format is id=\"[OwnerUin]\", for example, id=\"100000000001\". Multiple grantees can be specified using commas (,), for example, id='100000000001',id=\"100000000002\".")
	syncCmd.Flags().String("grant-read-acp", "", "Grants the grantee permission to read the object's Access Control List (ACL). The format is id=\"[OwnerUin]\", for example, id=\"100000000001\". Multiple grantees can be specified using commas (,), for example, id=\"100000000001\",id=\"100000000002\".")
	syncCmd.Flags().String("grant-write-acp", "", "Grants the grantee permission to write the object's Access Control List (ACL). The format is id=\"[OwnerUin]\", for example, id=\"100000000001\". Multiple grantees can be specified using commas (,), for example, id=\"100000000001\",id=\"100000000002\".")
	syncCmd.Flags().String("grant-full-control", "", "Grants the grantee full permissions to operate on the object. The format is id=\"[OwnerUin]\", for example, id=\"100000000001\". Multiple grantees can be specified using commas (,), for example, id=\"100000000001\",id=\"100000000002\".")
	syncCmd.Flags().String("tags", "", "The set of tags for the object, with a maximum of 10 tags (e.g., Key1=Value1 & Key2=Value2). The Key and Value in the tag set must be URL-encoded beforehand.")
	syncCmd.Flags().Bool("forbid-overwrite", false, "For storage buckets without versioning enabled, if not specified or set to false, uploading will overwrite objects with the same name by default; if set to true, overwriting objects with the same name is prohibited.")
	syncCmd.Flags().String("encryption-type", "", "Server-side encryption methods, optional values: SSE-COS and SSE-C.")
	syncCmd.Flags().String("server-side-encryption", "", "SSE-COS mode supports two encryption algorithms: AES256 and SM4.")
	syncCmd.Flags().String("sse-customer-algo", "", "SSE-C encryption refers to server-side encryption with customer-provided keys. The encryption keys are provided by the user, and when uploading objects, COS will use the user-provided encryption keys to encrypt the user's data. The SSE-C mode supports two encryption algorithms: AES256 and SM4.")
	syncCmd.Flags().String("sse-customer-key", "", "The user-provided key should be a 32-byte string, supporting combinations of numbers, letters, and special characters. Chinese characters are not supported.")
	syncCmd.Flags().String("sse-customer-key-md5", "", "The MD5 value of the user-provided key")
	syncCmd.Flags().Bool("check-point", true, "Whether to enable breakpoint resume, default is true, enable breakpoint resume.")
	syncCmd.Flags().Bool("ignore-empty-file", false, "This parameter will ignore zero-byte files.")
}
