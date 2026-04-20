// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"html/template"
	"net/http"
	"strings"

	"forgejo.org/modules/markup"
	"forgejo.org/modules/markup/markdown"
	"forgejo.org/modules/options"
	"forgejo.org/services/context"
)

const tplHelp = "hackforger/help"

// helpLangFile resolves a markdown filename for the given section using the
// requester's locale. Falls back to en-US when the locale file is missing.
func helpLangFile(ctx *context.Context, section string) (string, []byte, error) {
	lang := ctx.Locale.Language() // e.g. "zh-CN", "en-US"
	tryFiles := []string{
		section + "." + lang + ".md",
		section + ".en-US.md",
	}
	var lastErr error
	for _, name := range tryFiles {
		content, err := options.AssetFS().ReadFile("hackforger-help", name)
		if err == nil {
			return name, content, nil
		}
		lastErr = err
	}
	return "", nil, lastErr
}

func renderHelpSection(ctx *context.Context, section string) (template.HTML, error) {
	_, md, err := helpLangFile(ctx, section)
	if err != nil {
		return "", err
	}
	rendered, err := markdown.RenderString(&markup.RenderContext{Ctx: ctx}, strings.TrimSpace(string(md)))
	if err != nil {
		return "", err
	}
	return rendered, nil
}

// HelpPage renders the in-app help center — platform (Synnovator) + system (HackForger).
func HelpPage(ctx *context.Context) {
	platformHTML, err := renderHelpSection(ctx, "platform")
	if err != nil {
		ctx.ServerError("HelpPlatform", err)
		return
	}
	systemHTML, err := renderHelpSection(ctx, "system")
	if err != nil {
		ctx.ServerError("HelpSystem", err)
		return
	}
	ctx.Data["Title"] = ctx.Tr("hackforger.help.title")
	ctx.Data["SynnovatorHTML"] = platformHTML
	ctx.Data["HackforgerHTML"] = systemHTML
	ctx.HTML(http.StatusOK, tplHelp)
}
