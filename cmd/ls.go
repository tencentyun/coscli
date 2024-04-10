package cmd

import (
	"coscli/util"

	logger "github.com/sirupsen/logrus"
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
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		recursive, _ := cmd.Flags().GetBool("recursive")
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		if limit < 0 || limit > 1000 {
			logger.Fatalln("Flag --limit should in range 0~1000")
		}

		cosPath := ""
		if len(args) != 0 {
			cosPath = args[0]
		}

		cosUrl, err := util.FormatUrl(cosPath)
		if err != nil {
			logger.Fatalf("cos url format error:%v", err)
		}

		// 无参数，则列出当前账号下的所有存储桶
		if cosPath == "" {
			// 实例化cos client
			c := util.NewClient(&config, &param, "")
			util.ListBuckets(c, limit)
		} else if cosUrl.IsCosUrl() {
			// 实例化cos client
			bucketName := cosUrl.(*util.CosUrl).Bucket
			c := util.NewClient(&config, &param, bucketName)
			_, filters := util.GetFilter(include, exclude)
			util.ListObjects(c, cosUrl, limit, recursive, filters)
		} else {
			logger.Fatalln("cospath needs to contain cos://")
		}
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)

	lsCmd.Flags().Int("limit", 0, "Limit the number of objects listed(0~1000)")
	lsCmd.Flags().BoolP("recursive", "r", false, "List objects recursively")
	lsCmd.Flags().String("include", "", "List files that meet the specified criteria")
	lsCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
}
