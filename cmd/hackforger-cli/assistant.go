// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import "github.com/spf13/cobra"

var assistantCmd = &cobra.Command{
	Use:   "assistant",
	Short: "AI assistant",
}

var assistantChatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat with AI assistant",
	RunE: func(cmd *cobra.Command, args []string) error {
		query, _ := cmd.Flags().GetString("query")
		data, err := apiPost("assistant/chat", map[string]string{"query": query})
		if err != nil {
			return err
		}
		printJSON(data)
		return nil
	},
}

func init() {
	assistantChatCmd.Flags().String("query", "", "Question to ask")
	assistantCmd.AddCommand(assistantChatCmd)
	rootCmd.AddCommand(assistantCmd)
}
