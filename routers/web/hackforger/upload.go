// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"
	"net/http"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/services/attachment"
	"forgejo.org/services/context"
	"forgejo.org/services/context/upload"
)

// UploadHackforgerAttachment handles file uploads for HackForger entities (hackathon descriptions, etc.)
func UploadHackforgerAttachment(ctx *context.Context) {
	if !setting.Attachment.Enabled {
		ctx.Error(http.StatusNotFound, "attachment is not enabled")
		return
	}

	file, header, err := ctx.Req.FormFile("file")
	if err != nil {
		ctx.Error(http.StatusInternalServerError, fmt.Sprintf("FormFile: %v", err))
		return
	}
	defer file.Close()

	attach, err := attachment.UploadAttachment(ctx, file, setting.Attachment.AllowedTypes, header.Size, &repo_model.Attachment{
		Name:       header.Filename,
		UploaderID: ctx.Doer.ID,
		RepoID:     -1, // HackForger platform-level attachment (no specific repo)
	})
	if err != nil {
		if upload.IsErrFileTypeForbidden(err) {
			ctx.Error(http.StatusBadRequest, err.Error())
			return
		}
		ctx.Error(http.StatusInternalServerError, fmt.Sprintf("UploadAttachment: %v", err))
		return
	}

	log.Trace("HackForger attachment uploaded: %s", attach.UUID)
	ctx.JSON(http.StatusOK, map[string]string{
		"uuid": attach.UUID,
	})
}
