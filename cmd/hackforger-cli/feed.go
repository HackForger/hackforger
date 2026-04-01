// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "View activity feed",
}

var feedListCmd = &cobra.Command{
	Use:   "list",
	Short: "List feed events",
	RunE: func(cmd *cobra.Command, args []string) error {
		feedType, _ := cmd.Flags().GetString("type")
		data, err := apiGet(fmt.Sprintf("feed?type=%s", feedType))
		if err != nil {
			return err
		}
		printJSON(data)
		return nil
	},
}

func init() {
	feedListCmd.Flags().String("type", "global", "Feed type: global, following")
	feedCmd.AddCommand(feedListCmd)
	rootCmd.AddCommand(feedCmd)
}
