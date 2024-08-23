package cmd

import (
	"coscli/util"
	"fmt"

	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore objects",
	Long: `Restore objects

Format:
  ./coscli restore cos://<bucket-name>[/<prefix>] [flags]

Example:
  ./coscli restore cos://examplebucket/test/ -r -d 3 -m Expedited`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		recursive, _ := cmd.Flags().GetBool("recursive")
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		days, _ := cmd.Flags().GetInt("days")
		mode, _ := cmd.Flags().GetString("mode")

		_, filters := util.GetFilter(include, exclude)

		cosPath := ""
		if len(args) != 0 {
			cosPath = args[0]
		}
		cosUrl, err := util.FormatUrl(cosPath)
		if err != nil {
			return fmt.Errorf("cos url format error:%v", err)
		}
		if !cosUrl.IsCosUrl() {
			return fmt.Errorf("cospath needs to contain %s", util.SchemePrefix)
		}

		bucketName := cosUrl.(*util.CosUrl).Bucket
		c, err := util.NewClient(&config, &param, bucketName)
		if err != nil {
			return err
		}

		if recursive {
			err = util.RestoreObjects(c, cosUrl, days, mode, filters)
		} else {
			err = util.RestoreObject(c, bucketName, cosUrl.(*util.CosUrl).Object, days, mode)
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().BoolP("recursive", "r", false, "Restore objects recursively")
	restoreCmd.Flags().String("include", "", "Include files that meet the specified criteria")
	restoreCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
	restoreCmd.Flags().IntP("days", "d", 3, "Specifies the expiration time of temporary files")
	restoreCmd.Flags().StringP("mode", "m", "Standard", "Specifies the mode for fetching temporary files")
}
