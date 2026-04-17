package cmd

import (
	"coscli/util"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var statCmd = &cobra.Command{
	Use:   "stat",
	Short: "Query the metadata of an object",
	Long: `Query the metadata of an object

Format:
  ./coscli stat cos://<bucket-name>[/<key>] [flags]

Example:
  ./coscli stat cos://examplebucket-1234567890/test.txt
  ./coscli stat cos://examplebucket-1234567890/test.txt --version-id <version-id>`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		versionId, _ := cmd.Flags().GetString("version-id")

		cosUrl, err := util.FormatUrl(args[0])
		if err != nil {
			return fmt.Errorf("cos url format error: %v", err)
		}
		if !cosUrl.IsCosUrl() {
			return fmt.Errorf("cospath needs to contain cos://")
		}

		bucketName := cosUrl.(*util.CosUrl).Bucket
		objectKey := cosUrl.(*util.CosUrl).Object

		if objectKey == "" {
			return fmt.Errorf("object key cannot be empty, please specify a valid object path")
		}

		c, err := util.NewClient(&config, &param, bucketName)
		if err != nil {
			return err
		}

		info, err := util.StatObject(c, objectKey, versionId)
		if err != nil {
			return err
		}

		logger.Infof("Object: cos://%s/%s", bucketName, objectKey)
		logger.Infof("  ETag:                 %s", info.ETag)
		logger.Infof("  Content-Type:         %s", info.ContentType)
		logger.Infof("  Content-Length:       %s", info.ContentLength)
		logger.Infof("  Last-Modified:        %s", info.LastModified)
		logger.Infof("  Cache-Control:        %s", info.CacheControl)
		logger.Infof("  Content-Disposition:  %s", info.ContentDisposition)
		logger.Infof("  Content-Encoding:     %s", info.ContentEncoding)
		logger.Infof("  Content-Language:     %s", info.ContentLanguage)
		logger.Infof("  Expires:              %s", info.Expires)
		logger.Infof("  x-cos-storage-class:  %s", info.StorageClass)
		logger.Infof("  x-cos-version-id:     %s", info.VersionId)
		logger.Infof("  x-cos-object-type:    %s", info.ObjectType)
		logger.Infof("  x-cos-hash-crc64ecma: %s", info.CRC64)
		for k, v := range info.CustomMeta {
			logger.Infof("  %s: %s", k, v)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statCmd)
	statCmd.Flags().String("version-id", "", "Specify the version ID of the object to query")
}
