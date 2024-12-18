package cmd

import (
	"context"
	"coscli/util"
	"fmt"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List buckets or objects",
	Long: `List buckets or objects

Format:
  ./coscli ls cos://<bucket-name>[/prefix/] [flags]

Example:
  ./coscli ls cos://examplebucket/test/ -r`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		recursive, _ := cmd.Flags().GetBool("recursive")
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		allVersions, _ := cmd.Flags().GetBool("all-versions")

		if limit == 0 {
			limit = 10000
		} else if limit < 0 {
			return fmt.Errorf("Flag --limit should be greater than 0")
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
			// 实例化cos client
			c, err := util.NewClient(&config, &param, "")
			if err != nil {
				return err
			}
			err = util.ListBuckets(c, limit)
		} else if cosUrl.IsCosUrl() {
			// 实例化cos client
			bucketName := cosUrl.(*util.CosUrl).Bucket
			c, err := util.NewClient(&config, &param, bucketName)
			if err != nil {
				return err
			}
			// 判断存储桶是否开启版本控制
			if allVersions {
				res, _, err := util.GetBucketVersioning(c)
				if err != nil {
					return err
				}
				if res.Status != util.VersionStatusEnabled {
					return fmt.Errorf("versioning is not enabled on the current bucket")
				}
			}

			_, filters := util.GetFilter(include, exclude)
			// 根据s.Header判断是否是融合桶或者普通桶
			s, err := c.Bucket.Head(context.Background())
			if err != nil {
				return err
			}
			if s.Header.Get("X-Cos-Bucket-Arch") == "OFS" {
				if allVersions {
					return fmt.Errorf("the OFS bucket does not support listing multiple versions")
				} else {
					err = util.ListOfsObjects(c, cosUrl, limit, recursive, filters)
				}
			} else {
				if allVersions {
					err = util.ListObjectVersions(c, cosUrl, limit, recursive, filters)
				} else {
					err = util.ListObjects(c, cosUrl, limit, recursive, filters)
				}

			}

			if err != nil {
				return err
			}

		} else {
			return fmt.Errorf("cospath needs to contain cos://")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)

	lsCmd.Flags().Int("limit", 0, "Limit the number of objects listed(0~1000)")
	lsCmd.Flags().BoolP("recursive", "r", false, "List objects recursively")
	lsCmd.Flags().String("include", "", "List files that meet the specified criteria")
	lsCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
	lsCmd.Flags().BoolP("all-versions", "", false, "List all versions of objects, only available if bucket versioning is enabled.")
}
