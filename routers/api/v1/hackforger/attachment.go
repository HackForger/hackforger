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
	"forgejo.org/services/context/upload"
	"forgejo.org/services/context"
)

// AttachmentResponse represents the JSON returned after an upload.
// swagger:response HackforgerAttachment
type AttachmentResponse struct {
	UUID string `json:"uuid"`
}

// UploadAttachmentAPI uploads a platform-level (RepoID=-1) attachment used
// by hackathon descriptions, grant project pages, and submissions.
func UploadAttachmentAPI(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/attachments hackforgerAttachment hackforgerUploadAttachment
	// ---
	// summary: Upload a platform-level attachment (HackForger rich text)
	// consumes:
	//   - multipart/form-data
	// parameters:
	//   - name: file
	//     in: formData
	//     type: file
	//     required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/HackforgerAttachment"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	if !setting.Attachment.Enabled {
		ctx.Error(http.StatusNotFound, "AttachmentDisabled", "attachment is not enabled")
		return
	}

	file, header, err := ctx.Req.FormFile("file")
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "FormFile", fmt.Sprintf("FormFile: %v", err))
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
			ctx.Error(http.StatusBadRequest, "FileTypeForbidden", "file type not allowed")
			return
		}
		log.Error("UploadAttachmentAPI: %v", err)
		ctx.Error(http.StatusInternalServerError, "UploadFailed", "upload failed")
		return
	}

	ctx.JSON(http.StatusOK, AttachmentResponse{UUID: attach.UUID})
}
