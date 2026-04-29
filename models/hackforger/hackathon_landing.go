// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	"forgejo.org/models/db"
)

// ListPublishedHackathonsBySlugs returns hackathons matching the given slug list (only published).
// Used by the landing page stage hydration to look up which hackathons fill which slot.
func ListPublishedHackathonsBySlugs(ctx context.Context, slugs []string) ([]*Hackathon, error) {
	if len(slugs) == 0 {
		return []*Hackathon{}, nil
	}
	hackathons := []*Hackathon{}
	err := db.GetEngine(ctx).In("slug", slugs).Where("is_published = ?", true).Find(&hackathons)
	return hackathons, err
}

// ListTracksByHackathonIDs returns tracks grouped by hackathon ID.
// Single IN query to avoid N+1.
func ListTracksByHackathonIDs(ctx context.Context, ids []int64) (map[int64][]*HackathonTrack, error) {
	if len(ids) == 0 {
		return map[int64][]*HackathonTrack{}, nil
	}
	tracks := []*HackathonTrack{}
	if err := db.GetEngine(ctx).In("hackathon_id", ids).Find(&tracks); err != nil {
		return nil, err
	}
	result := make(map[int64][]*HackathonTrack)
	for _, t := range tracks {
		result[t.HackathonID] = append(result[t.HackathonID], t)
	}
	return result, nil
}
