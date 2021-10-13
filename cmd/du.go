package cmd

import (
	"coscli/util"
	"github.com/spf13/cobra"
)

var duCmd = &cobra.Command{
	Use:   "du",
	Short: "Displays the size of a bucket or objects",
	Long:  `Displays the size of a bucket or objects

Format:
  ./coscli du cos://<bucket_alias>[/prefix/] [flags]

Example:
  ./coscli du cos://examplebucket/test/`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		bucketName, cosPath := util.ParsePath(args[0])
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		if cosPath == "" {
			duBucket(bucketName, include, exclude)
		} else {
			duObjects(bucketName, cosPath, include, exclude)
		}
	},
}

func init() {
	rootCmd.AddCommand(duCmd)
	duCmd.Flags().String("include", "", "List files that meet the specified criteria")
	duCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
}

func duBucket(bucketName string, include string, exclude string) {
	c := util.NewClient(&config, bucketName)

	objects := util.GetObjectsListRecursive(c, "", 0, include, exclude)

	util.Statistic(objects)
}

func duObjects(bucketName string, cosPath string, include string, exclude string) {
	c := util.NewClient(&config, bucketName)

	objects := util.GetObjectsListRecursive(c, cosPath, 0, include, exclude)

	util.Statistic(objects)
}
