package cmd

import (
	"coscli/util"
	"fmt"
	"github.com/spf13/cobra"
)

var lsduCmd = &cobra.Command{
	Use:   "lsdu",
	Short: "Displays the size of a bucket or objects",
	Long: `Displays the size of a bucket or objects

Format:
  ./coscli lsdu cos://<bucket_alias>[/prefix/] [flags]

Example:
  ./coscli lsdu cos://examplebucket/test/`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		_, filters := util.GetFilter(include, exclude)

		cosPath := ""
		if len(args) != 0 {
			cosPath = args[0]
		}
		cosUrl, err := util.FormatUrl(cosPath)
		if err != nil {
			return fmt.Errorf("cos url format error:%v", err)
		}

		bucketName := cosUrl.(*util.CosUrl).Bucket
		c, err := util.NewClient(&config, &param, bucketName)
		if err != nil {
			return err
		}

		err = util.LsAndDuObjects(c, cosUrl, filters)
		return err
	},
}

func init() {
	rootCmd.AddCommand(lsduCmd)
	lsduCmd.Flags().String("include", "", "List files that meet the specified criteria")
	lsduCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
}
