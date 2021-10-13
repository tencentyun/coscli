package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var cpCmd = &cobra.Command{
	Use:   "cp",
	Short: "Upload, download or copy objects",
	Long:  `Upload, download or copy objects

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
		// args[0]: 源地址
		// args[1]: 目标地址
		if !util.IsCosPath(args[0]) && util.IsCosPath(args[1]) {
			// 上传
			upload(args, recursive, include, exclude, storageClass)
		}
		if util.IsCosPath(args[0]) && !util.IsCosPath(args[1]) {
			// 下载
			download(args, recursive, include, exclude)
		}
		if util.IsCosPath(args[0]) && util.IsCosPath(args[1]) {
			// 拷贝
			cosCopy(args, recursive, include, exclude)
		}
	},
}

func init() {
	rootCmd.AddCommand(cpCmd)

	cpCmd.Flags().BoolP("recursive", "r", false, "Copy objects recursively")
	cpCmd.Flags().String("include", "", "Include files that meet the specified criteria")
	cpCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
	cpCmd.Flags().String("storage-class", "", "Specifying a storage class")
}

func upload(args []string, recursive bool, include string, exclude string, storageClass string) {
	_, localPath := util.ParsePath(args[0])
	bucketName, cosPath := util.ParsePath(args[1])
	c := util.NewClient(&config, bucketName)

	if recursive {
		util.MultiUpload(c, localPath, bucketName, cosPath, include, exclude, storageClass)
	} else {
		util.SingleUpload(c, localPath, bucketName, cosPath, storageClass)
	}
}

func download(args []string, recursive bool, include string, exclude string) {
	bucketName, cosPath := util.ParsePath(args[0])
	_, localPath := util.ParsePath(args[1])
	c := util.NewClient(&config, bucketName)

	if recursive {
		util.MultiDownload(c, bucketName, cosPath, localPath, include, exclude)
	} else {
		util.SingleDownload(c, bucketName, cosPath, localPath)
	}
}

func cosCopy(args []string, recursive bool, include string, exclude string) {
	bucketName1, cosPath1 := util.ParsePath(args[0])
	bucketName2, cosPath2 := util.ParsePath(args[1])
	c2 := util.NewClient(&config, bucketName2)

	if recursive {
		c1 := util.NewClient(&config, bucketName1)

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
			fmt.Println("Copy", srcPath, "=>", dstPath)

			url := util.GenURL(&config, bucketName1)
			srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, srcKey)

			_, _, err := c2.Object.Copy(context.Background(), dstKey, srcURL, nil)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	} else {
		if cosPath2 == "" || cosPath2[len(cosPath2)-1] == '/' {
			fmt.Println("When copying a single file, you need to specify a full path")
			os.Exit(1)
		}
		
		fmt.Println("Copy", args[0], "=>", args[1])
		url := util.GenURL(&config, bucketName1)
		srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, cosPath1)

		_, _, err := c2.Object.Copy(context.Background(), cosPath2, srcURL, nil)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
