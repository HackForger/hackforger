// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var reputationCmd = &cobra.Command{
	Use:   "reputation",
	Short: "View reputation",
}

var reputationGetCmd = &cobra.Command{
	Use:   "get [username]",
	Short: "Get user reputation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := apiGet(fmt.Sprintf("reputation/users/%s", args[0]))
		if err != nil {
			return err
		}
		printJSON(data)
		return nil
	},
}

var reputationLeaderboardCmd = &cobra.Command{
	Use:   "leaderboard",
	Short: "View reputation leaderboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := apiGet("reputation/leaderboard")
		if err != nil {
			return err
		}
		printJSON(data)
		return nil
	},
}

func init() {
	reputationCmd.AddCommand(reputationGetCmd, reputationLeaderboardCmd)
	rootCmd.AddCommand(reputationCmd)
}
