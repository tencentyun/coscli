package cmd

import (
	"context"
	"encoding/xml"
	"fmt"

	"coscli/util"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tencentyun/cos-go-sdk-v5"
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
		var err error
		if recursive {
			err = restoreObjects(args[0], days, mode, include, exclude)
		} else {
			err = restoreObject(args[0], days, mode)
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

func restoreObject(arg string, days int, mode string) error {
	bucketName, cosPath := util.ParsePath(arg)
	c, err := util.NewClient(&config, &param, bucketName)
	if err != nil {
		return err
	}

	opt := &cos.ObjectRestoreOptions{
		XMLName:       xml.Name{},
		Days:          days,
		Tier:          &cos.CASJobParameters{Tier: mode},
		XOptionHeader: nil,
	}

	logger.Infof("Restore cos://%s/%s\n", bucketName, cosPath)
	_, err = c.Object.PostRestore(context.Background(), cosPath, opt)
	if err != nil {
		logger.Errorln(err)
		return err
	}
	return nil
}

func restoreObjects(arg string, days int, mode string, include string, exclude string) error {
	bucketName, cosPath := util.ParsePath(arg)
	c, err := util.NewClient(&config, &param, bucketName)
	if err != nil {
		return err
	}

	objects, _, err := util.GetObjectsListRecursive(c, cosPath, 0, include, exclude)
	if err != nil {
		return err
	}
	succeed_num := 0
	failed_num := 0

	for _, o := range objects {
		err := restoreObject(fmt.Sprintf("cos://%s/%s", bucketName, o.Key), days, mode)
		if err != nil {
			failed_num += 1
		} else {
			succeed_num += 1
		}
	}
	return nil
}
