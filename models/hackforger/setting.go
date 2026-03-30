// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	"forgejo.org/models/db"
)

// HackforgerSetting stores admin-configurable key-value settings
// (e.g. reputation weights, tier thresholds).
type HackforgerSetting struct {
	ID    int64  `xorm:"pk autoincr"`
	Key   string `xorm:"VARCHAR(100) UNIQUE NOT NULL"`
	Value string `xorm:"TEXT NOT NULL"`
}

func init() {
	db.RegisterModel(new(HackforgerSetting))
}

// GetSetting returns the value stored for key, or "" if the key does not exist.
func GetSetting(ctx context.Context, key string) (string, error) {
	s := new(HackforgerSetting)
	has, err := db.GetEngine(ctx).Where("`key` = ?", key).Get(s)
	if err != nil {
		return "", err
	}
	if !has {
		return "", nil
	}
	return s.Value, nil
}

// GetSettingWithDefault returns the stored value for key, or defaultValue when
// the key is absent or an error occurs.
func GetSettingWithDefault(ctx context.Context, key, defaultValue string) string {
	val, err := GetSetting(ctx, key)
	if err != nil || val == "" {
		return defaultValue
	}
	return val
}

// SetSetting upserts a key-value pair in the settings table.
func SetSetting(ctx context.Context, key, value string) error {
	s := new(HackforgerSetting)
	has, err := db.GetEngine(ctx).Where("`key` = ?", key).Get(s)
	if err != nil {
		return err
	}
	if has {
		s.Value = value
		_, err = db.GetEngine(ctx).ID(s.ID).Cols("value").Update(s)
		return err
	}
	s = &HackforgerSetting{Key: key, Value: value}
	_, err = db.GetEngine(ctx).Insert(s)
	return err
}
