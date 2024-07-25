package cmd

import (
	"coscli/util"
	"fmt"
	"github.com/spf13/cobra"
)

var catCmd = &cobra.Command{
	Use:   "cat",
	Short: "Cat object info",
	Long: `Cat object info

Format:
  ./coscli cat cos://<bucket-name>-<appid>/<object>

Example:
  ./coscli cat cos://examplebucket-1234567890/test.txt`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cosUrl, err := util.FormatUrl(args[0])
		if !cosUrl.IsCosUrl() {
			return fmt.Errorf("cospath needs to contain cos://")
		}

		// 实例化cos client
		bucketName := cosUrl.(*util.CosUrl).Bucket
		c, err := util.NewClient(&config, &param, bucketName)

		err = util.CatObject(c, cosUrl)
		return err
	},
}

func init() {
	rootCmd.AddCommand(catCmd)
}
