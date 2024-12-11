package cmd

import (
	"context"
	"coscli/util"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var abortCmd = &cobra.Command{
	Use:   "abort",
	Short: "Abort parts",
	Long: `Abort parts

Format:
  ./coscli abort cos://<bucket-name>[/<prefix>] [flags]

Example:
  ./coscli abort cos://examplebucket/test/`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		failOutput, _ := cmd.Flags().GetBool("fail-output")
		failOutputPath, _ := cmd.Flags().GetString("fail-output-path")

		_, filters := util.GetFilter(include, exclude)

		fo := &util.FileOperations{
			Operation: util.Operation{
				FailOutput:     failOutput,
				FailOutputPath: failOutputPath,
				Filters:        filters,
			},
			Config:    &config,
			Param:     &param,
			ErrOutput: &util.ErrOutput{},
		}

		err := util.AbortUploads(args, fo)
		return err
	},
}

func init() {
	rootCmd.AddCommand(abortCmd)

	abortCmd.Flags().String("include", "", "List files that meet the specified criteria")
	abortCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
	abortCmd.Flags().Bool("fail-output", true, "This option determines whether the error output for failed file uploads or downloads is enabled. If enabled, the error messages for any failed file transfers will be recorded in a file within the specified directory (if not specified, the default is coscli_output). If disabled, only the number of error files will be output to the console.")
	abortCmd.Flags().String("fail-output-path", "coscli_output", "This option specifies the designated error output folder where the error messages for failed file uploads or downloads will be recorded. By providing a custom folder path, you can control the location and name of the error output folder. If this option is not set, the default error log folder (coscli_output) will be used.")
}

func abortParts(arg string, include string, exclude string) error {
	bucketName, cosPath := util.ParsePath(arg)
	c, err := util.NewClient(&config, &param, bucketName)
	if err != nil {
		return err
	}
	uploads, err := util.GetUploadsListRecursive(c, cosPath, 0, include, exclude)
	if err != nil {
		return err
	}

	successCnt, failCnt := 0, 0
	for _, u := range uploads {
		_, err := c.Object.AbortMultipartUpload(context.Background(), u.Key, u.UploadID)
		if err != nil {
			logger.Infoln("Abort fail!    UploadID:", u.UploadID, "Key:", u.Key)
			failCnt++
		} else {
			logger.Infoln("Abort success! UploadID:", u.UploadID, "Key:", u.Key)
			successCnt++
		}
	}
	logger.Infoln("Total:", len(uploads), ",", successCnt, "Success,", failCnt, "Fail")
	return nil
}
