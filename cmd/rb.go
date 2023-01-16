package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"os"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rbCmd = &cobra.Command{
	Use:   "rb",
	Short: "Remove bucket",
	Long: `Remove bucket

Format:
  ./coscli rb cos://<bucket-name>-<app-id> -e <endpoint>

Example:
  ./coscli rb cos://example-1234567890 -e cos.ap-beijing.myqcloud.com`,
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
		bucketIDName, _ := util.ParsePath(args[0])
		flagRegion, _ := cmd.Flags().GetString("region")
		Force, _ := cmd.Flags().GetBool("force")
		if param.Endpoint == "" && flagRegion != "" {
			param.Endpoint = fmt.Sprintf("cos.%s.myqcloud.com", flagRegion)
		}
		var choice string
		if Force {
			logger.Infof("Do you want to clear all inside the bucket and delete bucket %s ? (y/n)", bucketIDName)
			_, _ = fmt.Scanf("%s\n", &choice)
			if choice == "" || choice == "y" || choice == "Y" || choice == "yes" || choice == "Yes" || choice == "YES" {
				removeObjects1(args, "", "", true)
				abortParts(args[0], "", "")
				removeBucket(bucketIDName)
			}
		} else {
			logger.Infof("Do you want to delete %s? (y/n)", bucketIDName)
			_, _ = fmt.Scanf("%s\n", &choice)
			if choice == "" || choice == "y" || choice == "Y" || choice == "yes" || choice == "Yes" || choice == "YES" {
				removeBucket(bucketIDName)
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(rbCmd)
	rbCmd.Flags().BoolP("force", "f", false, "Clear all inside the bucket and delete bucket")
	rbCmd.Flags().StringP("region", "r", "", "Region")
}

func removeBucket(bucketIDName string) {
	c := util.NewClient(&config, &param, bucketIDName)
	_, err := c.Bucket.Delete(context.Background())
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}
	logger.Infof("Delete a empty bucket! name: %s\n", bucketIDName)
}
