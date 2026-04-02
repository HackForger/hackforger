// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

// HookEvents is a set of web hook events
// update TestCreateWebhook in models/webhook/webhook_test.go when adding or changing values here
type HookEvents struct {
	Create                   bool `json:"create"`
	Delete                   bool `json:"delete"`
	Fork                     bool `json:"fork"`
	Issues                   bool `json:"issues"`
	IssueAssign              bool `json:"issue_assign"`
	IssueLabel               bool `json:"issue_label"`
	IssueMilestone           bool `json:"issue_milestone"`
	IssueComment             bool `json:"issue_comment"`
	Push                     bool `json:"push"`
	PullRequest              bool `json:"pull_request"`
	PullRequestAssign        bool `json:"pull_request_assign"`
	PullRequestLabel         bool `json:"pull_request_label"`
	PullRequestMilestone     bool `json:"pull_request_milestone"`
	PullRequestComment       bool `json:"pull_request_comment"`
	PullRequestReview        bool `json:"pull_request_review"`
	PullRequestSync          bool `json:"pull_request_sync"`
	PullRequestReviewRequest bool `json:"pull_request_review_request"`
	Wiki                     bool `json:"wiki"`
	Repository               bool `json:"repository"`
	Release                  bool `json:"release"`
	Package                  bool `json:"package"`
	ActionRunFailure         bool `json:"action_run_failure"`
	ActionRunRecover         bool `json:"action_run_recover"`
	ActionRunSuccess         bool `json:"action_run_success"`

	// HackForger events
	HackathonCreated       bool `json:"hackathon_created"`
	HackathonStatusChanged bool `json:"hackathon_status_changed"`
	HackathonSubmission    bool `json:"hackathon_submission"`
	HackathonScored        bool `json:"hackathon_scored"`
	HackathonFinalized     bool `json:"hackathon_finalized"`
	BountyCreated          bool `json:"bounty_created"`
	BountyApplication      bool `json:"bounty_application"`
	BountyClaimed          bool `json:"bounty_claimed"`
	BountyCompleted        bool `json:"bounty_completed"`
	BountyPaid             bool `json:"bounty_paid"`
	BountyWinners          bool `json:"bounty_winners"`
	BountyExpired          bool `json:"bounty_expired"`
	BountyCancelled        bool `json:"bounty_cancelled"`
	GrantRoundCreated      bool `json:"grant_round_created"`
	GrantRoundOpened       bool `json:"grant_round_opened"`
	GrantProjectSubmitted  bool `json:"grant_project_submitted"`
	GrantAwarded           bool `json:"grant_awarded"`
	GrantRoundFinalized    bool `json:"grant_round_finalized"`
	CreditsDeposited       bool `json:"credits_deposited"`
	CreditsRedeemed        bool `json:"credits_redeemed"`
}

// HookEvent represents events that will deliver a hook.
type HookEvent struct {
	PushOnly       bool   `json:"push_only"`
	SendEverything bool   `json:"send_everything"`
	ChooseEvents   bool   `json:"choose_events"`
	BranchFilter   string `json:"branch_filter"`

	HookEvents `json:"events"`
}
