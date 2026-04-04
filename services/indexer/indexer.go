// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package indexer

import (
	code_indexer "forgejo.org/modules/indexer/code"
	hackforger_indexer "forgejo.org/modules/indexer/hackforger"
	issue_indexer "forgejo.org/modules/indexer/issues"
	stats_indexer "forgejo.org/modules/indexer/stats"
	notify_service "forgejo.org/services/notify"
)

// Init initialize the repo indexer
func Init() error {
	notify_service.RegisterNotifier(NewNotifier())

	issue_indexer.InitIssueIndexer(false)
	hackforger_indexer.InitHackforgerIndexer(false)
	code_indexer.Init()
	return stats_indexer.Init()
}
