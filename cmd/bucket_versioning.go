package cmd

import (
	"coscli/util"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var bucketVersioningCmd = &cobra.Command{
	Use:   "bucket-versioning",
	Short: "Modify bucket versioning",
	Long: `Modify bucket versioning

Format:
	./coscli bucket-versioning --method [method] cos://<bucket-name> versioning

Example:
	./coscli bucket-versioning --method put cos://examplebucket versioning
	./coscli bucket-versioning --method get cos://examplebucket`,
	RunE: func(cmd *cobra.Command, args []string) error {
		method, _ := cmd.Flags().GetString("method")

		cosUrl, err := util.FormatUrl(args[0])
		if err != nil {
			return fmt.Errorf("cos url format error:%v", err)
		}

		if !cosUrl.IsCosUrl() {
			return fmt.Errorf("cospath needs to contain cos://")
		}

		bucketName := cosUrl.(*util.CosUrl).Bucket

		c, err := util.NewClient(&config, &param, bucketName)
		if err != nil {
			return err
		}

		if method == "put" {
			if len(args) < 2 {
				return fmt.Errorf("not enough arguments in call to put bucket versioning")
			}
			status := args[1]
			if status != util.VersionStatusEnabled && status != util.VersionStatusSuspended {
				return fmt.Errorf("the bucket versioning status can only be either Suspended or Enabled")
			}

			err := util.PutBucketVersioning(c, status)
			if err != nil {
				return err
			}
			logger.Infof("the bucket versioning status has been changed to %s", status)
		}

		if method == "get" {
			res, err := util.GetBucketVersioning(c)
			if err != nil {
				return err
			}
			switch res.Status {
			case util.VersionStatusEnabled, util.VersionStatusSuspended:
				logger.Infof("bucket versioning status is %s", res.Status)
			default:
				logger.Infof("bucket versioning status is Closed")
			}
		}

		return err
	},
}

func init() {
	rootCmd.AddCommand(bucketVersioningCmd)
	bucketVersioningCmd.Flags().String("method", "", "put/get")
}
