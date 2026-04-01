// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var grantCmd = &cobra.Command{
	Use:   "grant",
	Short: "Manage grants",
}

var grantRoundListCmd = &cobra.Command{
	Use:   "round-list",
	Short: "List grant rounds",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := apiGet("grant-rounds")
		if err != nil {
			return err
		}
		printJSON(data)
		return nil
	},
}

var grantRoundGetCmd = &cobra.Command{
	Use:   "round-get [id]",
	Short: "Get grant round details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := apiGet(fmt.Sprintf("grant-rounds/%s", args[0]))
		if err != nil {
			return err
		}
		printJSON(data)
		return nil
	},
}

func init() {
	grantCmd.AddCommand(grantRoundListCmd, grantRoundGetCmd)
	rootCmd.AddCommand(grantCmd)
}
