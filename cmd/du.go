package cmd

import (
	"coscli/util"

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
		bucketName, cosPath := util.ParsePath(args[0])
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		var err error
		if cosPath == "" {
			err = duBucket(bucketName, include, exclude)
		} else {
			err = duObjects(bucketName, cosPath, include, exclude)
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(duCmd)
	duCmd.Flags().String("include", "", "List files that meet the specified criteria")
	duCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
}

func duBucket(bucketName string, include string, exclude string) error {
	c, err := util.NewClient(&config, &param, bucketName)
	if err != nil {
		return err
	}

	objects, _, err := util.GetObjectsListRecursive(c, "", 0, include, exclude)
	if err != nil {
		return err
	}
	util.Statistic(objects)
	return nil
}

func duObjects(bucketName string, cosPath string, include string, exclude string) error {
	c, err := util.NewClient(&config, &param, bucketName)
	if err != nil {
		return err
	}

	objects, _, err := util.GetObjectsListRecursive(c, cosPath, 0, include, exclude)
	if err != nil {
		return err
	}
	util.Statistic(objects)
	return nil
}
