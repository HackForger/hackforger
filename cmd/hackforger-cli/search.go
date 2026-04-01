// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search HackForger entities",
}

var searchQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Search by keyword",
	RunE: func(cmd *cobra.Command, args []string) error {
		q, _ := cmd.Flags().GetString("q")
		scope, _ := cmd.Flags().GetString("scope")
		page, _ := cmd.Flags().GetInt("page")
		limit, _ := cmd.Flags().GetInt("limit")
		path := fmt.Sprintf("search?q=%s&scope=%s&page=%d&limit=%d", q, scope, page, limit)
		data, err := apiGet(path)
		if err != nil {
			return err
		}
		printJSON(data)
		return nil
	},
}

func init() {
	searchQueryCmd.Flags().String("q", "", "Search keyword")
	searchQueryCmd.Flags().String("scope", "all", "Scope: all, hackathons, bounties, grants")
	searchQueryCmd.Flags().Int("page", 1, "Page number")
	searchQueryCmd.Flags().Int("limit", 20, "Results per page")
	searchCmd.AddCommand(searchQueryCmd)
	rootCmd.AddCommand(searchCmd)
}
