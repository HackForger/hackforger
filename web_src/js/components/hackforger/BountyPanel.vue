<script>
import {GET, POST, PUT, DELETE} from '../../modules/fetch.js';

const STATUS_OPEN = 0;
const STATUS_CLAIMED = 1;
const STATUS_IN_REVIEW = 2;
const STATUS_COMPLETED = 3;

const MODE_EXCLUSIVE = 0;
const MODE_COMPETITIVE = 1;

export default {
  props: {
    bountyId: {type: String, required: true},
    repoOwner: {type: String, required: true},
    repoName: {type: String, required: true},
    status: {type: Number, required: true},
    mode: {type: Number, required: true},
    isPublisher: {type: Boolean, default: false},
    claimerId: {type: Number, default: 0},
  },
  data() {
    return {
      currentStatus: this.status,
      applications: [],
      showApplyForm: false,
      applyMessage: '',
      winners: [],
      showWinnersForm: false,
      winnerEntries: [{user_id: '', rank: 1}],
      loading: false,
      error: '',
    };
  },
  computed: {
    apiBase() {
      return `/${this.repoOwner}/${this.repoName}/bounties/${this.bountyId}`;
    },
    canApply() {
      return !this.isPublisher && this.currentStatus === STATUS_OPEN;
    },
    canComplete() {
      return this.isPublisher && this.currentStatus === STATUS_IN_REVIEW;
    },
    canRejectDelivery() {
      return this.isPublisher && this.currentStatus === STATUS_IN_REVIEW && this.mode === MODE_EXCLUSIVE;
    },
    canMarkPaid() {
      return this.isPublisher && this.currentStatus === STATUS_COMPLETED;
    },
    canCancel() {
      return this.isPublisher && (this.currentStatus === STATUS_OPEN || this.currentStatus === STATUS_CLAIMED);
    },
    canStartReview() {
      return this.isPublisher && this.currentStatus === STATUS_OPEN && this.mode === MODE_COMPETITIVE;
    },
    canSelectWinners() {
      return this.isPublisher && this.currentStatus === STATUS_IN_REVIEW && this.mode === MODE_COMPETITIVE;
    },
  },
  mounted() {
    if (this.isPublisher) {
      this.loadApplications();
    }
    if (this.mode === MODE_COMPETITIVE && this.currentStatus >= STATUS_COMPLETED) {
      this.loadWinners();
    }
  },
  methods: {
    async apiCall(path, method = 'POST', body = null) {
      this.loading = true;
      this.error = '';
      try {
        const opts = body ? {data: body} : {};
        let resp;
        switch (method) {
          case 'GET': resp = await GET(this.apiBase + path, opts); break;
          case 'POST': resp = await POST(this.apiBase + path, opts); break;
          case 'PUT': resp = await PUT(this.apiBase + path, opts); break;
          case 'DELETE': resp = await DELETE(this.apiBase + path, opts); break;
          default: resp = await POST(this.apiBase + path, opts);
        }
        if (!resp.ok) {
          const data = await resp.json().catch(() => ({}));
          throw new Error(data.message || `HTTP ${resp.status}`);
        }
        return resp.status === 204 ? null : await resp.json();
      } catch (e) {
        this.error = e.message;
        return null;
      } finally {
        this.loading = false;
      }
    },
    async loadApplications() {
      const data = await this.apiCall('/applications', 'GET');
      if (data) this.applications = data;
      // Clear error from initial load — auth errors are expected for non-API sessions
      this.error = '';
    },
    async submitApplication() {
      const result = await this.apiCall('/apply', 'POST', {message: this.applyMessage});
      if (result !== null) {
        this.showApplyForm = false;
        this.applyMessage = '';
      }
    },
    async reviewApplication(appId, action) {
      await this.apiCall(`/applications/${appId}`, 'POST', {action});
      this.loadApplications();
      if (action === 'accept' && this.mode === MODE_EXCLUSIVE) {
        this.currentStatus = STATUS_CLAIMED;
      }
    },
    async completeBounty() {
      const result = await this.apiCall('/complete');
      if (result !== null) this.currentStatus = STATUS_COMPLETED;
    },
    async rejectDelivery() {
      const result = await this.apiCall('/reject-delivery');
      if (result !== null) this.currentStatus = STATUS_CLAIMED;
    },
    async markPaid() {
      const result = await this.apiCall('/pay');
      if (result !== null) this.currentStatus = STATUS_COMPLETED + 1; // Paid
    },
    async cancelBounty() {
      const result = await this.apiCall('/cancel');
      if (result !== null) this.currentStatus = 6; // Cancelled
    },
    async startReview() {
      const result = await this.apiCall('/review');
      if (result !== null) this.currentStatus = STATUS_IN_REVIEW;
    },
    addWinnerEntry() {
      this.winnerEntries.push({user_id: '', rank: this.winnerEntries.length + 1});
    },
    removeWinnerEntry(index) {
      this.winnerEntries.splice(index, 1);
    },
    async submitWinners() {
      const winners = this.winnerEntries
        .filter((e) => e.user_id)
        .map((e) => ({user_id: Number(e.user_id), rank: e.rank}));
      if (winners.length === 0) return;
      const result = await this.apiCall('/winners', 'POST', {winners});
      if (result !== null) {
        this.currentStatus = STATUS_COMPLETED;
        this.showWinnersForm = false;
        this.loadWinners();
      }
    },
    async loadWinners() {
      const data = await this.apiCall('/winners', 'GET');
      if (data) this.winners = data;
      this.error = '';
    },
  },
};
</script>

<template>
  <div class="tw-mt-2">
    <div v-if="error" class="ui small negative message">{{ error }}</div>

    <!-- Apply button for non-publishers on Open bounties -->
    <div v-if="canApply">
      <button v-if="!showApplyForm" class="hf-btn hf-btn-primary tw-w-full" @click="showApplyForm = true">
        Apply for this Bounty
      </button>
      <div v-else class="ui form tw-mt-2">
        <div class="field">
          <textarea v-model="applyMessage" rows="3" placeholder="Why should you be assigned?"></textarea>
        </div>
        <button class="hf-btn hf-btn-primary" :disabled="loading" @click="submitApplication">Submit</button>
        <button class="hf-btn hf-btn-secondary" @click="showApplyForm = false">Cancel</button>
      </div>
    </div>

    <!-- Applications list for publisher -->
    <div v-if="isPublisher && applications.length > 0" class="tw-mt-2">
      <strong>Applications:</strong>
      <div v-for="app in applications" :key="app.id" class="tw-flex tw-items-center tw-justify-between tw-py-1">
        <span>{{ app.username || `User #${app.user_id}` }}: {{ app.message }}</span>
        <span v-if="app.status === 0">
          <button class="hf-btn hf-btn-primary" @click="reviewApplication(app.id, 'accept')">Accept</button>
          <button class="hf-btn hf-btn-danger" @click="reviewApplication(app.id, 'reject')">Reject</button>
        </span>
        <span v-else class="hf-badge">{{ app.status === 1 ? 'Accepted' : 'Rejected' }}</span>
      </div>
    </div>

    <!-- Winners display (competitive, after completion) -->
    <div v-if="winners.length > 0" class="tw-mt-2">
      <strong>Winners:</strong>
      <div v-for="w in winners" :key="w.ID" class="tw-flex tw-items-center tw-justify-between tw-py-1">
        <span>{{ w.Username || `User #${w.UserID}` }}</span>
        <span class="hf-badge">#{{ w.Rank }}</span>
      </div>
    </div>

    <!-- Select Winners form (competitive, publisher, in review) -->
    <div v-if="canSelectWinners" class="tw-mt-2">
      <button v-if="!showWinnersForm" class="hf-btn hf-btn-primary tw-w-full" @click="showWinnersForm = true">
        Select Winners
      </button>
      <div v-else class="ui form tw-mt-2">
        <div v-for="(entry, idx) in winnerEntries" :key="idx" class="tw-flex tw-gap-2 tw-items-center tw-mb-1">
          <input v-model="entry.user_id" type="number" placeholder="User ID" class="ui mini input" style="width: 100px;">
          <span class="tw-text-sm">Rank #{{ entry.rank }}</span>
          <button v-if="winnerEntries.length > 1" class="hf-btn hf-btn-secondary" @click="removeWinnerEntry(idx)">
            &times;
          </button>
        </div>
        <div class="tw-flex tw-gap-1 tw-mt-1">
          <button class="hf-btn hf-btn-secondary" @click="addWinnerEntry">+ Add</button>
          <button class="hf-btn hf-btn-primary" :disabled="loading" @click="submitWinners">Submit Winners</button>
          <button class="hf-btn hf-btn-secondary" @click="showWinnersForm = false">Cancel</button>
        </div>
      </div>
    </div>

    <!-- Publisher actions -->
    <div v-if="isPublisher" class="tw-mt-2 tw-flex tw-flex-col tw-gap-1">
      <button v-if="canStartReview" class="hf-btn hf-btn-primary tw-w-full" :disabled="loading" @click="startReview">
        Start Review
      </button>
      <button v-if="canComplete" class="hf-btn hf-btn-primary tw-w-full" :disabled="loading" @click="completeBounty">
        Complete Bounty
      </button>
      <button v-if="canRejectDelivery" class="hf-btn hf-btn-secondary tw-w-full" :disabled="loading" @click="rejectDelivery">
        Reject Delivery
      </button>
      <button v-if="canMarkPaid" class="hf-btn hf-btn-primary tw-w-full" :disabled="loading" @click="markPaid">
        Mark as Paid
      </button>
      <button v-if="canCancel" class="hf-btn hf-btn-danger tw-w-full" :disabled="loading" @click="cancelBounty">
        Cancel Bounty
      </button>
    </div>
  </div>
</template>
