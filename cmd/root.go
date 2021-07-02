package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile         string
	eventsFile      string
	useRebootParser bool

	rootCmd = &cobra.Command{
		Use:   "cmd",
		Short: "cmd gets, parses, and validates a list of events",
	}
)

func execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initializeConfig)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./interpreter.yaml)")
	rootCmd.PersistentFlags().StringVarP(&eventsFile, "events", "e", "", "json file containing list of events")
	rootCmd.PersistentFlags().BoolVarP(&useRebootParser, "reboot", "r", false, "whether to parse just reboot events or parse cycles based on boot-time")
}

func initializeConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("interpreter")
	}

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
