package cmd

import (
	"coscli/util"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var configAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Used to add a new bucket configuration",
	Long:  `Used to add a new bucket configuration

Format:
  ./coscli config add -b <bucket-name> -r <region> -a <alias> [-c <config-file-path>]

Example:
  ./coscli config add -b example-1234567890 -r ap-shanghai -a example`,
	Run: func(cmd *cobra.Command, args []string) {
		addBucketConfig(cmd)
	},
}

func init() {
	configCmd.AddCommand(configAddCmd)

	configAddCmd.Flags().StringP("bucket", "b", "", "Bucket name")
	configAddCmd.Flags().StringP("region", "r", "", "Bucket region")
	configAddCmd.Flags().StringP("alias", "a", "", "Bucket alias")

	_ = configAddCmd.MarkFlagRequired("bucket")
	_ = configAddCmd.MarkFlagRequired("region")
}

func addBucketConfig(cmd *cobra.Command) {
	name, _ := cmd.Flags().GetString("bucket")
	region, _ := cmd.Flags().GetString("region")
	alias, _ := cmd.Flags().GetString("alias")
	if alias == "" {
		alias = name
	}

	bucket := util.Bucket{
		Name:   name,
		Region: region,
		Alias:  alias,
	}

	for _, b := range config.Buckets {
		if name == b.Name {
			_, _ = fmt.Fprintln(os.Stderr, "The bucket already exists, fail to add!")
			os.Exit(1)
		} else if alias == b.Name {
			_, _ = fmt.Fprintln(os.Stderr, "The alias cannot be the same as other bucket's name")
			os.Exit(1)
		} else if alias == b.Alias {
			_, _ = fmt.Fprintln(os.Stderr, "The alias already exists, fail to add!")
			os.Exit(1)
		}
	}

	config.Buckets = append(config.Buckets, bucket)
	viper.Set("cos.buckets", config.Buckets)

	if err := viper.WriteConfigAs(viper.ConfigFileUsed()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Add successfully! name: %s, region: %s, alias: %s\n", name, region, alias)
}
