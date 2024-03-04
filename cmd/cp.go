package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"coscli/util"

	"github.com/tencentyun/cos-go-sdk-v5"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var cpCmd = &cobra.Command{
	Use:   "cp",
	Short: "Upload, download or copy objects",
	Long: `Upload, download or copy objects

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
		meta, err := util.MetaStringToHeader(metaString)
		if err != nil {
			logger.Fatalln("Copy invalid meta " + err.Error())
		}
		// args[0]: 源地址
		// args[1]: 目标地址
		if !util.IsCosPath(args[0]) && util.IsCosPath(args[1]) {
			// 上传
			op := &util.UploadOptions{
				StorageClass: storageClass,
				RateLimiting: rateLimiting,
				PartSize:     partSize,
				ThreadNum:    threadNum,
				Meta:         meta,
			}
			upload(args, recursive, include, exclude, op)
		} else if util.IsCosPath(args[0]) && !util.IsCosPath(args[1]) {
			// 下载
			op := &util.DownloadOptions{
				RateLimiting: rateLimiting,
				PartSize:     partSize,
				ThreadNum:    threadNum,
			}
			download(args, recursive, include, exclude, op)
		} else if util.IsCosPath(args[0]) && util.IsCosPath(args[1]) {
			// 拷贝
			cosCopy(args, recursive, include, exclude, meta, storageClass)
		} else {
			logger.Fatalln("cospath needs to contain cos://")
		}
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
	cpCmd.Flags().Int("thread-num", 5, "Specifies the number of concurrent upload or download threads")
	cpCmd.Flags().String("meta", "",
		"Set the meta information of the file, "+
			"the format is header:value#header:value, the example is Cache-Control:no-cache#Content-Encoding:gzip")
}

func upload(args []string, recursive bool, include string, exclude string, op *util.UploadOptions) {
	_, localPath := util.ParsePath(args[0])
	bucketName, cosPath := util.ParsePath(args[1])
	c := util.NewClient(&config, &param, bucketName)

	if recursive {
		util.MultiUpload(c, localPath, bucketName, cosPath, include, exclude, op)
	} else {
		util.SingleUpload(c, localPath, bucketName, cosPath, op)
	}
}

func download(args []string, recursive bool, include string, exclude string, op *util.DownloadOptions) {
	bucketName, cosPath := util.ParsePath(args[0])
	_, localPath := util.ParsePath(args[1])
	c := util.NewClient(&config, &param, bucketName)

	if recursive {
		util.MultiDownload(c, bucketName, cosPath, localPath, include, exclude, op)
	} else {
		util.SingleDownload(c, bucketName, cosPath, localPath, op, false)
	}
}

func cosCopy(args []string, recursive bool, include string, exclude string, meta util.Meta, storageClass string) {
	bucketName1, cosPath1 := util.ParsePath(args[0])
	bucketName2, cosPath2 := util.ParsePath(args[1])
	c2 := util.NewClient(&config, &param, bucketName2)

	if recursive {
		c1 := util.NewClient(&config, &param, bucketName1)
		// 路径分隔符
		// 记录是否是代码添加的路径分隔符
		isAddSeparator := false
		// 源路径若不以路径分隔符结尾，则添加
		if !strings.HasSuffix(cosPath1, "/")  && cosPath1 != ""{
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
			logger.Infoln("Copy", srcPath, "=>", dstPath)

			url := util.GenURL(&config, &param, bucketName1)
			srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, srcKey)

			_, _, err := c2.Object.Copy(context.Background(), dstKey, srcURL, opt)
			if err != nil {
				logger.Fatalln(err)
				os.Exit(1)
			}
		}
	} else {

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

		logger.Infoln("Copy", args[0], "=>", args[1])
		url := util.GenURL(&config, &param, bucketName1)
		srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, cosPath1)

		_, _, err := c2.Object.Copy(context.Background(), cosPath2, srcURL, nil)
		if err != nil {
			logger.Fatalln(err)
			os.Exit(1)
		}
	}
}
