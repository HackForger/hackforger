// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	flagURL    string
	flagToken  string
	flagOutput string
)

var rootCmd = &cobra.Command{
	Use:   "hackforger-cli",
	Short: "CLI for HackForger platform",
	Long:  "Command-line interface for interacting with a HackForger instance.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagURL, "url", os.Getenv("HACKFORGER_URL"), "HackForger instance URL (env: HACKFORGER_URL)")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", os.Getenv("HACKFORGER_TOKEN"), "API token (env: HACKFORGER_TOKEN)")
	rootCmd.PersistentFlags().StringVar(&flagOutput, "output", "json", "Output format: json")
}

func apiGet(path string) ([]byte, error) {
	url := strings.TrimRight(flagURL, "/") + "/api/v1/hackforger/" + strings.TrimLeft(path, "/")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if flagToken != "" {
		req.Header.Set("Authorization", "token "+flagToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func apiPost(path string, payload any) ([]byte, error) {
	url := strings.TrimRight(flagURL, "/") + "/api/v1/hackforger/" + strings.TrimLeft(path, "/")
	var bodyReader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		bodyReader = strings.NewReader(string(data))
	}
	req, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		return nil, err
	}
	if flagToken != "" {
		req.Header.Set("Authorization", "token "+flagToken)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func printJSON(data []byte) {
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		fmt.Println(string(data))
		return
	}
	pretty, _ := json.MarshalIndent(parsed, "", "  ")
	fmt.Println(string(pretty))
}
