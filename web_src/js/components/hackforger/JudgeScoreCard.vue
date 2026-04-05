<script>
import {POST} from '../../modules/fetch.js';

export default {
  props: {
    hackathonSlug: {type: String, required: true},
    tracks: {type: Array, required: true},
    submissions: {type: Object, required: true},
    rubrics: {type: Object, required: true},
  },
  data() {
    return {
      activeTrackId: this.tracks.length ? this.tracks[0].id : 0,
      scores: {},    // {subId: {criteriaId: {score, comment}}}
      saving: {},    // {subId: bool}
      saved: {},     // {subId: bool}
      errors: {},    // {subId: string}
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
    async submitScores(submissionId) {
      this.saving[submissionId] = true;
      this.errors[submissionId] = '';
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
        } else {
          const data = await resp.json();
          this.errors[submissionId] = data.message || 'Error';
        }
      } catch {
        this.errors[submissionId] = 'Network error';
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

  <!-- Progress bar -->
  <div class="hf-card">
    <div class="hf-card-body">
      <div class="tw-flex tw-justify-between tw-mb-2">
        <span>{{ scoredCount }} / {{ activeSubmissions.length }} submissions scored</span>
      </div>
      <div class="ui indicating progress">
        <div class="bar" :style="{width: progressPercent + '%'}"></div>
      </div>
    </div>
  </div>

  <!-- Submission cards -->
  <div v-for="sub in activeSubmissions" :key="sub.id" class="hf-card">
    <div class="hf-card-head">
      <span>{{ sub.title }}</span>
      <span v-if="sub.user_name" class="tw-text-sm tw-text-gray tw-ml-2">
        by <a :href="'/' + sub.user_name">{{ sub.user_name }}</a>
      </span>
      <span v-if="saved[sub.id]" class="hf-badge hf-badge-green">Scored</span>
    </div>
    <div class="hf-card-body">
    <p v-if="sub.track_name" class="tw-text-sm tw-mb-1">
      <span class="hf-badge">{{ sub.track_name }}</span>
    </p>
    <p v-if="sub.repo_full_name" class="tw-text-sm tw-mb-1">
      <svg class="svg octicon-repo" width="16" height="16" aria-hidden="true"><use xlink:href="#octicon-repo"></use></svg>
      <a :href="'/' + sub.repo_full_name">{{ sub.repo_full_name }}</a>
    </p>
    <p v-if="sub.description" class="tw-text-sm tw-text-gray">{{ sub.description }}</p>
    <p v-if="sub.demo_url"><a :href="sub.demo_url" target="_blank">Demo</a></p>

    <!-- Criteria form -->
    <div class="ui form tw-mt-4">
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
              :disabled="saving[sub.id]"
              @click="submitScores(sub.id)">
        Submit Scores
      </button>
      <div v-if="errors[sub.id]" class="ui error message visible">{{ errors[sub.id] }}</div>
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
