package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"coscli/util"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tencentyun/cos-go-sdk-v5"
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
			logger.Fatalln("--storage-class can only use in upload")
			os.Exit(1)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
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
		backupDir, _ := cmd.Flags().GetString("backup-dir")
		force, _ := cmd.Flags().GetBool("force")

		meta, err := util.MetaStringToHeader(metaString)
		if err != nil {
			logger.Fatalln("Sync invalid meta, reason: " + err.Error())
		}

		if retryNum < 0 || retryNum > 10 {
			logger.Fatalln("retry-num must be between 0 and 10 (inclusive)")
			return
		}

		if errRetryNum < 0 || errRetryNum > 10 {
			logger.Fatalln("err-retry-num must be between 0 and 10 (inclusive)")
			return
		}

		if errRetryInterval < 0 || errRetryInterval > 10 {
			logger.Fatalln("err-retry-interval must be between 0 and 10 (inclusive)")
			return
		}

		srcUrl, err := util.FormatUrl(args[0])
		if err != nil {
			logger.Fatalf("format srcURL error,%v", err)
		}

		destUrl, err := util.FormatUrl(args[1])
		if err != nil {
			logger.Fatalf("format destURL error,%v", err)
		}

		if srcUrl.IsFileUrl() && destUrl.IsFileUrl() {
			logger.Fatalln("not support cp between local directory")
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
		util.InitSnapshotDb(srcUrl, destUrl, fo)
		startT := time.Now().UnixNano() / 1000 / 1000
		if srcUrl.IsFileUrl() && destUrl.IsCosUrl() {
			// 检查错误输出日志是否是本地路径的子集
			err = util.CheckPath(srcUrl, fo, util.TypeFailOutputPath)
			if err != nil {
				logger.Fatalln(err)
			}
			// 格式化上传路径
			util.FormatUploadPath(srcUrl, destUrl, fo)
			// 实例化cos client
			bucketName := destUrl.(*util.CosUrl).Bucket
			c := util.NewClient(fo.Config, fo.Param, bucketName)
			// crc64校验开关
			c.Conf.EnableCRC = fo.Operation.DisableCrc64
			// 上传
			util.SyncUpload(c, srcUrl, destUrl, fo)
		} else if srcUrl.IsCosUrl() && destUrl.IsFileUrl() {
			// 检查错误输出日志是否是本地路径的子集
			err = util.CheckPath(destUrl, fo, util.TypeFailOutputPath)
			if err != nil {
				logger.Fatalln(err)
			}
			bucketName := srcUrl.(*util.CosUrl).Bucket
			c := util.NewClient(fo.Config, fo.Param, bucketName)
			// crc64校验开关
			c.Conf.EnableCRC = fo.Operation.DisableCrc64
			// 格式化下载路径
			util.FormatDownloadPath(srcUrl, destUrl, fo, c)

			// 下载
			util.SyncDownload(c, srcUrl, destUrl, fo)
		} else if srcUrl.IsCosUrl() && destUrl.IsCosUrl() {
			// 实例化来源 cos client
			srcBucketName := srcUrl.(*util.CosUrl).Bucket
			srcClient := util.NewClient(fo.Config, fo.Param, srcBucketName)

			// 实例化目标 cos client
			destBucketName := destUrl.(*util.CosUrl).Bucket
			destClient := util.NewClient(fo.Config, fo.Param, destBucketName)
			// crc64校验开关
			destClient.Conf.EnableCRC = fo.Operation.DisableCrc64

			// 格式化copy路径
			util.FormatCopyPath(srcUrl, destUrl, fo, srcClient, destClient)
			// 拷贝
			util.SyncCosCopy(srcClient, destClient, srcUrl, destUrl, fo)
		} else {
			logger.Fatalln("cospath needs to contain cos://")
		}
		util.CloseErrorOutputFile(fo)
		endT := time.Now().UnixNano() / 1000 / 1000
		util.PrintCostTime(startT, endT)
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
	cpCmd.Flags().Int("retry-num", 0, "Rate-limited retry. Specify 1-10 times. When multiple machines concurrently execute download operations on the same COS directory, rate-limited retry can be performed by specifying this parameter.")
	cpCmd.Flags().Int("err-retry-num", 0, "Error retry attempts. Specify 1-10 times, or 0 for no retry.")
	cpCmd.Flags().Int("err-retry-interval", 0, "Retry interval (available only when specifying error retry attempts 1-10). Specify an interval of 1-10 seconds, or if not specified or set to 0, a random interval within 1-10 seconds will be used for each retry.")
	syncCmd.Flags().Int("routines", 3, "Specifies the number of files concurrent upload or download threads")
	syncCmd.Flags().Bool("fail-output", false, "This option determines whether the error output for failed file uploads or downloads is enabled. If enabled, the error messages for any failed file transfers will be recorded in a file within the specified directory (if not specified, the default is coscli_output). If disabled, only the number of error files will be output to the console.")
	syncCmd.Flags().String("fail-output-path", "coscli_output", "This option specifies the designated error output folder where the error messages for failed file uploads or downloads will be recorded. By providing a custom folder path, you can control the location and name of the error output folder. If this option is not set, the default error log folder (coscli_output) will be used.")
	syncCmd.Flags().Bool("only-current-dir", false, "Upload only the files in the current directory, ignoring subdirectories and their contents")
	syncCmd.Flags().Bool("disable-all-symlink", true, "Ignore all symbolic link subfiles and symbolic link subdirectories when uploading, not uploaded by default")
	syncCmd.Flags().Bool("enable-symlink-dir", false, "Upload linked subdirectories, not uploaded by default")
	syncCmd.Flags().Bool("disable-crc64", false, "Disable CRC64 data validation. By default, coscli enables CRC64 validation for data transfer")
	syncCmd.Flags().String("backup-dir", "", "Synchronize deleted file backups, used to save the destination-side files that have been deleted but do not exist on the source side.")
	syncCmd.Flags().Bool("force", false, "Force the operation without prompting for confirmation")
}

func syncCopy(args []string, recursive bool, include string, exclude string, meta util.Meta, storageClass string) {
	bucketName1, cosPath1 := util.ParsePath(args[0])
	bucketName2, cosPath2 := util.ParsePath(args[1])
	c2 := util.NewClient(&config, &param, bucketName2)
	c1 := util.NewClient(&config, &param, bucketName1)

	if recursive {
		// 路径分隔符
		// 记录是否是代码添加的路径分隔符
		isAddSeparator := false
		// 源路径若不以路径分隔符结尾，则添加
		if !strings.HasSuffix(cosPath1, "/") && cosPath1 != "" {
			isAddSeparator = true
			cosPath1 += "/"
		}
		// 判断cosDir是否是文件夹
		isDir := util.CheckCosPathType(c1, cosPath1, 0)

		if isDir {
			// cosPath1是文件夹 且 cosPath2不以路径分隔符结尾，则添加
			if cosPath2 != "" && !strings.HasSuffix(cosPath2, string(filepath.Separator)) {
				cosPath2 += string(filepath.Separator)
			} else {
				// 若cosPath2以路径分隔符结尾，且cosPath1传入时不以路径分隔符结尾，则需将cos路径的最终文件拼接至local路径最后
				if isAddSeparator {
					fileName := filepath.Base(cosPath1)
					cosPath2 += fileName
					cosPath2 += string(filepath.Separator)
				}
			}
		} else {
			// cosPath1不是文件夹且路径分隔符为代码添加,则去掉路径分隔符
			if isAddSeparator {
				cosPath1 = strings.TrimSuffix(cosPath1, "/")
			}
		}

		objects, _ := util.GetObjectsListRecursive(c1, cosPath1, 0, include, exclude)
		opt := &cos.ObjectCopyOptions{
			ObjectCopyHeaderOptions: &cos.ObjectCopyHeaderOptions{
				CacheControl:       meta.CacheControl,
				ContentDisposition: meta.ContentDisposition,
				ContentEncoding:    meta.ContentEncoding,
				ContentType:        meta.ContentType,
				Expires:            meta.Expires,
				XCosStorageClass:   storageClass,
				XCosMetaXXX:        meta.XCosMetaXXX,
			},
		}

		if meta.CacheControl != "" || meta.ContentDisposition != "" || meta.ContentEncoding != "" ||
			meta.ContentType != "" || meta.Expires != "" || meta.MetaChange {
		}
		{
			opt.ObjectCopyHeaderOptions.XCosMetadataDirective = "Replaced"
		}
		for _, o := range objects {
			srcKey := o.Key
			objName := srcKey[len(cosPath1):]

			// 格式化文件名
			dstKey := cosPath2 + objName
			if objName == "" && (strings.HasSuffix(cosPath2, "/") || cosPath2 == "") {
				fileName := filepath.Base(o.Key)
				dstKey = cosPath2 + fileName
			}

			if dstKey == "" {
				continue
			}
			srcPath := fmt.Sprintf("cos://%s/%s", bucketName1, srcKey)
			dstPath := fmt.Sprintf("cos://%s/%s", bucketName2, dstKey)

			headOpt := &cos.ObjectHeadOptions{
				IfModifiedSince:       "",
				XCosSSECustomerAglo:   "",
				XCosSSECustomerKey:    "",
				XCosSSECustomerKeyMD5: "",
				XOptionHeader:         nil,
			}
			resp, err := c2.Object.Head(context.Background(), dstKey, headOpt)

			// 不存在，则拷贝
			if err != nil {
				if resp != nil && resp.StatusCode == 404 {
					logger.Infoln("Copy", srcPath, "=>", dstPath)

					url := util.GenURL(&config, &param, bucketName1)
					srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, srcKey)

					_, _, err = c2.Object.Copy(context.Background(), dstKey, srcURL, opt)
					if err != nil {
						logger.Fatalln(err)
						os.Exit(1)
					}
				} else {
					logger.Fatalln(err)
					os.Exit(1)
				}
			} else {
				// 存在，判断crc64
				crc1, _, _ := util.ShowHash(c1, srcKey, "crc64")
				crc2, _, _ := util.ShowHash(c2, dstKey, "crc64")
				if crc1 == crc2 {
					logger.Infoln("Skip", srcPath)
				} else {
					logger.Infoln("Copy", srcPath, "=>", dstPath)

					url := util.GenURL(&config, &param, bucketName1)
					srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, srcKey)

					_, _, err = c2.Object.Copy(context.Background(), dstKey, srcURL, opt)
					if err != nil {
						logger.Fatalln(err)
						os.Exit(1)
					}
				}
			}
		}
	} else { // 非递归，单个拷贝

		if len(cosPath1) == 0 {
			logger.Errorln("Invalid srcPath")
			os.Exit(1)
		}

		if strings.HasSuffix(cosPath1, "/") {
			logger.Errorln("srcPath is a dir")
			os.Exit(1)
		}

		if cosPath2 == "" || strings.HasSuffix(cosPath2, "/") {
			fileName := filepath.Base(cosPath1)
			cosPath2 = filepath.Join(cosPath2, fileName)
			args[1] = filepath.Join(args[1], fileName)
		}

		headOpt := &cos.ObjectHeadOptions{
			IfModifiedSince:       "",
			XCosSSECustomerAglo:   "",
			XCosSSECustomerKey:    "",
			XCosSSECustomerKeyMD5: "",
			XOptionHeader:         nil,
		}
		resp, err := c2.Object.Head(context.Background(), cosPath2, headOpt)
		opt := &cos.ObjectCopyOptions{
			ObjectCopyHeaderOptions: &cos.ObjectCopyHeaderOptions{
				CacheControl:       meta.CacheControl,
				ContentDisposition: meta.ContentDisposition,
				ContentEncoding:    meta.ContentEncoding,
				ContentType:        meta.ContentType,
				Expires:            meta.Expires,
				XCosStorageClass:   storageClass,
				XCosMetaXXX:        meta.XCosMetaXXX,
			},
		}

		if meta.CacheControl != "" || meta.ContentDisposition != "" || meta.ContentEncoding != "" ||
			meta.ContentType != "" || meta.Expires != "" || meta.MetaChange {
		}
		{
			opt.ObjectCopyHeaderOptions.XCosMetadataDirective = "Replaced"
		}
		// 不存在，则拷贝
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				logger.Infoln("Copy", args[0], "=>", args[1])
				url := util.GenURL(&config, &param, bucketName1)
				srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, cosPath1)

				_, _, err := c2.Object.Copy(context.Background(), cosPath2, srcURL, opt)
				if err != nil {
					logger.Fatalln(err)
					os.Exit(1)
				}
			} else {
				logger.Fatalln(err)
				os.Exit(1)
			}
		} else {
			// 存在，判断crc64
			crc1, _, _ := util.ShowHash(c1, cosPath1, "crc64")
			crc2, _, _ := util.ShowHash(c2, cosPath2, "crc64")
			if crc1 == crc2 {
				logger.Infoln("Skip", args[0])
			} else {
				logger.Infoln("Copy", args[0], "=>", args[1])

				url := util.GenURL(&config, &param, bucketName1)
				srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, cosPath1)

				_, _, err = c2.Object.Copy(context.Background(), cosPath2, srcURL, opt)
				if err != nil {
					logger.Fatalln(err)
					os.Exit(1)
				}
			}
		}
	}
}
