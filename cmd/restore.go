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
		failOutput, _ := cmd.Flags().GetBool("fail-output")
		failOutputPath, _ := cmd.Flags().GetString("fail-output-path")
		routines, _ := cmd.Flags().GetInt("routines")

		if days < 1 || days > 365 {
			return fmt.Errorf("Flag --days should in range 1~365")
		}
		if routines < 1 || routines > 10000 {
			return fmt.Errorf("Flag --routines should in range 1~10000")
		}

		_, filters := util.GetFilter(include, exclude)

		fo := &util.FileOperations{
			Operation: util.Operation{
				Recursive:      recursive,
				Filters:        filters,
				FailOutput:     failOutput,
				FailOutputPath: failOutputPath,
				Days:           days,
				RestoreMode:    mode,
				Routines:       routines,
			},
			Config:    &config,
			Param:     &param,
			ErrOutput: &util.ErrOutput{},
			Command:   util.CommandRestore,
		}

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
			var bucketType string
			// 获取桶类型
			bucketType, err = util.GetBucketType(c, fo.Param, fo.Config, bucketName)
			if err != nil {
				return err
			}
			err = util.RestoreObjects(c, cosUrl, fo, bucketType)
		} else {
			object := cosUrl.(*util.CosUrl).Object
			resp, restoreErr := util.TryRestoreObject(c, bucketName, object, days, mode)
			if restoreErr != nil && !(resp != nil && resp.StatusCode == 409) {
				err = restoreErr
			} else {
				// 409 表示对象已在回热中，视为提交成功
				fmt.Printf("Restore cos://%s/%s submitted.\n", bucketName, object)
			}
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
	restoreCmd.Flags().Bool("fail-output", true, "This option determines whether error output for failed file restore is enabled. If enabled, any error messages for failed file reheats will be recorded in a file within the specified directory (if not specified, the default directory is coscli_output). If disabled, only the number of error files will be output to the console.")
	restoreCmd.Flags().String("fail-output-path", "coscli_output", "This option specifies the error output folder where error messages for file restore failures will be recorded. By providing a custom folder path, you can control the location and name of the error output folder. If this option is not set, the default error log folder (coscli_output) will be used.")
	restoreCmd.Flags().Int("routines", 3, "Specifies the number of concurrent restore tasks, range 1~10000")
}
