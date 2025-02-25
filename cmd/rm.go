package cmd

import (
	"coscli/util"
	"fmt"

	"github.com/spf13/cobra"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		recursive, _ := cmd.Flags().GetBool("recursive")
		force, _ := cmd.Flags().GetBool("force")
		onlyCurrentDir, _ := cmd.Flags().GetBool("only-current-dir")
		retryNum, _ := cmd.Flags().GetInt("retry-num")
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		failOutput, _ := cmd.Flags().GetBool("fail-output")
		failOutputPath, _ := cmd.Flags().GetString("fail-output-path")
		allVersions, _ := cmd.Flags().GetBool("all-versions")
		versionId, _ := cmd.Flags().GetString("version-id")

		_, filters := util.GetFilter(include, exclude)

		if versionId != "" && recursive {
			return fmt.Errorf("version-id can only be used to delete a single version of an object")
		}

		if allVersions && !recursive {
			return fmt.Errorf("all-versions can not be used to delete single object")
		}

		fo := &util.FileOperations{
			Operation: util.Operation{
				Recursive:      recursive,
				Filters:        filters,
				OnlyCurrentDir: onlyCurrentDir,
				Force:          force,
				RetryNum:       retryNum,
				FailOutput:     failOutput,
				FailOutputPath: failOutputPath,
				AllVersions:    allVersions,
				VersionId:      versionId,
			},
			Monitor:   &util.FileProcessMonitor{},
			Config:    &config,
			Param:     &param,
			ErrOutput: &util.ErrOutput{},
			Command:   util.CommandRm,
		}
		var err error
		if recursive {
			err = util.RemoveObjects(args, fo)
		} else {
			err = util.RemoveObject(args, fo)
		}
		return err
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
	rmCmd.Flags().Bool("fail-output", true, "This option determines whether error output for failed file deletions is enabled. If enabled, any error messages for failed file deletions will be recorded in a file within the specified directory (if not specified, the default directory is coscli_output). If disabled, only the number of error files will be output to the console.")
	rmCmd.Flags().String("fail-output-path", "coscli_output", "This option specifies the error output folder where error messages for failed file deletions will be recorded. By providing a custom folder path, you can control the location and name of the error output folder. If this option is not set, the default error log folder (coscli_output) will be used.")
	rmCmd.Flags().BoolP("all-versions", "", false, "remove all versions of objects, only available if bucket versioning is enabled.")
	rmCmd.Flags().String("version-id", "", "remove Downloading a specified version of a object, only available if bucket versioning is enabled.")
}
