package cmd

import (
	"coscli/util"
	"fmt"
	"github.com/spf13/cobra"
)

var lspartsCmd = &cobra.Command{
	Use:   "lsparts",
	Short: "List multipart uploads",
	Long: `List multipart uploads

Format:
  ./coscli lsparts cos://<bucket-name>[/<prefix>] [flags]

Example:
  ./coscli lsparts cos://examplebucket/test/`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		uploadId, _ := cmd.Flags().GetString("uploadid")
		if limit == 0 {
			limit = 10000
		} else if limit < 0 {
			return fmt.Errorf("Flag --limit should be greater than 0")
		}

		cosUrl, err := util.FormatUrl(args[0])
		if err != nil {
			return fmt.Errorf("cos url format error:%v", err)
		}

		if !cosUrl.IsCosUrl() {
			return fmt.Errorf("cospath needs to contain cos://")
		}

		_, filters := util.GetFilter(include, exclude)

		bucketName := cosUrl.(*util.CosUrl).Bucket

		c, err := util.NewClient(&config, &param, bucketName)
		if err != nil {
			return err
		}

		if uploadId != "" {
			err = util.ListParts(c, cosUrl, limit, uploadId)
		} else {
			err = util.ListUploads(c, cosUrl, limit, filters)
		}

		return err
	},
}

func init() {
	rootCmd.AddCommand(lspartsCmd)

	lspartsCmd.Flags().Int("limit", 0, "Limit the number of parts listed(0~1000)")
	lspartsCmd.Flags().String("include", "", "List files that meet the specified criteria")
	lspartsCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
	lspartsCmd.Flags().String("uploadid", "", "Identify the ID of this multipart upload, which is obtained when initializing the multipart upload using the Initiate Multipart Upload interface.")
}
