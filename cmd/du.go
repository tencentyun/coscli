package cmd

import (
	"coscli/util"
	"fmt"
	"github.com/spf13/cobra"
)

var duCmd = &cobra.Command{
	Use:   "du",
	Short: "Displays the size of a bucket or objects",
	Long: `Displays the size of a bucket or objects

Format:
  ./coscli du cos://<bucket_alias>[/prefix/] [flags]

Example:
  ./coscli du cos://examplebucket/test/`,
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

		err = util.DuObjects(c, cosUrl, filters)
		return err
	},
}

func init() {
	rootCmd.AddCommand(duCmd)
	duCmd.Flags().String("include", "", "List files that meet the specified criteria")
	duCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
}
