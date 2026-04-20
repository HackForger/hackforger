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

func TestErrNoRubricConfigured_Error(t *testing.T) {
	err := hackforger_service.ErrNoRubricConfigured{TrackID: 99}
	assert.Equal(t, "no scoring rubric configured for track [track: 99]", err.Error())
}

func TestErrNoRubricConfigured_IsInvalidArgument(t *testing.T) {
	err := hackforger_service.ErrNoRubricConfigured{TrackID: 1}
	assert.True(t, errors.Is(err, util.ErrInvalidArgument))
}

func TestIsErrNoRubricConfigured(t *testing.T) {
	err := hackforger_service.ErrNoRubricConfigured{TrackID: 1}
	assert.True(t, hackforger_service.IsErrNoRubricConfigured(err))
	assert.False(t, hackforger_service.IsErrNoRubricConfigured(errors.New("x")))
}
