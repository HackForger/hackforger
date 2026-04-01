// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var bountyCmd = &cobra.Command{
	Use:   "bounty",
	Short: "Manage bounties",
}

var bountyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all bounties",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := apiGet("bounties")
		if err != nil {
			return err
		}
		printJSON(data)
		return nil
	},
}

var bountyGetCmd = &cobra.Command{
	Use:   "get [owner] [repo] [id]",
	Short: "Get bounty details",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := fmt.Sprintf("../repos/%s/%s/hackforger/bounties/%s", args[0], args[1], args[2])
		data, err := apiGet(path)
		if err != nil {
			return err
		}
		printJSON(data)
		return nil
	},
}

func init() {
	bountyCmd.AddCommand(bountyListCmd, bountyGetCmd)
	rootCmd.AddCommand(bountyCmd)
}
