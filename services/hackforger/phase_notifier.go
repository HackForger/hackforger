// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"
	"sync"
	"time"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/log"

	"github.com/go-co-op/gocron"
)

var (
	phaseScheduler     *gocron.Scheduler
	phaseSchedulerOnce sync.Once
)

// SetPhaseScheduler injects the gocron scheduler from the cron package.
// Called during server startup to avoid import cycles.
func SetPhaseScheduler(s *gocron.Scheduler) {
	phaseScheduler = s
}

// SchedulePhaseJobs registers gocron one-shot jobs for a phase's start and end times.
func SchedulePhaseJobs(phase *hackforger_model.Phase) {
	if phaseScheduler == nil {
		return
	}
	now := time.Now().Unix()

	if phase.StartTime > now {
		startTag := fmt.Sprintf("phase-%d-start", phase.ID)
		startTime := time.Unix(phase.StartTime, 0)
		_, err := phaseScheduler.Every(1).Second().LimitRunsTo(1).StartAt(startTime).Tag(startTag).Do(onPhaseEvent, phase.ActivityKind, phase.ActivityID, phase.ID, "start")
		if err != nil {
			log.Error("SchedulePhaseJobs: failed to schedule start for phase %d: %v", phase.ID, err)
		}
	}

	if phase.EndTime > now {
		endTag := fmt.Sprintf("phase-%d-end", phase.ID)
		endTime := time.Unix(phase.EndTime, 0)
		_, err := phaseScheduler.Every(1).Second().LimitRunsTo(1).StartAt(endTime).Tag(endTag).Do(onPhaseEvent, phase.ActivityKind, phase.ActivityID, phase.ID, "end")
		if err != nil {
			log.Error("SchedulePhaseJobs: failed to schedule end for phase %d: %v", phase.ID, err)
		}
	}
}

// UnschedulePhaseJobs removes gocron jobs for a phase.
func UnschedulePhaseJobs(phaseID int64) {
	if phaseScheduler == nil {
		return
	}
	_ = phaseScheduler.RemoveByTag(fmt.Sprintf("phase-%d-start", phaseID))
	_ = phaseScheduler.RemoveByTag(fmt.Sprintf("phase-%d-end", phaseID))
}

// RebuildPhaseSchedule scans all future phases from DB and registers gocron jobs.
// Called at server startup.
func RebuildPhaseSchedule(ctx context.Context) error {
	now := time.Now().Unix()
	phases, err := hackforger_model.GetFuturePhases(ctx, now)
	if err != nil {
		return fmt.Errorf("RebuildPhaseSchedule: %w", err)
	}
	for _, phase := range phases {
		SchedulePhaseJobs(phase)
	}
	log.Info("RebuildPhaseSchedule: registered jobs for %d future phases", len(phases))
	return nil
}

// CheckHackathonTransitions is the fallback cron sweep that syncs all published hackathons.
// Catches any transitions missed by the precise gocron jobs.
func CheckHackathonTransitions(ctx context.Context) error {
	// On first run, also rebuild the phase schedule
	phaseSchedulerOnce.Do(func() {
		if err := RebuildPhaseSchedule(ctx); err != nil {
			log.Error("CheckHackathonTransitions: RebuildPhaseSchedule: %v", err)
		}
	})

	hackathons, err := hackforger_model.ListPublishedHackathons(ctx)
	if err != nil {
		return err
	}
	for _, h := range hackathons {
		if syncErr := SyncStatusCache(ctx, h.ID, 0); syncErr != nil {
			log.Error("CheckHackathonTransitions: hackathon %d: %v", h.ID, syncErr)
		}
	}
	return nil
}

// onPhaseEvent is the callback fired by gocron when a phase starts or ends.
func onPhaseEvent(activityKind string, activityID, phaseID int64, event string) {
	ctx := context.Background()
	log.Info("PhaseNotifier: phase %d %s (activity: %s/%d)", phaseID, event, activityKind, activityID)

	if activityKind == "hackathon" {
		if err := SyncStatusCache(ctx, activityID, 0); err != nil {
			log.Error("PhaseNotifier: SyncStatusCache failed for hackathon %d: %v", activityID, err)
		}
	}
}
