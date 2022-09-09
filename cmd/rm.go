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

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove objects",
	Long: `Remove objects

Format:
  ./coscli rm cos://<bucket-name>[/prefix/] [cos://<bucket-name>[/prefix/]...] [flags]

Example:
  ./coscli rm cos://example/test/ -r`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
			return err
		}
		for _, arg := range args {
			bucketName, _ := util.ParsePath(arg)
			if bucketName == "" {
				return fmt.Errorf("Invalid arguments! ")
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		recursive, _ := cmd.Flags().GetBool("recursive")
		force, _ := cmd.Flags().GetBool("force")
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		if recursive {
			removeObjects1(args, include, exclude, force)
		} else {
			removeObject(args, force)
		}
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)

	rmCmd.Flags().BoolP("recursive", "r", false, "Delete object recursively")
	rmCmd.Flags().BoolP("force", "f", false, "Force delete")
	rmCmd.Flags().String("include", "", "List files that meet the specified criteria")
	rmCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
}

func removeObjects(args []string, include string, exclude string, force bool) {
	for _, arg := range args {
		bucketName, cosDir := util.ParsePath(arg)
		c := util.NewClient(&config, &param, bucketName)

		if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
			cosDir += "/"
		}

		objects, _ := util.GetObjectsListRecursive(c, cosDir, 0, include, exclude)
		if len(objects) == 0 {
			logger.Infoln("No objects were deleted!")
			return
		}

		var oKeys []cos.Object
		for _, o := range objects {
			if !force {
				logger.Infof("Do you want to delete %s? (y/n)", o.Key)
				var choice string
				_, _ = fmt.Scanf("%s\n", &choice)
				if choice == "" || choice == "y" || choice == "Y" || choice == "yes" || choice == "Yes" || choice == "YES" {
					oKeys = append(oKeys, cos.Object{Key: o.Key})
				}
			} else {
				oKeys = append(oKeys, cos.Object{Key: o.Key})
			}
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
		if len(res.Errors) == 0 {
			logger.Infof("\nAll deleted successfully!\n")
		} else {
			logger.Infoln()
			for i, e := range res.Errors {
				logger.Infoln(i+1, ". Fail to delete", e.Key)
				logger.Infoln("    Error Code: ", e.Code, " Message: ", e.Message)
			}
		}
	}
}

func removeObjects1(args []string, include string, exclude string, force bool) {
	for _, arg := range args {
		bucketName, cosDir := util.ParsePath(arg)
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
					recursiveRemoveObject(bucketName, v, force)
				}
				isTruncated = false
			} else {
				isTruncated = t
				nextMarker = m
				var oKeys []cos.Object
				for _, o := range objects {
					if !force {
						logger.Infof("Do you want to delete %s? (y/n)", o.Key)
						var choice string
						_, _ = fmt.Scanf("%s\n", &choice)
						if choice == "" || choice == "y" || choice == "Y" || choice == "yes" || choice == "Yes" || choice == "YES" {
							oKeys = append(oKeys, cos.Object{Key: o.Key})
						}
					} else {
						oKeys = append(oKeys, cos.Object{Key: o.Key})
					}
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
			logger.Infof("\nAll deleted successfully!\n")
		}
	}
}

func removeObject(args []string, force bool) {
	for _, arg := range args {
		bucketName, cosPath := util.ParsePath(arg)
		c := util.NewClient(&config, &param, bucketName)

		opt := &cos.ObjectDeleteOptions{
			XCosSSECustomerAglo:   "",
			XCosSSECustomerKey:    "",
			XCosSSECustomerKeyMD5: "",
			XOptionHeader:         nil,
			VersionId:             "",
		}

		if !force {
			logger.Infof("Do you want to delete %s? (y/n)", cosPath)
			var choice string
			_, _ = fmt.Scanf("%s\n", &choice)
			if choice == "" || choice == "y" || choice == "Y" || choice == "yes" || choice == "Yes" || choice == "YES" {
				_, err := c.Object.Delete(context.Background(), cosPath, opt)
				if err != nil {
					logger.Fatalln(err)
					os.Exit(1)
				}
				logger.Infoln("Delete", arg, "successfully!")
			}
		} else {
			_, err := c.Object.Delete(context.Background(), cosPath, opt)
			if err != nil {
				logger.Fatalln(err)
				os.Exit(1)
			}
			logger.Infoln("Delete", arg, "successfully!")
		}
	}
}

func recursiveRemoveObject(bucketName string, cosPath string, force bool) {
	c := util.NewClient(&config, &param, bucketName)
	opt := &cos.ObjectDeleteOptions{
		XCosSSECustomerAglo:   "",
		XCosSSECustomerKey:    "",
		XCosSSECustomerKeyMD5: "",
		XOptionHeader:         nil,
		VersionId:             "",
	}

	if !force {
		logger.Infof("Do you want to delete %s? (y/n)", cosPath)
		var choice string
		_, _ = fmt.Scanf("%s\n", &choice)
		if choice == "" || choice == "y" || choice == "Y" || choice == "yes" || choice == "Yes" || choice == "YES" {
			_, err := c.Object.Delete(context.Background(), cosPath, opt)
			if err != nil {
				logger.Fatalln(err)
				os.Exit(1)
			}
			logger.Infoln("Delete", "cos://"+bucketName+"/"+cosPath, "successfully!")
		}
	} else {
		_, err := c.Object.Delete(context.Background(), cosPath, opt)
		if err != nil {
			logger.Fatalln(err)
			os.Exit(1)
		}
		logger.Infoln("Delete", "cos://"+bucketName+"/"+cosPath, "successfully!")
	}
}

//获取所有文件和目录
func getFilesAndDirs(c *cos.Client, cosDir string, nextMarker string, include string, exclude string) (files []string) {
	objects, _, _, commonPrefixes := util.GetObjectsListIterator(c, cosDir, nextMarker, include, exclude)
	tempFiles := make([]string, 0)
	tempFiles = append(tempFiles, cosDir)
	for _, v := range objects {
		files = append(files, v.Key)
	}
	if len(commonPrefixes) > 0 {
		for _, v := range commonPrefixes {
			files = append(files, getFilesAndDirs(c, v, nextMarker, include, exclude)...)
		}
	}
	files = append(files, tempFiles...)
	return files
}
