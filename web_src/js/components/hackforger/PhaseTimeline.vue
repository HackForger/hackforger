<script>
import {GET, POST, PUT, DELETE} from '../../modules/fetch.js';
import {createSortable} from '../../modules/sortable.js';

export default {
  props: {
    hackathonSlug: {type: String, required: true},
    initialPhases: {type: Array, default: () => []},
    phaseTypes: {type: Array, default: () => []},
    isPublished: {type: Boolean, default: false},
  },
  data() {
    return {
      phases: [...this.initialPhases],
      statusCache: 0,
      loading: false,
      error: '',
      newPhase: {phaseTypeId: '', startTime: '', endTime: ''},
      editingPhaseId: null,
      editingName: '',
    };
  },
  computed: {
    sortedPhases() {
      return [...this.phases].sort((a, b) => a.sort_order - b.sort_order || a.start_time - b.start_time);
    },
    allTypesWithState() {
      return this.phaseTypes.map((pt) => {
        const alreadyAdded = pt.is_unique && this.phases.some((p) => p.phase_type_id === pt.id);
        return {
          ...pt,
          disabled: alreadyAdded,
          label: alreadyAdded
            ? `${pt.display_name} ${pt.already_added_tip || '(already added)'}`
            : pt.display_name,
        };
      });
    },
  },
  async mounted() {
    await this.$nextTick();
    const listEl = this.$refs.phaseList;
    if (listEl) {
      createSortable(listEl, {
        handle: '.drag-handle',
        filter: '.is-locked',
        onEnd: () => this.onDragEnd(),
      });
    }
  },
  methods: {
    phaseState(phase) {
      const now = Math.floor(Date.now() / 1000);
      if (phase.end_time < now) return 'locked';
      if (phase.start_time <= now && phase.end_time > now) return 'active';
      return 'future';
    },
    phaseTypeName(phase) {
      if (phase.custom_name) return phase.custom_name;
      const pt = this.phaseTypes.find((t) => t.id === phase.phase_type_id);
      return pt ? pt.display_name : `Phase #${phase.id}`;
    },
    startRename(phase) {
      this.editingPhaseId = phase.id;
      this.editingName = phase.custom_name || this.phaseTypeName(phase);
      this.$nextTick(() => {
        const input = this.$el.querySelector('input[type="text"]');
        if (input) input.focus();
      });
    },
    async saveRename(phase) {
      this.editingPhaseId = null;
      const pt = this.phaseTypes.find((t) => t.id === phase.phase_type_id);
      const customName = (pt && this.editingName === pt.display_name) ? '' : this.editingName;
      if (customName === (phase.custom_name || '')) return;
      this.loading = true;
      try {
        const resp = await PUT(`/hackathon/${this.hackathonSlug}/manage/phases/${phase.id}`, {
          data: {start_time: phase.start_time, end_time: phase.end_time, custom_name: customName},
        });
        if (resp.ok) {
          const data = await resp.json();
          this.phases = data.phases || [];
        }
      } finally {
        this.loading = false;
      }
    },
    cancelRename() {
      this.editingPhaseId = null;
    },
    toDateLocal(unix) {
      if (!unix) return '';
      const d = new Date(unix * 1000);
      const pad = (n) => String(n).padStart(2, '0');
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
    },
    fromDateLocal(str, isEnd) {
      if (!str) return 0;
      const dt = isEnd ? `${str}T23:59:00` : `${str}T00:00:00`;
      return Math.floor(new Date(dt).getTime() / 1000);
    },
    async addPhase() {
      const startTime = this.fromDateLocal(this.newPhase.startTime, false);
      const endTime = this.fromDateLocal(this.newPhase.endTime, true);
      if (!this.newPhase.phaseTypeId || !startTime || !endTime) {
        this.error = 'Please fill all fields';
        return;
      }
      if (endTime <= startTime) {
        this.error = 'End time must be after start time';
        return;
      }
      this.loading = true;
      this.error = '';
      try {
        const resp = await POST(`/hackathon/${this.hackathonSlug}/manage/phases`, {
          data: {
            phase_type_id: Number(this.newPhase.phaseTypeId),
            start_time: startTime,
            end_time: endTime,
            sort_order: this.phases.length + 1,
          },
        });
        if (!resp.ok) {
          const data = await resp.json();
          this.error = data.error || `HTTP ${resp.status}`;
          return;
        }
        const data = await resp.json();
        this.phases = data.phases || [];
        this.statusCache = data.status_cache;
        this.newPhase = {phaseTypeId: '', startTime: '', endTime: ''};
      } catch (e) {
        this.error = e.message;
      } finally {
        this.loading = false;
      }
    },
    async updatePhase(phase, field, event) {
      const val = this.fromDateLocal(event.target.value, field === 'end');
      if (!val) return;
      const startTime = field === 'start' ? val : phase.start_time;
      const endTime = field === 'end' ? val : phase.end_time;
      if (endTime <= startTime) {
        this.error = 'End time must be after start time';
        return;
      }
      this.loading = true;
      this.error = '';
      try {
        const resp = await PUT(`/hackathon/${this.hackathonSlug}/manage/phases/${phase.id}`, {
          data: {start_time: startTime, end_time: endTime},
        });
        if (!resp.ok) {
          const data = await resp.json();
          this.error = data.error || `HTTP ${resp.status}`;
          return;
        }
        const data = await resp.json();
        this.phases = data.phases || [];
        this.statusCache = data.status_cache;
      } catch (e) {
        this.error = e.message;
      } finally {
        this.loading = false;
      }
    },
    async onDragEnd() {
      // Read new order from DOM data attributes
      const items = this.$refs.phaseList.querySelectorAll('[data-phase-id]');
      const orders = Array.from(items).map((el, i) => ({
        id: Number(el.dataset.phaseId),
        sort_order: i + 1,
      }));
      this.loading = true;
      try {
        const resp = await POST(`/hackathon/${this.hackathonSlug}/manage/phases/reorder`, {
          data: {orders},
        });
        if (resp.ok) {
          const data = await resp.json();
          this.phases = data.phases || [];
        }
      } finally {
        this.loading = false;
      }
    },
    async deletePhase(phase) {
      this.loading = true;
      this.error = '';
      try {
        const resp = await DELETE(`/hackathon/${this.hackathonSlug}/manage/phases/${phase.id}`);
        if (!resp.ok) {
          const data = await resp.json();
          this.error = data.error || `HTTP ${resp.status}`;
          return;
        }
        const data = await resp.json();
        this.phases = data.phases || [];
        this.statusCache = data.status_cache;
      } catch (e) {
        this.error = e.message;
      } finally {
        this.loading = false;
      }
    },
  },
};
</script>

<template>
  <div>
    <div v-if="error" class="ui small negative message">{{ error }}</div>

    <!-- Phase list -->
    <div ref="phaseList">
    <div v-for="phase in sortedPhases" :key="phase.id"
         :data-phase-id="phase.id"
         class="tw-flex tw-items-center tw-gap-3 tw-py-2 tw-border-b"
         :class="{'tw-opacity-50': phaseState(phase) === 'locked', 'is-locked': phaseState(phase) === 'locked'}">
      <span v-if="phaseState(phase) !== 'locked'" class="drag-handle tw-cursor-grab tw-text-gray tw-select-none" style="font-size:18px">&#x2807;</span>
      <span v-else class="tw-inline-block" style="width:14px"></span>
      <span class="hf-badge"
            :class="{'hf-badge-neutral': phaseState(phase) === 'locked',
                     'hf-badge-green': phaseState(phase) === 'active',
                     'hf-badge-blue': phaseState(phase) === 'future'}">
        {{ phaseState(phase) }}
      </span>
      <template v-if="editingPhaseId === phase.id">
        <input type="text" v-model="editingName" class="tw-text-sm tw-min-w-24 tw-max-w-48"
               @blur="saveRename(phase)" @keyup.enter="saveRename(phase)" @keyup.escape="cancelRename"
               autofocus>
      </template>
      <strong v-else class="tw-min-w-24 tw-max-w-48 tw-truncate tw-flex tw-items-center tw-gap-1">
        {{ phaseTypeName(phase) }}
        <button v-if="phaseState(phase) !== 'locked'" class="tw-text-gray tw-cursor-pointer tw-border-0 tw-bg-transparent tw-p-0"
                @click="startRename(phase)" title="Rename">
          <svg width="14" height="14" viewBox="0 0 16 16"><path fill="currentColor" d="M11.013 1.427a1.75 1.75 0 012.474 0l1.086 1.086a1.75 1.75 0 010 2.474l-8.61 8.61c-.21.21-.47.364-.756.445l-3.251.93a.75.75 0 01-.927-.928l.929-3.25a1.75 1.75 0 01.445-.758l8.61-8.61zm1.414 1.06a.25.25 0 00-.354 0L3.463 11.1a.25.25 0 00-.064.108l-.563 1.97 1.971-.564a.25.25 0 00.108-.064l8.61-8.61a.25.25 0 000-.354l-1.086-1.086z"/></svg>
        </button>
      </strong>

      <input type="date"
             :value="toDateLocal(phase.start_time)"
             :disabled="phaseState(phase) !== 'future'"
             @change="updatePhase(phase, 'start', $event)"
             class="tw-text-sm">

      <span class="tw-text-gray">&rarr;</span>

      <input type="date"
             :value="toDateLocal(phase.end_time)"
             :disabled="phaseState(phase) === 'locked'"
             @change="updatePhase(phase, 'end', $event)"
             class="tw-text-sm">

      <button v-if="phaseState(phase) === 'future'"
              class="hf-btn hf-btn-danger"
              :disabled="loading"
              @click="deletePhase(phase)">
        &times;
      </button>
    </div>

    </div>

    <div v-if="!sortedPhases.length" class="tw-text-gray tw-py-2">
      No phases configured yet. Add phases below.
    </div>

    <!-- Add phase form -->
    <div class="tw-flex tw-items-center tw-gap-2 tw-mt-3">
      <select v-model="newPhase.phaseTypeId" class="tw-text-sm">
        <option value="" disabled>Select phase type...</option>
        <option v-for="pt in allTypesWithState" :key="pt.id"
                :value="pt.id" :disabled="pt.disabled">
          {{ pt.label }}
        </option>
      </select>
      <input type="date" v-model="newPhase.startTime" class="tw-text-sm">
      <input type="date" v-model="newPhase.endTime" class="tw-text-sm">
      <button class="hf-btn hf-btn-primary" :disabled="loading" @click="addPhase">
        + Add Phase
      </button>
    </div>
  </div>
</template>
