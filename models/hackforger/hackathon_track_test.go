// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"encoding/json"
	"testing"

	hackforger_model "forgejo.org/models/hackforger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePrizeDistRatios(t *testing.T) {
	t.Run("Valid_TwoWay", func(t *testing.T) {
		ratios := []hackforger_model.PrizeDistRatio{
			{Rank: 1, Pct: 60},
			{Rank: 2, Pct: 40},
		}
		err := hackforger_model.ValidatePrizeDistRatios(ratios)
		assert.NoError(t, err)
	})

	t.Run("Valid_ThreeWay", func(t *testing.T) {
		ratios := []hackforger_model.PrizeDistRatio{
			{Rank: 1, Pct: 50},
			{Rank: 2, Pct: 30},
			{Rank: 3, Pct: 20},
		}
		err := hackforger_model.ValidatePrizeDistRatios(ratios)
		assert.NoError(t, err)
	})

	t.Run("SumNot100", func(t *testing.T) {
		ratios := []hackforger_model.PrizeDistRatio{
			{Rank: 1, Pct: 60},
			{Rank: 2, Pct: 30},
		}
		err := hackforger_model.ValidatePrizeDistRatios(ratios)
		require.Error(t, err)
		assert.True(t, hackforger_model.IsErrInvalidDistRatios(err))
		assert.Contains(t, err.Error(), "pct sum must be 100")
	})

	t.Run("NonContiguousRanks", func(t *testing.T) {
		ratios := []hackforger_model.PrizeDistRatio{
			{Rank: 1, Pct: 60},
			{Rank: 3, Pct: 40},
		}
		err := hackforger_model.ValidatePrizeDistRatios(ratios)
		require.Error(t, err)
		assert.True(t, hackforger_model.IsErrInvalidDistRatios(err))
		assert.Contains(t, err.Error(), "contiguous")
	})

	t.Run("ZeroPct", func(t *testing.T) {
		ratios := []hackforger_model.PrizeDistRatio{
			{Rank: 1, Pct: 100},
			{Rank: 2, Pct: 0},
		}
		err := hackforger_model.ValidatePrizeDistRatios(ratios)
		require.Error(t, err)
		assert.True(t, hackforger_model.IsErrInvalidDistRatios(err))
		assert.Contains(t, err.Error(), "pct must be > 0")
	})

	t.Run("Empty", func(t *testing.T) {
		err := hackforger_model.ValidatePrizeDistRatios(nil)
		require.Error(t, err)
		assert.True(t, hackforger_model.IsErrInvalidDistRatios(err))
		assert.Contains(t, err.Error(), "must not be empty")
	})
}

func TestParsePrizeDistRatios(t *testing.T) {
	t.Run("ValidJSON", func(t *testing.T) {
		input := `[{"rank":1,"pct":60},{"rank":2,"pct":40}]`
		ratios, err := hackforger_model.ParsePrizeDistRatios(input)
		require.NoError(t, err)
		require.Len(t, ratios, 2)
		assert.Equal(t, 1, ratios[0].Rank)
		assert.Equal(t, 60, ratios[0].Pct)
		assert.Equal(t, 2, ratios[1].Rank)
		assert.Equal(t, 40, ratios[1].Pct)
	})

	t.Run("EmptyString", func(t *testing.T) {
		ratios, err := hackforger_model.ParsePrizeDistRatios("")
		require.NoError(t, err)
		assert.Nil(t, ratios)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		_, err := hackforger_model.ParsePrizeDistRatios("not json")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid prize distribution ratios JSON")
	})

	t.Run("RoundTrip", func(t *testing.T) {
		original := []hackforger_model.PrizeDistRatio{
			{Rank: 1, Pct: 50},
			{Rank: 2, Pct: 30},
			{Rank: 3, Pct: 20},
		}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		parsed, err := hackforger_model.ParsePrizeDistRatios(string(data))
		require.NoError(t, err)
		assert.Equal(t, original, parsed)
	})
}
