package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	httpPort     int
	urlCfgPath   string
	urlCachePath string

	lookupCmd = &cobra.Command{
		Use:   "url-lookup",
		Short: "URL lookup service.",
		Long:  "URL lookup service returns URL information including if it's safe to open.",
		Args:  cobra.ExactArgs(0),
		RunE: func(c *cobra.Command, args []string) error {
			if _, err := os.Stat(urlCfgPath); err != nil {
				fmt.Printf("URL configuration path '%v': %v\n", urlCfgPath, err)
				os.Exit(-1)
			}

			if _, err := os.Stat(urlCachePath); err != nil {
				fmt.Printf("URL cache path '%v': %v\n", urlCachePath, err)
				os.Exit(-1)
			}

			stop := make(chan struct{})
			err := newLookupServer(httpPort, urlCfgPath, urlCachePath, stop)
			waitSignal(stop)
			return err
		},
	}
)

func waitSignal(stop chan struct{}) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	close(stop)
}

func init() {
	lookupCmd.PersistentFlags().IntVar(&httpPort, "port", 16888, "URL lookup service port")
	lookupCmd.PersistentFlags().StringVar(&urlCfgPath, "url-config-path", "", "URL configuration path")
	lookupCmd.PersistentFlags().StringVar(&urlCachePath, "url-cache-path", "", "URL cache path")
	lookupCmd.MarkPersistentFlagRequired("url-config-path")
	lookupCmd.MarkPersistentFlagRequired("url-cache-path")
}

func main() {
	if err := lookupCmd.Execute(); err != nil {
		fmt.Printf("Failed to start url lookup service: %v", err)
		os.Exit(-1)
	}
}
