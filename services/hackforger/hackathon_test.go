// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"errors"
	"testing"

	"forgejo.org/modules/util"
	hackforger_service "forgejo.org/services/hackforger"

	"github.com/stretchr/testify/assert"
)

func TestErrNoCriteria_Error_Global(t *testing.T) {
	err := hackforger_service.ErrNoCriteria{HackathonID: 42}
	assert.Equal(t, "hackathon has no scoring criteria [id: 42]", err.Error())
}

func TestErrNoCriteria_Error_TrackScoped(t *testing.T) {
	err := hackforger_service.ErrNoCriteria{HackathonID: 42, TrackID: 7}
	assert.Equal(t, "track has no enabled scoring criteria [hackathon: 42, track: 7]", err.Error())
}

func TestErrNoCriteria_IsInvalidArgument(t *testing.T) {
	err := hackforger_service.ErrNoCriteria{HackathonID: 42}
	assert.True(t, errors.Is(err, util.ErrInvalidArgument),
		"ErrNoCriteria should unwrap to util.ErrInvalidArgument so middleware maps to 400")
}

func TestIsErrNoCriteria(t *testing.T) {
	err := hackforger_service.ErrNoCriteria{HackathonID: 1}
	assert.True(t, hackforger_service.IsErrNoCriteria(err))
	assert.False(t, hackforger_service.IsErrNoCriteria(errors.New("unrelated")))
}
