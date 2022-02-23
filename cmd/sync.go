package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"os"

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
			_, _ = fmt.Fprintln(os.Stderr, "--storage-class can only use in upload")
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
			syncUpload(args, recursive, include, exclude, op)
		}
		if util.IsCosPath(args[0]) && !util.IsCosPath(args[1]) {
			// 下载
			op := &util.DownloadOptions{
				RateLimiting: rateLimiting,
				PartSize:     partSize,
				ThreadNum:    threadNum,
			}
			syncDownload(args, recursive, include, exclude, op)
		}
		if util.IsCosPath(args[0]) && util.IsCosPath(args[1]) {
			// 拷贝
			syncCopy(args, recursive, include, exclude)
		}
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
}

func syncUpload(args []string, recursive bool, include string, exclude string, op *util.UploadOptions) {
	_, localPath := util.ParsePath(args[0])
	bucketName, cosPath := util.ParsePath(args[1])
	c := util.NewClient(&config, &param, bucketName)

	if recursive {
		util.SyncMultiUpload(c, localPath, bucketName, cosPath, include, exclude, op)
	} else {
		util.SyncSingleUpload(c, localPath, bucketName, cosPath, op)
	}
}

func syncDownload(args []string, recursive bool, include string, exclude string, op *util.DownloadOptions) {
	bucketName, cosPath := util.ParsePath(args[0])
	_, localPath := util.ParsePath(args[1])
	c := util.NewClient(&config, &param, bucketName)

	if recursive {
		util.SyncMultiDownload(c, bucketName, cosPath, localPath, include, exclude, op)
	} else {
		util.SyncSingleDownload(c, bucketName, cosPath, localPath, op)
	}
}

func syncCopy(args []string, recursive bool, include string, exclude string) {
	bucketName1, cosPath1 := util.ParsePath(args[0])
	bucketName2, cosPath2 := util.ParsePath(args[1])
	c2 := util.NewClient(&config, &param, bucketName2)
	c1 := util.NewClient(&config, &param, bucketName1)

	if recursive {
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
					fmt.Println("Copy", srcPath, "=>", dstPath)

					url := util.GenURL(&config, &param, bucketName1)
					srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, srcKey)

					_, _, err = c2.Object.Copy(context.Background(), dstKey, srcURL, nil)
					if err != nil {
						_, _ = fmt.Fprintln(os.Stderr, err)
						os.Exit(1)
					}
				} else {
					_, _ = fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			} else {
				// 存在，判断crc64
				crc1, _ := util.ShowHash(c1, srcKey, "crc64")
				crc2, _ := util.ShowHash(c2, dstKey, "crc64")
				if crc1 == crc2 {
					fmt.Println("Skip", srcPath)
				} else {
					fmt.Println("Copy", srcPath, "=>", dstPath)

					url := util.GenURL(&config, &param, bucketName1)
					srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, srcKey)

					_, _, err = c2.Object.Copy(context.Background(), dstKey, srcURL, nil)
					if err != nil {
						_, _ = fmt.Fprintln(os.Stderr, err)
						os.Exit(1)
					}
				}
			}
		}
	} else { // 非递归，单个拷贝
		if cosPath2 == "" || cosPath2[len(cosPath2)-1] == '/' {
			fmt.Println("When copying a single file, you need to specify a full path")
			os.Exit(1)
		}

		headOpt := &cos.ObjectHeadOptions{
			IfModifiedSince:       "",
			XCosSSECustomerAglo:   "",
			XCosSSECustomerKey:    "",
			XCosSSECustomerKeyMD5: "",
			XOptionHeader:         nil,
		}
		resp, err := c2.Object.Head(context.Background(), cosPath2, headOpt)

		// 不存在，则拷贝
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				fmt.Println("Copy", args[0], "=>", args[1])
				url := util.GenURL(&config, &param, bucketName1)
				srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, cosPath1)

				_, _, err := c2.Object.Copy(context.Background(), cosPath2, srcURL, nil)
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			} else {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		} else {
			// 存在，判断crc64
			crc1, _ := util.ShowHash(c1, cosPath1, "crc64")
			crc2, _ := util.ShowHash(c2, cosPath2, "crc64")
			if crc1 == crc2 {
				fmt.Println("Skip", args[0])
			} else {
				fmt.Println("Copy", args[0], "=>", args[1])

				url := util.GenURL(&config, &param, bucketName1)
				srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, cosPath1)

				_, _, err = c2.Object.Copy(context.Background(), cosPath2, srcURL, nil)
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			}
		}
	}
}
