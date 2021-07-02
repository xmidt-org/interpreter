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

	RootCmd = &cobra.Command{
		Use:   "cmd",
		Short: "cmd gets, parses, and validates a list of events",
	}
)

func Execute() error {
	return RootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initializeConfig)
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./interpreter.yaml)")
	RootCmd.PersistentFlags().StringVarP(&eventsFile, "events", "e", "", "json file containing list of events")
	RootCmd.PersistentFlags().BoolVar(&useRebootParser, "reboot", false, "whether to use reboot parser or parse cycles based on boot-time")
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
