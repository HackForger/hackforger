// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package structs

import "forgejo.org/modules/json"

// HackforgerWebhookPayload represents a HackForger webhook event payload.
type HackforgerWebhookPayload struct {
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   int64          `json:"entity_id"`
	EntityName string         `json:"entity_name"`
	Sender     *User          `json:"sender"`
	Extra      map[string]any `json:"extra,omitempty"`
}

// JSONPayload implements Payloader.
func (p *HackforgerWebhookPayload) JSONPayload() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// Ensure compile-time interface compliance.
var _ Payloader = &HackforgerWebhookPayload{}
