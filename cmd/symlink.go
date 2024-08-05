package cmd

import (
	"coscli/util"
	"fmt"
	"strings"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var symlinkCmd = &cobra.Command{
	Use:   "symlink",
	Short: "Create/Get symlink ",
	Long: `Create/Get symlink

Format:
  ./coscli symlink --method create cos://<bucket-name>-<appid>/test1 --link linkKey

Example:
  ./coscli symlink --method create cos://examplebucket-1234567890/test1 --link linkKey`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		method, _ := cmd.Flags().GetString("method")
		linkKey, _ := cmd.Flags().GetString("link")

		linkKey = strings.ToLower(linkKey)

		cosUrl, err := util.FormatUrl(args[0])
		if err != nil {
			return err
		}
		if !cosUrl.IsCosUrl() {
			return fmt.Errorf("cospath needs to contain cos://")
		}

		// 实例化cos client
		bucketName := cosUrl.(*util.CosUrl).Bucket
		c, err := util.NewClient(&config, &param, bucketName)
		if err != nil {
			return err
		}

		if method == "create" {
			err = util.CreateSymlink(c, cosUrl, linkKey)
			if err != nil {
				return err
			}
			logger.Infof("Create symlink successfully! object: %s, symlink: %s", cosUrl.(*util.CosUrl).Object, linkKey)
		} else if method == "get" {
			res, err := util.GetSymlink(c, linkKey)
			if err != nil {
				return err
			}
			logger.Infof("Link-object: %s", res)
		} else {
			return fmt.Errorf("--method can only be selected create get and get")
		}

		return err
	},
}

func init() {
	rootCmd.AddCommand(symlinkCmd)
	symlinkCmd.Flags().StringP("method", "", "", "Create/Get")
	symlinkCmd.Flags().StringP("link", "", "", "link key")
}
