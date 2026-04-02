// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import "github.com/spf13/cobra"

var creditsCmd = &cobra.Command{
	Use:   "credits",
	Short: "Manage credits",
}

var creditsBalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Get your credits balance",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := apiGet("credits/balance")
		if err != nil {
			return err
		}
		printJSON(data)
		return nil
	},
}

var creditsTransactionsCmd = &cobra.Command{
	Use:   "transactions",
	Short: "List credit transactions",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := apiGet("credits/transactions")
		if err != nil {
			return err
		}
		printJSON(data)
		return nil
	},
}

func init() {
	creditsCmd.AddCommand(creditsBalanceCmd, creditsTransactionsCmd)
	rootCmd.AddCommand(creditsCmd)
}
