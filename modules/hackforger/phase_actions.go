// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import "fmt"

// Action vocabularies per activity kind.
// Adding a new action requires: register constant here + implement business logic.
var actionVocabulary = map[string]map[string]string{
	"hackathon": {
		"register":        "hackforger.action.register",
		"form_team":       "hackforger.action.form_team",
		"submit_work":     "hackforger.action.submit_work",
		"score":           "hackforger.action.score",
		"publish_results": "hackforger.action.publish_results",
	},
	"bounty": {
		"apply":     "hackforger.action.apply",
		"claim":     "hackforger.action.claim",
		"submit_pr": "hackforger.action.submit_pr",
		"review":    "hackforger.action.review",
		"settle":    "hackforger.action.settle",
	},
	"grant": {
		"apply":    "hackforger.action.apply",
		"review":   "hackforger.action.review",
		"approve":  "hackforger.action.approve",
		"disburse": "hackforger.action.disburse",
	},
}

// GetActionsForKind returns all valid action strings for an activity kind.
func GetActionsForKind(activityKind string) (map[string]string, error) {
	actions, ok := actionVocabulary[activityKind]
	if !ok {
		return nil, fmt.Errorf("unknown activity kind: %s", activityKind)
	}
	return actions, nil
}

// ValidateActions checks that all actions are valid for the given activity kind.
func ValidateActions(activityKind string, actions []string) error {
	vocab, err := GetActionsForKind(activityKind)
	if err != nil {
		return err
	}
	for _, action := range actions {
		if _, ok := vocab[action]; !ok {
			return fmt.Errorf("invalid action %q for activity kind %q", action, activityKind)
		}
	}
	return nil
}
