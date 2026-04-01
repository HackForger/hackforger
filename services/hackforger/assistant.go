// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

// AssistantResponse represents the AI assistant's response.
type AssistantResponse struct {
	Message    string `json:"message"`
	Status     string `json:"status"`     // "placeholder" | "ready"
	Disclaimer string `json:"disclaimer"`
}

// Chat returns a placeholder response. Future: swap for LLM integration.
func Chat(query string) *AssistantResponse {
	return &AssistantResponse{
		Message:    "AI assistant coming soon. Stay tuned!",
		Status:     "placeholder",
		Disclaimer: "AI responses may be inaccurate. Please double-check.",
	}
}
