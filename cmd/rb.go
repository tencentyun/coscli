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
		failOutput, _ := cmd.Flags().GetBool("fail-output")
		failOutputPath, _ := cmd.Flags().GetString("fail-output-path")
		if param.Endpoint == "" && flagRegion != "" {
			param.Endpoint = fmt.Sprintf("cos.%s.myqcloud.com", flagRegion)
		}
		var choice string
		var err error

		c, err := util.NewClient(&config, &param, bucketIDName)
		if err != nil {
			return err
		}

		if Force {
			logger.Infof("Do you want to clear all inside the bucket and delete bucket %s ? (y/n)", bucketIDName)
			_, _ = fmt.Scanf("%s\n", &choice)
			if choice == "" || choice == "y" || choice == "Y" || choice == "yes" || choice == "Yes" || choice == "YES" {
				fo := &util.FileOperations{
					Operation: util.Operation{
						Force:          true,
						FailOutput:     failOutput,
						FailOutputPath: failOutputPath,
					},
					Config:    &config,
					Param:     &param,
					ErrOutput: &util.ErrOutput{},
				}

				// 根据s.Header判断是否是融合桶或者普通桶
				s, err := c.Bucket.Head(context.Background())
				if err != nil {
					return err
				}

				if s.Header.Get("X-Cos-Bucket-Arch") == "OFS" {
					fo.Operation.AllVersions = true
				} else {
					// 判桶断是否开启版本控制，开启后需清理历史版本
					res, _, err := util.GetBucketVersioning(c)
					if err != nil {
						return err
					}
					if res.Status == util.VersionStatusEnabled {
						fo.Operation.AllVersions = true
					}
				}

				err = util.RemoveObjects(args, fo)
				if err != nil {
					return err
				}

				err = util.AbortUploads(args, fo)
				if err != nil {
					return err
				}

				err = util.RemoveBucket(bucketIDName, c)
				if err != nil {
					return err
				}
			}
		} else {
			logger.Infof("Do you want to delete %s? (y/n)", bucketIDName)
			_, _ = fmt.Scanf("%s\n", &choice)
			if choice == "" || choice == "y" || choice == "Y" || choice == "yes" || choice == "Yes" || choice == "YES" {
				err = util.RemoveBucket(bucketIDName, c)
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
	rbCmd.Flags().Bool("fail-output", true, "This option determines whether the error output for failed file uploads or downloads is enabled. If enabled, the error messages for any failed file transfers will be recorded in a file within the specified directory (if not specified, the default is coscli_output). If disabled, only the number of error files will be output to the console.")
	rbCmd.Flags().String("fail-output-path", "coscli_output", "This option specifies the designated error output folder where the error messages for failed file uploads or downloads will be recorded. By providing a custom folder path, you can control the location and name of the error output folder. If this option is not set, the default error log folder (coscli_output) will be used.")
}
