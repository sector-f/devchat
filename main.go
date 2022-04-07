package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "devchat",
		Short: "Hugo is a very fast static site generator",
		RunE: func(cmd *cobra.Command, args []string) error {
			configFilename, _ := cmd.PersistentFlags().GetString("config")

			var (
				conf config
				err  error
			)

			if configFilename == "" {
				conf = defaultConfig()
			} else {
				conf, err = configFromFile(configFilename)
				if err != nil {
					return err
				}
			}

			s, err := newServer(conf)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			c := make(chan os.Signal, 2)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)

			shutdownServer := s.run()

			<-c
			shutdownServer()

			return nil
		},
	}

	rootCmd.PersistentFlags().StringP("config", "c", "", "path to config file")
	rootCmd.Execute()
}
