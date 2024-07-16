package cmd

import (
	"context"
	"coscli/util"
	"fmt"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		bucketIDName, _ := util.ParsePath(args[0])
		flagRegion, _ := cmd.Flags().GetString("region")
		Force, _ := cmd.Flags().GetBool("force")
		if param.Endpoint == "" && flagRegion != "" {
			param.Endpoint = fmt.Sprintf("cos.%s.myqcloud.com", flagRegion)
		}
		var choice string
		var err error
		if Force {
			logger.Infof("Do you want to clear all inside the bucket and delete bucket %s ? (y/n)", bucketIDName)
			_, _ = fmt.Scanf("%s\n", &choice)
			if choice == "" || choice == "y" || choice == "Y" || choice == "yes" || choice == "Yes" || choice == "YES" {
				fo := &util.FileOperations{
					Operation: util.Operation{
						Force: true,
					},
					Monitor:   &util.FileProcessMonitor{},
					Config:    &config,
					Param:     &param,
					ErrOutput: &util.ErrOutput{},
				}
				err = util.RemoveObjects(args, fo)
				if err != nil {
					return err
				}

				err = abortParts(args[0], "", "")
				if err != nil {
					return err
				}
				err = removeBucket(bucketIDName)
				if err != nil {
					return err
				}
			}
		} else {
			logger.Infof("Do you want to delete %s? (y/n)", bucketIDName)
			_, _ = fmt.Scanf("%s\n", &choice)
			if choice == "" || choice == "y" || choice == "Y" || choice == "yes" || choice == "Yes" || choice == "YES" {
				err = removeBucket(bucketIDName)
				if err != nil {
					return err
				}
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rbCmd)
	rbCmd.Flags().BoolP("force", "f", false, "Clear all inside the bucket and delete bucket")
	rbCmd.Flags().StringP("region", "r", "", "Region")
}

func removeBucket(bucketIDName string) error {
	c, err := util.NewClient(&config, &param, bucketIDName)
	if err != nil {
		return err
	}
	_, err = c.Bucket.Delete(context.Background())
	if err != nil {
		return err
	}
	logger.Infof("Delete a empty bucket! name: %s\n", bucketIDName)
	return nil
}
