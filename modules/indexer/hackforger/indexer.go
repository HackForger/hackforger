// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	hackforger_bleve "forgejo.org/modules/indexer/hackforger/bleve"
	hackforger_db "forgejo.org/modules/indexer/hackforger/db"
	"forgejo.org/modules/indexer/hackforger/internal"
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"
	"forgejo.org/modules/process"
	"forgejo.org/modules/queue"
	"forgejo.org/modules/setting"
)

// IndexerMetadata is used to send data to the queue.
type IndexerMetadata struct {
	ID         int64  `json:"id"`
	EntityType string `json:"entity_type"`
	IsDelete   bool   `json:"is_delete"`
}

var (
	hackforgerIndexerQueue *queue.WorkerPoolQueue[*IndexerMetadata]
	globalIndexer          atomic.Pointer[internal.Indexer]
	dummyIndexer           *internal.Indexer
)

func init() {
	i := internal.NewDummyIndexer()
	dummyIndexer = &i
	globalIndexer.Store(dummyIndexer)
}

// InitHackforgerIndexer initializes the HackForger entity indexer.
// If syncReindex is true, it blocks until the full reindex completes.
func InitHackforgerIndexer(syncReindex bool) {
	ctx, _, finished := process.GetManager().AddTypedContext(context.Background(), "Service: HackforgerIndexer", process.SystemProcessType, false)

	indexerInitWaitChannel := make(chan time.Duration, 1)

	// Create the queue
	hackforgerIndexerQueue = queue.CreateSimpleQueue(ctx, "hackforger_indexer", getQueueHandler(ctx))

	graceful.GetManager().RunAtTerminate(finished)

	go func() {
		start := time.Now()
		log.Info("PID %d: Initializing HackForger Indexer: %s", os.Getpid(), setting.Indexer.HackforgerType)

		var (
			indexer internal.Indexer
			existed bool
			err     error
		)

		switch setting.Indexer.HackforgerType {
		case "bleve":
			defer func() {
				if r := recover(); r != nil {
					log.Error("PANIC whilst initializing HackForger indexer: %v\nStacktrace: %s", r, log.Stack(2))
					log.Error("You can remove the %q directory to recreate the indexes", setting.Indexer.HackforgerPath)
					globalIndexer.Store(dummyIndexer)
					log.Fatal("PID: %d Unable to initialize Bleve HackForger Indexer at path: %s Error: %v", os.Getpid(), setting.Indexer.HackforgerPath, r)
				}
			}()
			indexer = hackforger_bleve.NewIndexer(setting.Indexer.HackforgerPath)
			existed, err = indexer.Init(ctx)
			if err != nil {
				log.Fatal("Unable to initialize Bleve HackForger Indexer at path: %s Error: %v", setting.Indexer.HackforgerPath, err)
			}
		case "db":
			indexer = hackforger_db.NewIndexer()
		default:
			log.Fatal("Unknown HackForger indexer type: %s", setting.Indexer.HackforgerType)
		}

		globalIndexer.Store(&indexer)

		graceful.GetManager().RunAtTerminate(func() {
			log.Debug("Closing HackForger indexer")
			(*globalIndexer.Load()).Close()
			log.Info("PID: %d HackForger Indexer closed", os.Getpid())
		})

		// Start processing the queue
		go graceful.GetManager().RunWithCancel(hackforgerIndexerQueue)

		// Populate the index if it's new
		if !existed {
			if syncReindex {
				graceful.GetManager().RunWithShutdownContext(populateHackforgerIndexer)
			} else {
				go graceful.GetManager().RunWithShutdownContext(populateHackforgerIndexer)
			}
		}

		indexerInitWaitChannel <- time.Since(start)
		close(indexerInitWaitChannel)
	}()

	if syncReindex {
		select {
		case <-indexerInitWaitChannel:
		case <-graceful.GetManager().IsShutdown():
		}
	} else if setting.Indexer.StartupTimeout > 0 {
		go func() {
			timeout := setting.Indexer.StartupTimeout
			if graceful.GetManager().IsChild() && setting.GracefulHammerTime > 0 {
				timeout += setting.GracefulHammerTime
			}
			select {
			case duration := <-indexerInitWaitChannel:
				log.Info("HackForger Indexer Initialization took %v", duration)
			case <-graceful.GetManager().IsShutdown():
				log.Warn("Shutdown occurred before HackForger index initialisation was complete")
			case <-time.After(timeout):
				hackforgerIndexerQueue.ShutdownWait(5 * time.Second)
				log.Fatal("HackForger Indexer Initialization timed-out after: %v", timeout)
			}
		}()
	}
}

func getQueueHandler(ctx context.Context) func(items ...*IndexerMetadata) []*IndexerMetadata {
	return func(items ...*IndexerMetadata) []*IndexerMetadata {
		var unhandled []*IndexerMetadata

		indexer := *globalIndexer.Load()
		for _, item := range items {
			log.Trace("HackForger IndexerMetadata Process: %s %d (delete=%t)", item.EntityType, item.ID, item.IsDelete)
			if item.IsDelete {
				if err := indexer.Delete(ctx, item.EntityType, item.ID); err != nil {
					log.Error("HackForger indexer handler: failed to delete %s/%d from index: %v", item.EntityType, item.ID, err)
					unhandled = append(unhandled, item)
				}
				continue
			}

			data, existed, err := loadIndexerData(ctx, item.EntityType, item.ID)
			if err != nil {
				log.Error("HackForger indexer handler: failed to load %s/%d: %v", item.EntityType, item.ID, err)
				unhandled = append(unhandled, item)
				continue
			}
			if !existed {
				if err := indexer.Delete(ctx, item.EntityType, item.ID); err != nil {
					log.Error("HackForger indexer handler: failed to delete non-existent %s/%d from index: %v", item.EntityType, item.ID, err)
					unhandled = append(unhandled, item)
				}
				continue
			}
			if err := indexer.Index(ctx, data); err != nil {
				log.Error("HackForger indexer handler: failed to index %s/%d: %v", item.EntityType, item.ID, err)
				unhandled = append(unhandled, item)
				continue
			}
		}

		return unhandled
	}
}

// loadIndexerData fetches entity data from the DB and converts it to IndexerData.
func loadIndexerData(ctx context.Context, entityType string, id int64) (*internal.IndexerData, bool, error) {
	switch entityType {
	case "hackathon":
		h, err := hackforger_model.GetHackathonByID(ctx, id)
		if err != nil {
			if hackforger_model.IsErrHackathonNotExist(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		return &internal.IndexerData{
			ID:          h.ID,
			EntityType:  "hackathon",
			Title:       h.Name,
			Content:     h.Description,
			Status:      int(h.Status),
			OwnerID:     h.OwnerID,
			OrgID:       h.OrgID,
			CreatedUnix: int64(h.CreatedUnix),
			UpdatedUnix: int64(h.UpdatedUnix),
		}, true, nil

	case "bounty":
		b, err := hackforger_model.GetBountyByID(ctx, id)
		if err != nil {
			if hackforger_model.IsErrBountyNotExist(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		return &internal.IndexerData{
			ID:          b.ID,
			EntityType:  "bounty",
			Title:       b.Title,
			Status:      int(b.Status),
			OwnerID:     b.PublisherID,
			RepoID:      b.RepoID,
			CreatedUnix: int64(b.CreatedUnix),
			UpdatedUnix: int64(b.UpdatedUnix),
		}, true, nil

	case "grant":
		r, err := hackforger_model.GetGrantRoundByID(ctx, id)
		if err != nil {
			if hackforger_model.IsErrGrantRoundNotExist(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		return &internal.IndexerData{
			ID:          r.ID,
			EntityType:  "grant",
			Title:       r.Name,
			Content:     r.Description,
			Status:      int(r.Status),
			OwnerID:     r.OwnerID,
			OrgID:       r.OrgID,
			CreatedUnix: int64(r.CreatedUnix),
			UpdatedUnix: int64(r.UpdatedUnix),
		}, true, nil

	case "submission":
		s, err := hackforger_model.GetSubmissionByID(ctx, id)
		if err != nil {
			if hackforger_model.IsErrSubmissionNotExist(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		return &internal.IndexerData{
			ID:          s.ID,
			EntityType:  "submission",
			Title:       s.Title,
			Content:     s.Description,
			Status:      int(s.Status),
			OwnerID:     s.UserID,
			HackathonID: s.HackathonID,
			RepoID:      s.RepoID,
			CreatedUnix: int64(s.CreatedUnix),
			UpdatedUnix: int64(s.UpdatedUnix),
		}, true, nil

	default:
		return nil, false, fmt.Errorf("unknown entity type: %s", entityType)
	}
}

// populateHackforgerIndexer populates the index with all existing entities.
func populateHackforgerIndexer(ctx context.Context) {
	ctx, _, finished := process.GetManager().AddTypedContext(ctx, "Service: PopulateHackforgerIndexer", process.SystemProcessType, true)
	defer finished()

	if err := PopulateHackforgerIndexer(ctx); err != nil {
		log.Error("HackForger indexer population failed: %v", err)
	}
}

// PopulateHackforgerIndexer indexes all HackForger entities from the database.
func PopulateHackforgerIndexer(ctx context.Context) error {
	indexer := *globalIndexer.Load()

	// Index hackathons
	for page := 1; ; page++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		hackathons, _, err := hackforger_model.ListHackathons(ctx, hackforger_model.ListHackathonsOptions{
			ListOptions: db.ListOptions{Page: page, PageSize: 50},
		})
		if err != nil {
			return fmt.Errorf("populate hackathons page %d: %w", page, err)
		}
		if len(hackathons) == 0 {
			break
		}
		var batch []*internal.IndexerData
		for _, h := range hackathons {
			batch = append(batch, &internal.IndexerData{
				ID:          h.ID,
				EntityType:  "hackathon",
				Title:       h.Name,
				Content:     h.Description,
				Status:      int(h.Status),
				OwnerID:     h.OwnerID,
				OrgID:       h.OrgID,
				CreatedUnix: int64(h.CreatedUnix),
				UpdatedUnix: int64(h.UpdatedUnix),
			})
		}
		if err := indexer.Index(ctx, batch...); err != nil {
			return fmt.Errorf("index hackathons batch page %d: %w", page, err)
		}
	}

	// Index bounties
	for page := 1; ; page++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		bounties, _, err := hackforger_model.ListBounties(ctx, hackforger_model.ListBountiesOptions{
			ListOptions: db.ListOptions{Page: page, PageSize: 50},
		})
		if err != nil {
			return fmt.Errorf("populate bounties page %d: %w", page, err)
		}
		if len(bounties) == 0 {
			break
		}
		var batch []*internal.IndexerData
		for _, b := range bounties {
			batch = append(batch, &internal.IndexerData{
				ID:          b.ID,
				EntityType:  "bounty",
				Title:       b.Title,
				Status:      int(b.Status),
				OwnerID:     b.PublisherID,
				RepoID:      b.RepoID,
				CreatedUnix: int64(b.CreatedUnix),
				UpdatedUnix: int64(b.UpdatedUnix),
			})
		}
		if err := indexer.Index(ctx, batch...); err != nil {
			return fmt.Errorf("index bounties batch page %d: %w", page, err)
		}
	}

	// Index grant rounds
	for page := 1; ; page++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rounds, _, err := hackforger_model.ListGrantRounds(ctx, hackforger_model.ListGrantRoundsOptions{
			ListOptions: db.ListOptions{Page: page, PageSize: 50},
		})
		if err != nil {
			return fmt.Errorf("populate grant rounds page %d: %w", page, err)
		}
		if len(rounds) == 0 {
			break
		}
		var batch []*internal.IndexerData
		for _, r := range rounds {
			batch = append(batch, &internal.IndexerData{
				ID:          r.ID,
				EntityType:  "grant",
				Title:       r.Name,
				Content:     r.Description,
				Status:      int(r.Status),
				OwnerID:     r.OwnerID,
				OrgID:       r.OrgID,
				CreatedUnix: int64(r.CreatedUnix),
				UpdatedUnix: int64(r.UpdatedUnix),
			})
		}
		if err := indexer.Index(ctx, batch...); err != nil {
			return fmt.Errorf("index grant rounds batch page %d: %w", page, err)
		}
	}

	// Index submissions
	for page := 1; ; page++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		submissions, _, err := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
			ListOptions: db.ListOptions{Page: page, PageSize: 50},
		})
		if err != nil {
			return fmt.Errorf("populate submissions page %d: %w", page, err)
		}
		if len(submissions) == 0 {
			break
		}
		var batch []*internal.IndexerData
		for _, s := range submissions {
			batch = append(batch, &internal.IndexerData{
				ID:          s.ID,
				EntityType:  "submission",
				Title:       s.Title,
				Content:     s.Description,
				Status:      int(s.Status),
				OwnerID:     s.UserID,
				HackathonID: s.HackathonID,
				RepoID:      s.RepoID,
				CreatedUnix: int64(s.CreatedUnix),
				UpdatedUnix: int64(s.UpdatedUnix),
			})
		}
		if err := indexer.Index(ctx, batch...); err != nil {
			return fmt.Errorf("index submissions batch page %d: %w", page, err)
		}
	}

	log.Debug("HackForger Indexer population complete")
	return nil
}

// UpdateHackforgerIndexer pushes an entity update to the indexer queue.
func UpdateHackforgerIndexer(ctx context.Context, entityType string, ids ...int64) {
	for _, id := range ids {
		if err := pushQueue(ctx, &IndexerMetadata{ID: id, EntityType: entityType}); err != nil {
			log.Error("Unable to push %s/%d to HackForger indexer: %v", entityType, id, err)
		}
	}
}

// DeleteHackforgerIndexer pushes a delete request to the indexer queue.
func DeleteHackforgerIndexer(ctx context.Context, entityType string, ids ...int64) {
	for _, id := range ids {
		if err := pushQueue(ctx, &IndexerMetadata{ID: id, EntityType: entityType, IsDelete: true}); err != nil {
			log.Error("Unable to push delete %s/%d to HackForger indexer: %v", entityType, id, err)
		}
	}
}

// SearchHackforger searches for HackForger entities.
func SearchHackforger(ctx context.Context, opts *internal.SearchOptions) (*internal.SearchResult, error) {
	indexer := *globalIndexer.Load()
	return indexer.Search(ctx, opts)
}

// IsAvailable checks if the HackForger indexer is available.
func IsAvailable(ctx context.Context) bool {
	return (*globalIndexer.Load()).Ping(ctx) == nil
}

func pushQueue(ctx context.Context, data *IndexerMetadata) error {
	if hackforgerIndexerQueue == nil {
		log.Warn("Trying to push %+v to HackForger indexer queue, but the queue is not initialized", data)
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	err := hackforgerIndexerQueue.Push(data)
	if errors.Is(err, queue.ErrAlreadyInQueue) {
		return nil
	}
	return err
}
