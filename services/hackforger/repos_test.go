// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"testing"

	hackforger_service "forgejo.org/services/hackforger"

	"github.com/stretchr/testify/assert"
)

func TestErrRepoAccessDenied_Error(t *testing.T) {
	err := hackforger_service.ErrRepoAccessDenied{UserID: 42, RepoID: 7}
	assert.Equal(t, "user 42 lacks write permission to repo 7", err.Error())
}

func TestIsErrRepoAccessDenied(t *testing.T) {
	err := hackforger_service.ErrRepoAccessDenied{UserID: 1, RepoID: 2}
	assert.True(t, hackforger_service.IsErrRepoAccessDenied(err))
	assert.False(t, hackforger_service.IsErrRepoAccessDenied(nil))
	assert.False(t, hackforger_service.IsErrRepoAccessDenied(assert.AnError))
}
