<script>
import {POST} from '../../modules/fetch.js';
import {showInfoToast, showErrorToast} from '../../modules/toast.js';
import {formatDatetime} from '../../utils/time.js';
// NOTE: Forgejo's `showInfoToast` renders green with octicon-check (toast.js:11-15).
// It is the canonical success toast; no `showSuccessToast` exists. Do not swap.

export default {
  props: {
    hackathonSlug: {type: String, required: true},
    tracks: {type: Array, required: true},
    submissions: {type: Object, required: true},
    rubrics: {type: Object, required: true},
    messages: {type: Object, default: () => ({})},
  },
  data() {
    return {
      activeTrackId: this.tracks.length ? this.tracks[0].id : 0,
      scores: {},       // {subId: {criteriaId: {score, comment}}}
      saving: {},       // {subId: bool}
      saved: {},        // {subId: bool}
      errors: {},       // {subId: string}
      lastSavedAt: {},  // {subId: timestamp ms}
      expanded: {},     // {subId: bool} — collapsed summary unless true
      globalError: '',  // top-of-page error banner text
    };
  },
  created() {
    // Pre-populate ALL keys for Vue reactivity tracking.
    // Vue 3 data() objects need keys to exist at creation time for reactivity.
    for (const trackId in this.submissions) {
      const rubric = this.rubrics[trackId] || [];
      for (const sub of this.submissions[trackId]) {
        this.scores[sub.id] = {};
        this.saving[sub.id] = false;
        this.saved[sub.id] = false;
        this.errors[sub.id] = '';
        this.lastSavedAt[sub.id] = 0;
        this.expanded[sub.id] = false;
        for (const c of rubric) {
          this.scores[sub.id][c.criteria_id] = {score: 0, comment: ''};
        }
        // Overwrite with existing scores if any
        if (sub.existing_scores) {
          for (const s of sub.existing_scores) {
            this.scores[sub.id][s.criteria_id] = {score: s.score, comment: s.comment};
          }
          if (sub.existing_scores.length > 0) this.saved[sub.id] = true;
        }
      }
    }
  },
  computed: {
    activeSubmissions() {
      return this.submissions[this.activeTrackId] || [];
    },
    activeRubric() {
      return this.rubrics[this.activeTrackId] || [];
    },
    scoredCount() {
      return this.activeSubmissions.filter((s) => this.saved[s.id]).length;
    },
    progressPercent() {
      return this.activeSubmissions.length ? (this.scoredCount / this.activeSubmissions.length * 100) : 0;
    },
    nextUnscoredSubId() {
      const s = this.activeSubmissions.find((s) => !this.saved[s.id]);
      return s ? s.id : null;
    },
  },
  methods: {
    getScore(subId, cId) {
      return this.scores[subId]?.[cId]?.score ?? 0;
    },
    setScore(subId, cId, event) {
      if (!this.scores[subId]) this.scores[subId] = {};
      if (!this.scores[subId][cId]) this.scores[subId][cId] = {score: 0, comment: ''};
      const val = parseFloat(event.target.value);
      this.scores[subId][cId].score = Number.isNaN(val) ? 0 : val;
    },
    getComment(subId, cId) {
      return this.scores[subId]?.[cId]?.comment ?? '';
    },
    setComment(subId, cId, event) {
      if (!this.scores[subId]) this.scores[subId] = {};
      if (!this.scores[subId][cId]) this.scores[subId][cId] = {score: 0, comment: ''};
      this.scores[subId][cId].comment = event.target.value;
    },
    scrollToSubmission(subId) {
      const el = document.getElementById('submission-' + subId);
      if (el) el.scrollIntoView({behavior: 'smooth', block: 'center'});
    },
    formatTime(ms) {
      if (!ms) return '';
      // Forgejo locale-aware formatter; respects 12/24h preference.
      return formatDatetime(new Date(ms), {hour: 'numeric', minute: '2-digit'});
    },
    fillTemplate(tpl, ...args) {
      // Replace %s (or %d) placeholders in order — locale ini uses these.
      let i = 0;
      return (tpl || '').replace(/%[sd]/g, () => (i < args.length ? String(args[i++]) : ''));
    },
    async submitScores(submissionId) {
      this.saving[submissionId] = true;
      this.errors[submissionId] = '';
      this.globalError = '';

      if (this.activeRubric.length === 0) {
        this.globalError = this.messages.rubricNotConfigured;
        showErrorToast(this.globalError);
        this.saving[submissionId] = false;
        return;
      }

      const subScores = this.scores[submissionId] || {};
      const payload = this.activeRubric.map((c) => ({
        criteria_id: c.criteria_id,
        score: Number(subScores[c.criteria_id]?.score || 0),
        comment: subScores[c.criteria_id]?.comment || '',
      }));
      try {
        const resp = await POST(
          `/hackathon/${this.hackathonSlug}/judge/${submissionId}/scores`,
          {data: {scores: payload}},
        );
        if (resp.ok) {
          this.saved[submissionId] = true;
          this.lastSavedAt[submissionId] = Date.now();
          this.expanded[submissionId] = false;
          showInfoToast(this.messages.scoreSaved);
        } else {
          const data = await resp.json().catch(() => ({}));
          const msg = data.message || this.messages.errorGeneric;
          this.errors[submissionId] = msg;
          this.globalError = msg;
          showErrorToast(msg);
        }
      } catch (e) {
        this.errors[submissionId] = this.messages.errorGeneric;
        this.globalError = this.messages.errorGeneric;
        showErrorToast(this.messages.errorGeneric);
      }
      this.saving[submissionId] = false;
    },
  },
};
</script>

<template>
  <!-- Track tabs -->
  <div class="ui secondary pointing menu" v-if="tracks.length > 1">
    <a v-for="t in tracks" :key="t.id"
       :class="['item', {active: activeTrackId === t.id}]"
       @click="activeTrackId = t.id">
      {{ t.name }}
    </a>
  </div>

  <!-- Error banner (top) -->
  <div v-if="globalError" class="ui error message visible tw-mb-4" role="alert">
    {{ globalError }}
  </div>

  <!-- Sticky progress bar — top:0 because judge page has no secondary tabbar. -->
  <div class="hf-card" style="position: sticky; top: 0; z-index: 10; background: var(--color-box-body);">
    <div class="hf-card-body">
      <div class="tw-flex tw-justify-between tw-items-center tw-mb-2">
        <span>{{ fillTemplate(messages.progressLabel, scoredCount, activeSubmissions.length) }}</span>
        <button v-if="nextUnscoredSubId"
                class="hf-btn hf-btn-sm hf-btn-outline"
                @click="scrollToSubmission(nextUnscoredSubId)">
          {{ messages.nextUnscored }}
        </button>
      </div>
      <div class="ui indicating progress">
        <div class="bar" :style="{width: progressPercent + '%'}"></div>
      </div>
    </div>
  </div>

  <!-- Submission cards -->
  <div v-for="sub in activeSubmissions" :key="sub.id"
       :id="'submission-' + sub.id"
       class="hf-card">
    <div class="hf-card-head">
      <span>{{ sub.title }}</span>
      <span v-if="sub.user_name" class="tw-text-sm tw-text-gray tw-ml-2">
        by <a :href="'/' + sub.user_name">{{ sub.user_name }}</a>
      </span>
      <span v-if="saved[sub.id]" class="hf-badge hf-badge-green">Scored</span>
    </div>

    <!-- Collapsed summary when saved and not expanded. -->
    <div v-if="saved[sub.id] && !expanded[sub.id]" class="hf-card-body">
      <div class="tw-flex tw-items-center tw-gap-2 tw-mb-3">
        <svg class="svg octicon-check-circle-fill" width="18" height="18" aria-hidden="true"><use xlink:href="#octicon-check-circle-fill"></use></svg>
        <span>{{ fillTemplate(messages.scoreSavedAt, formatTime(lastSavedAt[sub.id]) || '—') }}</span>
      </div>
      <div class="tw-grid tw-gap-2" style="grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));">
        <div v-for="c in activeRubric" :key="c.criteria_id">
          <div class="tw-text-sm tw-text-gray">{{ c.name }}</div>
          <strong>{{ scores[sub.id] && scores[sub.id][c.criteria_id] ? scores[sub.id][c.criteria_id].score : 0 }} / {{ c.max_score }}</strong>
        </div>
      </div>
      <button class="hf-btn hf-btn-outline tw-mt-3" @click="expanded[sub.id] = true">
        {{ messages.updateScores }}
      </button>
    </div>

    <!-- Editable form (default when unsaved, or expanded for updates). -->
    <div v-else class="hf-card-body">
      <p v-if="sub.track_name" class="tw-text-sm tw-mb-1">
        <span class="hf-badge">{{ sub.track_name }}</span>
      </p>
      <p v-if="sub.repo_full_name" class="tw-text-sm tw-mb-1">
        <svg class="svg octicon-repo" width="16" height="16" aria-hidden="true"><use xlink:href="#octicon-repo"></use></svg>
        <a :href="'/' + sub.repo_full_name">{{ sub.repo_full_name }}</a>
      </p>
      <p v-if="sub.description" class="tw-text-sm tw-text-gray">{{ sub.description }}</p>
      <p v-if="sub.demo_url"><a :href="sub.demo_url" target="_blank">Demo</a></p>

      <!-- Rubric-not-configured warning. -->
      <div v-if="!activeRubric.length" class="ui warning message">
        {{ messages.rubricNotConfigured }}
      </div>

      <!-- Criteria form -->
      <div v-else class="ui form tw-mt-4">
        <div v-for="c in activeRubric" :key="c.criteria_id" class="field">
          <label>{{ c.name }} <span class="tw-text-sm tw-text-gray">(0-{{ c.max_score }}, weight: {{ c.weight }})</span></label>
          <p v-if="c.description" class="tw-text-sm tw-text-gray tw-mb-2">{{ c.description }}</p>
          <div class="two fields">
            <div class="field">
              <input type="number" :min="0" :max="c.max_score" step="0.5"
                     :value="getScore(sub.id, c.criteria_id)"
                     @input="setScore(sub.id, c.criteria_id, $event)"
                     @change="setScore(sub.id, c.criteria_id, $event)"
                     placeholder="Score">
            </div>
            <div class="field">
              <input type="text"
                     :value="getComment(sub.id, c.criteria_id)"
                     @input="setComment(sub.id, c.criteria_id, $event)"
                     @change="setComment(sub.id, c.criteria_id, $event)"
                     placeholder="Comment (optional)">
            </div>
          </div>
        </div>
        <button class="hf-btn hf-btn-primary" :class="{loading: saving[sub.id]}"
                :disabled="saving[sub.id] || !activeRubric.length"
                @click="submitScores(sub.id)">
          {{ saved[sub.id] ? messages.updateScores : messages.submitScores }}
        </button>
        <button v-if="saved[sub.id] && expanded[sub.id]"
                class="hf-btn hf-btn-outline tw-ml-2"
                type="button"
                @click="expanded[sub.id] = false">
          Cancel
        </button>
      </div>
    </div>
  </div>

  <!-- Empty state -->
  <div v-if="!activeSubmissions.length" class="hf-card">
    <div class="hf-empty">
      No submissions to judge in this track.
    </div>
  </div>
</template>
