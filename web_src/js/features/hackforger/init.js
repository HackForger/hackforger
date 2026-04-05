// HackForger frontend module entry point.
// Lazy-loads Vue components when their mount elements are present.

import {initSearchModal} from './search-modal.js';
import {initHackathonEditor} from './hackathon-editor.js';

export function initHackforger() {
  initSearchModal();
  initHackathonEditor();
  // BountyPanel
  const bountyEl = document.getElementById('hackforger-bounty-panel');
  if (bountyEl) {
    (async () => {
      const {default: BountyPanel} = await import(
        /* webpackChunkName: "hackforger-bounty" */
        '../../components/hackforger/BountyPanel.vue'
      );
      const {createApp} = await import('vue');
      createApp(BountyPanel, {
        bountyId: bountyEl.getAttribute('data-bounty-id'),
        repoOwner: bountyEl.getAttribute('data-repo-owner'),
        repoName: bountyEl.getAttribute('data-repo-name'),
        status: Number(bountyEl.getAttribute('data-status')),
        mode: Number(bountyEl.getAttribute('data-mode')),
        isPublisher: bountyEl.getAttribute('data-is-publisher') === 'true',
        claimerId: Number(bountyEl.getAttribute('data-claimer-id')),
      }).mount(bountyEl);
    })();
  }

  // JudgeScoreCard
  const judgeEl = document.getElementById('hackforger-judge-scorecard');
  if (judgeEl) {
    (async () => {
      const {default: JudgeScoreCard} = await import(
        /* webpackChunkName: "hackforger-judge" */
        '../../components/hackforger/JudgeScoreCard.vue'
      );
      const {createApp} = await import('vue');
      createApp(JudgeScoreCard, {
        hackathonSlug: judgeEl.dataset.hackathonSlug,
        tracks: JSON.parse(judgeEl.dataset.tracks || '[]'),
        submissions: JSON.parse(judgeEl.dataset.submissions || '{}'),
        rubrics: JSON.parse(judgeEl.dataset.rubrics || '{}'),
      }).mount(judgeEl);
    })();
  }

  // PhaseTimeline
  const phaseEl = document.getElementById('hackforger-phase-timeline');
  if (phaseEl) {
    (async () => {
      const {default: PhaseTimeline} = await import(
        /* webpackChunkName: "hackforger-phase" */
        '../../components/hackforger/PhaseTimeline.vue'
      );
      const {createApp} = await import('vue');
      createApp(PhaseTimeline, {
        hackathonSlug: phaseEl.dataset.hackathonSlug,
        initialPhases: JSON.parse(phaseEl.dataset.phases || '[]'),
        phaseTypes: JSON.parse(phaseEl.dataset.phaseTypes || '[]'),
        isPublished: phaseEl.dataset.isPublished === 'true',
      }).mount(phaseEl);
    })();
  }
}
