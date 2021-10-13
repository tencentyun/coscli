package cmd

import (
	"context"
	"coscli/util"
	"encoding/xml"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/tencentyun/cos-go-sdk-v5"
	"os"
)

var rmCmd = &cobra.Command{
	Use: "rm",
	Short: "Remove objects",
	Long:  `Remove objects

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
			if bucketName == ""{
				return fmt.Errorf("Invalid arguments! ")
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		recursive, _ := cmd.Flags().GetBool("recursive")
		force, _ := cmd.Flags().GetBool("force")
		include, _ := cmd.Flags().GetString("include")
		exclude, _ := cmd.Flags().GetString("exclude")
		if recursive {
			removeObjects(args, include, exclude, force)
		} else {
			removeObject(args, force)
		}
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)

	rmCmd.Flags().BoolP("recursive", "r", false, "Delete object recursively")
	rmCmd.Flags().BoolP("force", "f", false, "Force delete")
	rmCmd.Flags().String("include", "", "List files that meet the specified criteria")
	rmCmd.Flags().String("exclude", "", "Exclude files that meet the specified criteria")
}

func removeObjects(args []string, include string, exclude string, force bool) {
	for _, arg := range args {
		bucketName, cosDir := util.ParsePath(arg)
		c := util.NewClient(&config, bucketName)

		if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
			cosDir += "/"
		}

		objects := util.GetObjectsListRecursive(c, cosDir, 0, include, exclude)
		if len(objects) == 0 {
			fmt.Println("No objects were deleted!")
			return
		}

		var oKeys []cos.Object
		for _, o := range objects {
			if !force {
				fmt.Printf("Do you want to delete %s? (y/n)", o.Key)
				var choice string
				_, _ = fmt.Scanf("%s\n", &choice)
				if choice == "" || choice == "y" || choice == "Y" || choice == "yes" || choice == "Yes" || choice == "YES" {
					oKeys = append(oKeys, cos.Object{Key: o.Key})
				}
			} else {
				oKeys = append(oKeys, cos.Object{Key: o.Key})
			}
		}
		opt := &cos.ObjectDeleteMultiOptions{
			XMLName: xml.Name{},
			Quiet:   false,
			Objects: oKeys,
		}

		res, _, err := c.Object.DeleteMulti(context.Background(), opt)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		for _, o := range res.DeletedObjects {
			fmt.Println("Delete ", o.Key)
		}
		if len(res.Errors) == 0 {
			fmt.Printf("\nAll deleted successfully!\n")
		} else {
			fmt.Println()
			for i, e := range res.Errors {
				fmt.Println(i+1, ". Fail to delete", e.Key)
				fmt.Println("    Error Code: ", e.Code, " Message: ", e.Message)
			}
		}
	}
}

func removeObject(args []string, force bool) {
	for _, arg := range args {
		bucketName, cosPath := util.ParsePath(arg)
		c := util.NewClient(&config, bucketName)

		opt := &cos.ObjectDeleteOptions{
			XCosSSECustomerAglo:   "",
			XCosSSECustomerKey:    "",
			XCosSSECustomerKeyMD5: "",
			XOptionHeader:         nil,
			VersionId:             "",
		}

		if !force {
			fmt.Printf("Do you want to delete %s? (y/n)", cosPath)
			var choice string
			_, _ = fmt.Scanf("%s\n", &choice)
			if choice == "" || choice == "y" || choice == "Y" || choice == "yes" || choice == "Yes" || choice == "YES" {
				_, err := c.Object.Delete(context.Background(), cosPath, opt)
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				fmt.Println("Delete", arg, "successfully!")
			}
		} else {
			_, err := c.Object.Delete(context.Background(), cosPath, opt)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("Delete", arg, "successfully!")
		}
	}
}
