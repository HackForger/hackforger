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

// Stub function bodies — implemented in Tasks 2 and 3.
func ListUserWritableRepos(ctx context.Context, user *user_model.User) ([]*repo_model.Repository, error) {
	_ = db.ListOptions{}
	_ = perm_model.AccessModeWrite
	_ = access_model.GetUserRepoPermission
	_ = repo_model.SearchRepository
	return nil, nil
}

func UserCanRegisterRepo(ctx context.Context, user *user_model.User, repoID int64) (*repo_model.Repository, error) {
	return nil, nil
}
