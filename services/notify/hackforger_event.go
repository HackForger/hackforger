// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package notify

import activities_model "forgejo.org/models/activities"

// HackforgerAudienceType determines how feed events are distributed.
// Values are bit flags and may be combined with bitwise OR.
type HackforgerAudienceType int

const (
	AudienceGlobal       HackforgerAudienceType = 1 << iota // 1 - UserID=0, visible to everyone
	AudienceFollowers                                        // 2 - Visible to ActUser's followers
	AudienceOrgMembers                                       // 4 - Visible to org members
	AudienceRepoWatchers                                     // 8 - Visible to repo watchers
	AudienceDirectUser                                       // 16 - Visible to a single target user
)

// HackforgerEventOpts holds parameters for HackForger entity events.
type HackforgerEventOpts struct {
	OpType       activities_model.ActionType // e.g., ActionHackathonCreated
	EntityType   string                      // "hackathon" | "bounty" | "grant" | "submission"
	EntityID     int64
	EntityName   string
	EntitySlug   string
	OrgID        int64
	RepoID       int64
	AudienceType HackforgerAudienceType // bit flags for Feed distribution
	TargetUserID int64                  // for direct-user audience
	Content      any                    // JSON-serializable extra data
}
