package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Prints information from a specified configuration file",
	Long:  `Prints information from a specified configuration file

Format:
  ./coscli config show [-c <config-file-path>]

Example:
  ./coscli config show`,
	Run: func(cmd *cobra.Command, args []string) {
		showConfig()
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
}

func showConfig() {
	fmt.Println("Configuration file path:")
	fmt.Printf("  %s\n", viper.ConfigFileUsed())
	fmt.Println("====================")
	fmt.Println("Basic Configuration Information:")
	fmt.Printf("  Secret ID:     %s\n", config.Base.SecretID)
	fmt.Printf("  Secret Key:    %s\n", config.Base.SecretKey)
	fmt.Printf("  Session Token: %s\n", config.Base.SessionToken)
	fmt.Println("====================")
	fmt.Println("Bucket Configuration Information:")

	for i, b := range config.Buckets {
		fmt.Printf("- Bucket %d :\n", i+1)
		fmt.Printf("  Name:  \t%s\n", b.Name)
		fmt.Printf("  Region:\t%s\n", b.Region)
		fmt.Printf("  Alias: \t%s\n", b.Alias)
	}
}
