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

		err := abortParts(args[0], include, exclude)
		return err
	},
}

func init() {
	rootCmd.AddCommand(abortCmd)

	abortCmd.Flags().String("include", "", "List files that meet the specified criteria")
	abortCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
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
