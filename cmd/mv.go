package cmd

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"

	"coscli/util"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tencentyun/cos-go-sdk-v5"
)

var mvCmd = &cobra.Command{
	Use:   "mv",
	Short: "Move objects",
	Long: `Move objects

Format:
  ./coscli mv <source_path> <destination_path> [flags]

Example: 
  Move:
    ./coscli mv ~/example.txt cos://examplebucket/example.txt`,
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
		metaString, _ := cmd.Flags().GetString("meta")
		meta, err := util.MetaStringToHeader(metaString)
		if err != nil {
			logger.Fatalln("Move invalid meta " + err.Error())
		}
		// args[0]: 源地址
		// args[1]: 目标地址
		if util.IsCosPath(args[0]) && util.IsCosPath(args[1]) {
			bucketIDNameSource, _ := util.ParsePath(args[0])
			bucketIDNameDest, _ := util.ParsePath(args[1])
			if bucketIDNameSource == bucketIDNameDest {
				// 移动
				move(args, recursive, include, exclude, meta, storageClass)
			} else {
				logger.Fatalln("cospath needs the same bucket")
			}
		} else {
			logger.Fatalln("cospath needs to contain cos://")
		}
	},
}

func init() {
	rootCmd.AddCommand(mvCmd)

	mvCmd.Flags().BoolP("recursive", "r", false, "Move objects recursively")
	mvCmd.Flags().String("include", "", "Include files that meet the specified criteria")
	mvCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
	mvCmd.Flags().String("storage-class", "", "Specifying a storage class")
	mvCmd.Flags().String("meta", "",
		"Set the meta information of the file, "+
			"the format is header:value#header:value, the example is Cache-Control:no-cache#Content-Encoding:gzip")
}

func move(args []string, recursive bool, include string, exclude string, meta util.Meta, storageClass string) {
	bucketName, cosPath1 := util.ParsePath(args[0])
	_, cosPath2 := util.ParsePath(args[1])
	c := util.NewClient(&config, &param, bucketName)

	s, _ := c.Bucket.Head(context.Background())
	// 根据s.Header判断是否是融合桶或者普通桶
	if s.Header.Get("X-Cos-Bucket-Arch") == "OFS" {
		srcPath := fmt.Sprintf("cos://%s/%s", bucketName, cosPath1)
		dstPath := fmt.Sprintf("cos://%s/%s", bucketName, cosPath2)
		logger.Infoln("Move", srcPath, "=>", dstPath)

		url := util.GenURL(&config, &param, bucketName)
		dstURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, cosPath2)

		var closeBody bool = true

		//dstURL:tina-coscli-test-123/x
		//cosPath1:ofs
		_, err := util.PutRename(context.Background(), &config, &param, c, cosPath1, dstURL, closeBody)
		if err != nil {
			logger.Fatalln(err)
			os.Exit(1)
		}
		logger.Infof("\nAll move successfully!\n")
	} else {
		cosCopy(args, recursive, include, exclude, meta, storageClass)
		if recursive {
			moveObjects(args, include, exclude)
		} else {
			moveObject(args)
		}
	}
}

func moveObjects(args []string, include string, exclude string) {
	bucketName, cosDir := util.ParsePath(args[0])
	c := util.NewClient(&config, &param, bucketName)

	if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
		cosDir += "/"
	}

	isTruncated := true
	nextMarker := ""
	deleteOrNot := false
	errorOrNot := false
	for isTruncated {
		objects, t, m, commonPrefixes := util.GetObjectsListIterator(c, cosDir, nextMarker, include, exclude)

		if len(commonPrefixes) > 0 {
			files := getFilesAndDirs(c, cosDir, nextMarker, include, exclude)

			for _, v := range files {
				recursivemoveObject(bucketName, v)
			}
			isTruncated = false
		} else {
			isTruncated = t
			nextMarker = m
			var oKeys []cos.Object
			for _, o := range objects {
				oKeys = append(oKeys, cos.Object{Key: o.Key})
			}
			if len(oKeys) > 0 {
				deleteOrNot = true
			}
			opt := &cos.ObjectDeleteMultiOptions{
				XMLName: xml.Name{},
				Quiet:   false,
				Objects: oKeys,
			}
			res, _, err := c.Object.DeleteMulti(context.Background(), opt)
			if err != nil {
				logger.Fatalln(err)
				os.Exit(1)
			}

			for _, o := range res.DeletedObjects {
				logger.Infoln("Delete ", o.Key)
			}
			if len(res.Errors) > 0 {
				errorOrNot = true
				logger.Infoln()
				for _, e := range res.Errors {
					logger.Infoln("Fail to delete", e.Key)
					logger.Infoln("    Error Code: ", e.Code, " Message: ", e.Message)
				}
			}
		}
	}

	if deleteOrNot == false {
		logger.Infoln("No objects were deleted!")
	}
	if errorOrNot == false {
		logger.Infof("\nAll move successfully!\n")
	}
}
func moveObject(args []string) {
	bucketName, cosPath := util.ParsePath(args[0])
	c := util.NewClient(&config, &param, bucketName)

	opt := &cos.ObjectDeleteOptions{
		XCosSSECustomerAglo:   "",
		XCosSSECustomerKey:    "",
		XCosSSECustomerKeyMD5: "",
		XOptionHeader:         nil,
		VersionId:             "",
	}
	_, err := c.Object.Delete(context.Background(), cosPath, opt)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}
	logger.Infoln("Delete", args[0], "successfully!")
	logger.Infof("\n Move successfully!\n")
}

func recursivemoveObject(bucketName string, cosPath string) {
	c := util.NewClient(&config, &param, bucketName)
	opt := &cos.ObjectDeleteOptions{
		XCosSSECustomerAglo:   "",
		XCosSSECustomerKey:    "",
		XCosSSECustomerKeyMD5: "",
		XOptionHeader:         nil,
		VersionId:             "",
	}

	_, err := c.Object.Delete(context.Background(), cosPath, opt)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}
}
