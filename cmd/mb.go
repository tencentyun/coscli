package cmd

import (
	"context"
	"fmt"
	"os"

	"coscli/util"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tencentyun/cos-go-sdk-v5"
)

var mbCmd = &cobra.Command{
	Use:   "mb",
	Short: "Create bucket",
	Long: `Create bucket

Format:
  ./coscli mb cos://<bucket-name>-<appid> -e <endpoint>

Example:
  ./coscli mb cos://examplebucket-1234567890 -e cos.ap-beijing.myqcloud.com`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			return err
		}
		bucketIDName, cosPath := util.ParsePath(args[0])
		if bucketIDName == "" || cosPath != "" {
			return fmt.Errorf("Invalid arguments! ")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		createBucket(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(mbCmd)

	mbCmd.Flags().StringP("region", "r", "", "Region")
	mbCmd.Flags().BoolP("ofs", "o", false, "Ofs")
}

func createBucket(cmd *cobra.Command, args []string) {
	flagRegion, _ := cmd.Flags().GetString("region")
	flagOfs, _ := cmd.Flags().GetBool("ofs")
	if param.Endpoint == "" && flagRegion != "" {
		param.Endpoint = fmt.Sprintf("cos.%s.myqcloud.com", flagRegion)
	}
	bucketIDName, _ := util.ParsePath(args[0])

	c := util.CreateClient(&config, &param, bucketIDName)

	opt := &cos.BucketPutOptions{
		XCosACL:                   "",
		XCosGrantRead:             "",
		XCosGrantWrite:            "",
		XCosGrantFullControl:      "",
		XCosGrantReadACP:          "",
		XCosGrantWriteACP:         "",
		CreateBucketConfiguration: nil,
	}

	if flagOfs {
		opt.CreateBucketConfiguration = &cos.CreateBucketConfiguration{
			BucketArchConfig: "OFS",
		}
	}

	_, err := c.Bucket.Put(context.Background(), opt)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}
	logger.Infof("Create a new bucket! name: %s\n", bucketIDName)
}
