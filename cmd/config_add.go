package cmd

import (
	"coscli/util"
	"os"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Used to add a new bucket configuration",
	Long: `Used to add a new bucket configuration

Format:
  ./coscli config add -b <bucket-name> -e <endpoint> -a <alias> [-c <config-file-path>]

Example:
  ./coscli config add -b example-1234567890 -r ap-shanghai -a example`,
	Run: func(cmd *cobra.Command, args []string) {
		addBucketConfig(cmd)
	},
}

func init() {
	configCmd.AddCommand(configAddCmd)

	configAddCmd.Flags().StringP("bucket", "b", "", "Bucket name")
	configAddCmd.Flags().StringP("endpoint", "e", "", "Bucket endpoint")
	configAddCmd.Flags().StringP("region", "r", "", "Bucket region")
	configAddCmd.Flags().StringP("alias", "a", "", "Bucket alias")

	_ = configAddCmd.MarkFlagRequired("bucket")
	// _ = configAddCmd.MarkFlagRequired("endpoint")
}

func addBucketConfig(cmd *cobra.Command) {
	name, _ := cmd.Flags().GetString("bucket")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	region, _ := cmd.Flags().GetString("region")
	alias, _ := cmd.Flags().GetString("alias")
	if alias == "" {
		alias = name
	}

	bucket := util.Bucket{
		Name:     name,
		Endpoint: endpoint,
		Region:   region,
		Alias:    alias,
	}

	for _, b := range config.Buckets {
		if name == b.Name {
			logger.Fatalln("The bucket already exists, fail to add!")
			os.Exit(1)
		} else if alias == b.Name {
			logger.Fatalln("The alias cannot be the same as other bucket's name")
			os.Exit(1)
		} else if alias == b.Alias {
			logger.Fatalln("The alias already exists, fail to add!")
			os.Exit(1)
		}
	}

	config.Buckets = append(config.Buckets, bucket)
	viper.Set("cos.buckets", config.Buckets)

	if err := viper.WriteConfigAs(viper.ConfigFileUsed()); err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}
	logger.Infof("Add successfully! name: %s, endpoint: %s, alias: %s\n", name, endpoint, alias)
}
