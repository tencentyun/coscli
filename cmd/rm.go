package cmd

import (
	"coscli/util"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tencentyun/cos-go-sdk-v5"
)

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove objects",
	Long: `Remove objects

Format:
  ./coscli rm cos://<bucket-name>[/prefix/] [cos://<bucket-name>[/prefix/]...] [flags]

Example:
  ./coscli rm cos://example/test/ -r`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
			return err
		}
		for _, arg := range args {
			bucketName, _ := util.ParsePath(arg)
			if bucketName == "" {
				return fmt.Errorf("Invalid arguments! ")
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		recursive, _ := cmd.Flags().GetBool("recursive")
		force, _ := cmd.Flags().GetBool("force")
		onlyCurrentDir, _ := cmd.Flags().GetBool("only-current-dir")
		retryNum, _ := cmd.Flags().GetInt("retry-num")
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		failOutput, _ := cmd.Flags().GetBool("fail-output")
		failOutputPath, _ := cmd.Flags().GetString("fail-output-path")

		_, filters := util.GetFilter(include, exclude)

		fo := &util.FileOperations{
			Operation: util.Operation{
				Recursive:      recursive,
				Filters:        filters,
				OnlyCurrentDir: onlyCurrentDir,
				Force:          force,
				RetryNum:       retryNum,
				FailOutput:     failOutput,
				FailOutputPath: failOutputPath,
			},
			Monitor:   &util.FileProcessMonitor{},
			Config:    &config,
			Param:     &param,
			ErrOutput: &util.ErrOutput{},
		}

		if recursive {
			util.RemoveObjects(args, fo)
		} else {
			util.RemoveObject(args, fo)
		}
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)

	rmCmd.Flags().BoolP("recursive", "r", false, "Delete object recursively")
	rmCmd.Flags().BoolP("force", "f", false, "Force delete")
	rmCmd.Flags().Bool("only-current-dir", false, "Upload only the files in the current directory, ignoring subdirectories and their contents")
	rmCmd.Flags().Int("retry-num", 0, "Rate-limited retry. Specify 1-10 times. When multiple machines concurrently execute download operations on the same COS directory, rate-limited retry can be performed by specifying this parameter.")
	rmCmd.Flags().String("include", "", "List files that meet the specified criteria")
	rmCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
	rmCmd.Flags().Bool("fail-output", false, "This option determines whether the error output for failed file uploads or downloads is enabled. If enabled, the error messages for any failed file transfers will be recorded in a file within the specified directory (if not specified, the default is coscli_output). If disabled, only the number of error files will be output to the console.")
	rmCmd.Flags().String("fail-output-path", "coscli_output", "This option specifies the designated error output folder where the error messages for failed file uploads or downloads will be recorded. By providing a custom folder path, you can control the location and name of the error output folder. If this option is not set, the default error log folder (coscli_output) will be used.")
}

// 获取所有文件和目录
func getFilesAndDirs(c *cos.Client, cosDir string, nextMarker string, include string, exclude string) (files []string) {
	objects, _, _, commonPrefixes := util.GetObjectsListIterator(c, cosDir, nextMarker, include, exclude)
	tempFiles := make([]string, 0)
	tempFiles = append(tempFiles, cosDir)
	for _, v := range objects {
		files = append(files, v.Key)
	}
	if len(commonPrefixes) > 0 {
		for _, v := range commonPrefixes {
			files = append(files, getFilesAndDirs(c, v, nextMarker, include, exclude)...)
		}
	}
	files = append(files, tempFiles...)
	return files
}
