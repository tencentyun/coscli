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
		allVersions, _ := cmd.Flags().GetBool("all-versions")
		_, filters := util.GetFilter(include, exclude)

		cosPath := args[0]
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

		// 判断存储桶是否开启版本控制
		if allVersions {
			res, err := util.GetBucketVersioning(c)
			if err != nil {
				return err
			}
			if res.Status != util.VersionStatusEnabled {
				return fmt.Errorf("versioning is not enabled on the current bucket")
			}
		}

		err = util.DuObjects(c, cosUrl, filters, util.DU_TYPE_CATEGORIZATION, allVersions)
		return err
	},
}

func init() {
	rootCmd.AddCommand(duCmd)
	duCmd.Flags().String("include", "", "List files that meet the specified criteria")
	duCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
	duCmd.Flags().BoolP("all-versions", "", false, "List all versions of objects, only available if bucket versioning is enabled.")
}
