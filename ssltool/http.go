package main

import "github.com/spf13/cobra"

func newHttpCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "http",
		Short: "http",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
