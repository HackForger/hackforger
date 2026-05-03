// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Package hackforger — helper for hackathon registration's "which repos can
// this user use" question. Extracted to share between web (template dropdown)
// and API (POST validation).
package hackforger

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	perm_model "forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
)

// dropdownMax is the number of writable repos shown in the registration UI.
// Set conservatively for UX (long dropdown is unusable). Power users with
// more writable repos can call the API directly with any repo_id.
const dropdownMax = 10

// fetchPageSize is the SearchRepository page size BEFORE the Write+ filter.
// Larger than dropdownMax so the post-filter has enough candidates: a user
// might have read-only repos that take up early slots. 50 balances "enough
// candidates for 10 writable" against "not too many wasted permission lookups."
const fetchPageSize = 50

// ErrRepoAccessDenied is returned when a user tries to register with a repo
// they don't have write access to.
type ErrRepoAccessDenied struct {
	UserID int64
	RepoID int64
}

func (e ErrRepoAccessDenied) Error() string {
	return fmt.Sprintf("user %d lacks write permission to repo %d", e.UserID, e.RepoID)
}

// IsErrRepoAccessDenied checks if err is ErrRepoAccessDenied.
func IsErrRepoAccessDenied(err error) bool {
	_, ok := err.(ErrRepoAccessDenied)
	return ok
}

// ListUserWritableRepos returns up to `dropdownMax` repositories the user has
// at least Write access to, including org-owned repos. Used to populate the
// hackathon registration repo dropdown.
//
// Note on choice of SearchRepoOptions:
//   - Actor=user + Private=true uses AccessibleRepositoryCondition, so
//     visibility includes user-owned + org-team-granted + collaborator repos.
//   - AllPublic/AllLimited=false intentionally excludes repos where the user
//     is "just a member of a public org with no team grant" — those won't
//     pass the Write+ post-filter anyway, so skipping them up front is faster.
//   - Each returned repo gets its Owner loaded so templates can render
//     "owner_name/repo_name" without N+1 lazy-load.
func ListUserWritableRepos(ctx context.Context, user *user_model.User) ([]*repo_model.Repository, error) {
	repos, _, err := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		Actor:              user,
		Private:            true,
		AllPublic:          false,
		AllLimited:         false,
		IncludeDescription: false,
		ListOptions:        db.ListOptions{Page: 1, PageSize: fetchPageSize},
	})
	if err != nil {
		return nil, err
	}

	out := make([]*repo_model.Repository, 0, dropdownMax)
	for _, r := range repos {
		if len(out) >= dropdownMax {
			break
		}
		perm, err := access_model.GetUserRepoPermission(ctx, r, user)
		if err != nil {
			continue
		}
		if perm.AccessMode < perm_model.AccessModeWrite {
			continue
		}
		if err := r.LoadOwner(ctx); err != nil {
			continue // skip repo whose owner can't be loaded
		}
		out = append(out, r)
	}
	return out, nil
}

func UserCanRegisterRepo(ctx context.Context, user *user_model.User, repoID int64) (*repo_model.Repository, error) {
	return nil, nil
}
