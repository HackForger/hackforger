// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var hackathonCmd = &cobra.Command{
	Use:   "hackathon",
	Short: "Manage hackathons",
}

var hackathonListCmd = &cobra.Command{
	Use:   "list",
	Short: "List hackathons",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := apiGet("hackathons")
		if err != nil {
			return err
		}
		printJSON(data)
		return nil
	},
}

var hackathonGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get hackathon details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := apiGet(fmt.Sprintf("hackathons/%s", args[0]))
		if err != nil {
			return err
		}
		printJSON(data)
		return nil
	},
}

func init() {
	hackathonCmd.AddCommand(hackathonListCmd, hackathonGetCmd)
	rootCmd.AddCommand(hackathonCmd)
}
