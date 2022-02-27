package cmd

import (
	"coscli/util"
	"os"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Used to delete an existing bucket",
	Long: `Used to delete an existing bucket

Format:
  ./coscli config delete -a <alias> [-c <config-file-path>]

Example:
  ./coscli config delete -a example`,
	Run: func(cmd *cobra.Command, args []string) {
		deleteBucketConfig(cmd)
	},
}

func init() {
	configCmd.AddCommand(configDeleteCmd)

	configDeleteCmd.Flags().StringP("alias", "a", "", "Bucket alias")

	_ = configDeleteCmd.MarkFlagRequired("alias")
}

func deleteBucketConfig(cmd *cobra.Command) {
	alias, _ := cmd.Flags().GetString("alias")
	b, i, err := util.FindBucket(&config, alias)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}
	config.Buckets = append(config.Buckets[:i], config.Buckets[i+1:]...)

	viper.Set("cos.buckets", config.Buckets)
	if err := viper.WriteConfigAs(viper.ConfigFileUsed()); err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}
	logger.Infof("Delete succeccfully! name: %s, endpoint: %s, alias: %s", b.Name, b.Endpoint, b.Alias)
}
