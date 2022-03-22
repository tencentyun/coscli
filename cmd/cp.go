package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"os"

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

		// args[0]: 源地址
		// args[1]: 目标地址
		if !util.IsCosPath(args[0]) && util.IsCosPath(args[1]) {
			// 上传
			op := &util.UploadOptions{
				StorageClass: storageClass,
				RateLimiting: rateLimiting,
				PartSize:     partSize,
				ThreadNum:    threadNum,
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
			cosCopy(args, recursive, include, exclude)
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
		util.SingleDownload(c, bucketName, cosPath, localPath, op)
	}
}

func cosCopy(args []string, recursive bool, include string, exclude string) {
	bucketName1, cosPath1 := util.ParsePath(args[0])
	bucketName2, cosPath2 := util.ParsePath(args[1])
	c2 := util.NewClient(&config, &param, bucketName2)

	if recursive {
		c1 := util.NewClient(&config, &param, bucketName1)

		if cosPath1 != "" && cosPath1[len(cosPath1)-1] != '/' {
			cosPath1 += "/"
		}
		if cosPath2 != "" && cosPath2[len(cosPath2)-1] != '/' {
			cosPath2 += "/"
		}

		objects := util.GetObjectsListRecursive(c1, cosPath1, 0, include, exclude)

		for _, o := range objects {
			srcKey := o.Key
			dstKey := cosPath2 + srcKey[len(cosPath1):]
			srcPath := fmt.Sprintf("cos://%s/%s", bucketName1, srcKey)
			dstPath := fmt.Sprintf("cos://%s/%s", bucketName2, dstKey)
			logger.Infoln("Copy", srcPath, "=>", dstPath)

			url := util.GenURL(&config, &param, bucketName1)
			srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, srcKey)

			_, _, err := c2.Object.Copy(context.Background(), dstKey, srcURL, nil)
			if err != nil {
				logger.Fatalln(err)
				os.Exit(1)
			}
		}
	} else {
		if cosPath2 == "" || cosPath2[len(cosPath2)-1] == '/' {
			logger.Infoln("When copying a single file, you need to specify a full path")
			os.Exit(1)
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
