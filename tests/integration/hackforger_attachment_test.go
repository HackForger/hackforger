// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIHackforgerAttachment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("Upload happy path", func(t *testing.T) {
		_, token := hackforgerLoginAs(t, 2)

		body := &bytes.Buffer{}
		mw := multipart.NewWriter(body)
		fw, err := mw.CreateFormFile("file", "design.png")
		assert.NoError(t, err)
		// 1x1 PNG (smallest valid png) so the upload service accepts it.
		_, err = fw.Write([]byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
			0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
			0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
			0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
			0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
			0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
			0x42, 0x60, 0x82,
		})
		assert.NoError(t, err)
		mw.Close()

		req := NewRequestWithBody(t, "POST", "/api/v1/hackforger/attachments", body).
			SetHeader("Content-Type", mw.FormDataContentType()).
			AddTokenAuth(token)

		resp := MakeRequest(t, req, http.StatusOK)
		var got struct {
			UUID string `json:"uuid"`
		}
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.NotEmpty(t, got.UUID)
		assert.True(t, strings.Contains(got.UUID, "-"), "uuid should be uuid-shaped")
	})

	t.Run("401 without token", func(t *testing.T) {
		body := &bytes.Buffer{}
		mw := multipart.NewWriter(body)
		mw.Close()
		req := NewRequestWithBody(t, "POST", "/api/v1/hackforger/attachments", body).
			SetHeader("Content-Type", mw.FormDataContentType())
		MakeRequest(t, req, http.StatusUnauthorized)
	})
}
