package cmd

import (
	"coscli/util"
	"fmt"

	"github.com/spf13/cobra"
)

var rovCmd = &cobra.Command{
	Use:   "rov",
	Short: "recovery object version",
	Long:  `recovery object version`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		notDryrun, _ := cmd.Flags().GetBool("not-dryrun")
		previous, _ := cmd.Flags().GetInt("previous")
		anyVersion, _ := cmd.Flags().GetBool("any-version")

		if limit == 0 {
			limit = 10000
		} else if limit < 0 {
			return fmt.Errorf("flag --limit should be greater than 0")
		}

		cosPath := ""
		if len(args) != 0 {
			cosPath = args[0]
		}

		cosUrl, err := util.FormatUrl(cosPath)
		if err != nil {
			return fmt.Errorf("cos url format error:%v", err)
		}

		// 无参数，则列出当前账号下的所有存储桶
		if cosPath == "" {
			return fmt.Errorf("bucket name is required")
		} else if cosUrl.IsCosUrl() {
			// 实例化cos client
			bucketName := cosUrl.(*util.CosUrl).Bucket
			c, err := util.NewClient(&config, &param, bucketName)
			if err != nil {
				return err
			}
			return util.RecoveryObjectVersion(c, cosUrl.(*util.CosUrl), previous, limit, !notDryrun, anyVersion)
		} else {
			return fmt.Errorf("cospath needs to contain cos://")
		}

	},
}

func init() {
	rootCmd.AddCommand(rovCmd)

	rovCmd.Flags().IntP("previous", "P", 1, "previous version to recovery. 1 means the latest version, 2 means the second latest version")
	rovCmd.Flags().IntP("limit", "l", 10000, "limit the number of objects to list")
	rovCmd.Flags().BoolP("not-dryrun", "n", false, "default dryrun mode, do not actually recovery object version")
	rovCmd.Flags().BoolP("any-version", "a", false, "if set, recovery any version of the object, else only recovery the latest delete marker")
}
