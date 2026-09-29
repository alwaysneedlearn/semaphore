<template>
  <div class="device-monitor pa-4">
    <div class="d-flex align-center mb-4 flex-wrap">
      <v-btn
        icon
        class="mr-2"
        :to="`/project/${projectId}/devices/list`"
        :title="$t('deviceTabList')"
      >
        <v-icon>mdi-arrow-left</v-icon>
      </v-btn>
      <div>
        <div class="text-h6">
          {{ $t('deviceMonitorTitle') }}
          <span v-if="device">— {{ device.hostname }} ({{ device.ip_address }})</span>
        </div>
        <div class="caption grey--text" v-if="connectionPreview">
          {{ connectionPreview.endpoint }}
        </div>
      </div>
      <v-spacer />
      <v-chip v-if="device" x-small :color="winrmStatusColor" dark class="mr-2">
        WinRM: {{ device.winrm_status || 'unknown' }}
      </v-chip>
      <v-btn
        icon
        small
        :loading="probing"
        class="mr-2"
        :title="$t('deviceProbe')"
        @click="probeDevice"
      >
        <v-icon>mdi-radar</v-icon>
      </v-btn>
    </div>

    <v-alert v-if="pageError" type="error" dense dismissible class="mb-4" @input="pageError = ''">
      {{ pageError }}
    </v-alert>

    <v-radio-group
      v-model="credentialMode"
      row
      dense
      class="mt-0 mb-2"
      @change="onCredentialModeChange"
    >
      <v-radio :label="$t('deviceWinrmCredentialWinrm')" value="winrm" />
      <v-radio
        :label="$t('deviceWinrmCredentialRdp')"
        value="rdp"
        :disabled="!rdpCredentialAvailable"
      />
    </v-radio-group>
    <v-checkbox
      v-model="forceOffline"
      dense
      hide-details
      class="mt-0 mb-4"
      :label="$t('deviceWinrmForceOffline')"
    />

    <v-card class="mb-4" outlined>
      <v-card-title class="subtitle-1 d-flex align-center">
        {{ $t('deviceMonitorUtilization') }}
        <v-spacer />
        <v-btn small depressed color="primary" :loading="metricsLoading" @click="loadMetrics">
          {{ $t('deviceMonitorRefreshMetrics') }}
        </v-btn>
      </v-card-title>
      <v-card-text>
        <div v-if="!metrics && !metricsLoading" class="caption grey--text">
          {{ $t('deviceMonitorMetricsHint') }}
        </div>
        <v-row v-else dense>
          <v-col cols="12" sm="4">
            <div class="caption grey--text">{{ $t('deviceMonitorCPU') }}</div>
            <div class="text-h5">{{ formatPercent(metrics && metrics.cpu_percent) }}</div>
          </v-col>
          <v-col cols="12" sm="4">
            <div class="caption grey--text">{{ $t('deviceMonitorMemory') }}</div>
            <div class="text-h5">
              {{ formatPercent(metrics && metrics.memory && metrics.memory.used_percent) }}
            </div>
            <div class="caption">
              {{ formatBytes(metrics && metrics.memory && metrics.memory.used_bytes) }}
              /
              {{ formatBytes(metrics && metrics.memory && metrics.memory.total_bytes) }}
            </div>
          </v-col>
          <v-col cols="12" sm="4">
            <div class="caption grey--text mb-1">{{ $t('deviceMonitorDisks') }}</div>
            <div v-if="!(metrics && metrics.disks && metrics.disks.length)" class="caption">—</div>
            <div v-for="d in (metrics && metrics.disks) || []" :key="d.name" class="mb-2">
              <div class="d-flex align-center justify-space-between">
                <strong>{{ d.name }}</strong>
                <span>{{ formatPercent(d.used_percent) }}</span>
              </div>
              <v-progress-linear
                :value="Number(d.used_percent) || 0"
                height="6"
                class="my-1"
                :color="diskBarColor(d.used_percent)"
              />
              <div class="caption grey--text">
                {{ formatBytes(d.used_bytes) }} / {{ formatBytes(d.total_bytes) }}
              </div>
            </div>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <v-card outlined>
      <v-card-title class="subtitle-1 d-flex align-center flex-wrap">
        {{ $t('deviceMonitorFiles') }}
        <v-spacer />
        <v-btn
          small
          text
          class="mr-2"
          :disabled="!parentPath && currentPath === ''"
          @click="goParent"
        >
          <v-icon left small>mdi-arrow-up</v-icon>
          {{ $t('deviceMonitorParent') }}
        </v-btn>
        <v-btn small depressed color="primary" :loading="fsLoading" @click="loadFS('refresh')">
          {{ $t('deviceMonitorRefreshFS') }}
        </v-btn>
      </v-card-title>
      <v-card-text>
        <div class="d-flex align-center flex-wrap mb-2 breadcrumb-path caption">
          <a
            href="#"
            class="text-decoration-none"
            :class="{ 'font-weight-bold': !currentPath }"
            @click.prevent="goRoots"
          >{{ $t('deviceMonitorBreadcrumbRoots') }}</a>
          <span
            v-for="(seg, i) in pathSegments"
            :key="'bc-' + i + '-' + seg.path"
            class="d-inline-flex align-center"
          >
            <v-icon x-small class="mx-1">mdi-chevron-right</v-icon>
            <a
              v-if="i < pathSegments.length - 1"
              href="#"
              class="text-decoration-none monospace-path"
              @click.prevent="goBreadcrumb(seg.path)"
            >{{ seg.label }}</a>
            <strong v-else class="monospace-path">{{ seg.label }}</strong>
          </span>
        </div>
        <div class="d-flex align-center flex-wrap mb-3">
          <v-text-field
            v-model="searchInput"
            dense
            outlined
            hide-details
            clearable
            class="mr-2 flex-grow-1"
            style="max-width: 360px"
            :label="$t('deviceMonitorSearch')"
            @keyup.enter="applySearch"
            @click:clear="clearSearch"
          />
          <v-btn
            small
            depressed
            color="primary"
            class="mr-2"
            :loading="fsLoading"
            @click="applySearch"
          >
            {{ $t('deviceMonitorSearchApply') }}
          </v-btn>
          <v-btn
            small
            text
            :disabled="!searchQuery && !searchInput"
            @click="clearSearch"
          >
            {{ $t('deviceMonitorSearchClear') }}
          </v-btn>
        </div>
        <v-data-table
          :headers="fsHeaders"
          :items="entries"
          :loading="fsLoading"
          :items-per-page="10"
          hide-default-footer
          dense
          class="elevation-0"
        >
          <template v-slot:item.name="{ item }">
            <a
              v-if="item.is_dir"
              href="#"
              class="text-decoration-none"
              @click.prevent="enterDir(item)"
            >
              <v-icon small class="mr-1">mdi-folder</v-icon>
              {{ item.name }}
              <v-chip v-if="item.is_hidden" x-small class="ml-1">hidden</v-chip>
            </a>
            <span v-else>
              <v-icon small class="mr-1">mdi-file</v-icon>
              {{ item.name }}
              <v-chip v-if="item.is_hidden" x-small class="ml-1">hidden</v-chip>
            </span>
          </template>
          <template v-slot:item.size_bytes="{ item }">
            {{ item.is_dir ? '—' : formatBytes(item.size_bytes) }}
          </template>
          <template v-slot:item.modified_at="{ item }">
            {{ formatTime(item.modified_at) }}
          </template>
          <template v-slot:item.actions="{ item }">
            <v-btn
              v-if="!item.is_dir"
              x-small
              text
              color="primary"
              :loading="downloadingPath === item.path"
              :disabled="!!downloadingPath"
              @click="downloadFile(item)"
            >
              {{ $t('deviceMonitorDownload') }}
            </v-btn>
          </template>
        </v-data-table>
        <div v-if="downloadingPath" class="mt-3">
          <div class="caption mb-1">
            {{ $t('deviceMonitorDownloadProgress', { percent: downloadPercentDisplay }) }}
          </div>
          <v-progress-linear
            :value="downloadPercent"
            :indeterminate="downloadPercent < 0"
            height="6"
            color="primary"
          />
        </div>
        <div class="d-flex align-center mt-3">
          <span class="caption grey--text">
            {{ $t('deviceMonitorPageInfo', { page, pageSize: 10 }) }}
          </span>
          <v-spacer />
          <v-btn small text :disabled="page <= 1 || fsLoading" @click="loadFS('prev')">
            {{ $t('deviceMonitorPrev') }}
          </v-btn>
          <v-btn small text :disabled="!hasNext || fsLoading" @click="loadFS('next')">
            {{ $t('deviceMonitorNext') }}
          </v-btn>
        </div>
        <div class="caption grey--text mt-2">
          {{ $t('deviceMonitorDownloadLimit') }}
        </div>
      </v-card-text>
    </v-card>
  </div>
</template>

<script>
import axios from 'axios';

export default {
  props: {
    projectId: [Number, String],
  },
  data() {
    return {
      device: null,
      credentialMode: 'winrm',
      forceOffline: true,
      connectionPreview: null,
      probing: false,
      pageError: '',
      metrics: null,
      metricsLoading: false,
      currentPath: '',
      parentPath: '',
      entries: [],
      hasNext: false,
      nextCursor: '',
      fsCursor: '',
      cursorStack: [],
      page: 1,
      searchInput: '',
      searchQuery: '',
      searchDebounceTimer: null,
      fsLoading: false,
      downloadingPath: '',
      downloadPercent: -1,
      fsHeaders: [
        {
          text: this.$t('deviceMonitorColName'),
          value: 'name',
          sortable: false,
        },
        {
          text: this.$t('deviceMonitorColSize'),
          value: 'size_bytes',
          sortable: false,
          width: '120px',
        },
        {
          text: this.$t('deviceMonitorColModified'),
          value: 'modified_at',
          sortable: false,
          width: '200px',
        },
        {
          text: '',
          value: 'actions',
          sortable: false,
          width: '100px',
        },
      ],
    };
  },
  computed: {
    deviceId() {
      return this.$route.params.deviceId;
    },
    apiBase() {
      return `/api/project/${this.projectId}/devices/${this.deviceId}`;
    },
    rdpCredentialAvailable() {
      return !!(this.device && this.device.rdp_user);
    },
    winrmStatusColor() {
      const s = this.device && this.device.winrm_status;
      if (s === 'online') return 'success';
      if (s === 'offline') return 'error';
      return 'grey';
    },
    pathSegments() {
      if (!this.currentPath) return [];
      const raw = String(this.currentPath).replace(/[/]+/g, '\\').replace(/\\+$/, '');
      const m = raw.match(/^([A-Za-z]:)(.*)$/);
      if (!m) return [];
      const drive = m[1].toUpperCase();
      const segs = [{ label: drive, path: `${drive}\\` }];
      const rest = (m[2] || '').replace(/^\\+/, '');
      if (!rest) return segs;
      let acc = drive;
      rest.split('\\').filter(Boolean).forEach((part) => {
        acc = `${acc}\\${part}`;
        segs.push({ label: part, path: acc });
      });
      return segs;
    },
    downloadPercentDisplay() {
      if (this.downloadPercent < 0) return '…';
      return String(this.downloadPercent);
    },
    canAutoQuery() {
      if (this.forceOffline) return true;
      return !!(this.device && this.device.winrm_status === 'online');
    },
  },
  watch: {
    searchInput() {
      if (this.searchDebounceTimer) {
        clearTimeout(this.searchDebounceTimer);
      }
      this.searchDebounceTimer = setTimeout(() => {
        this.applySearch();
      }, 300);
    },
  },
  beforeDestroy() {
    if (this.searchDebounceTimer) {
      clearTimeout(this.searchDebounceTimer);
    }
  },
  async created() {
    await this.loadDevice();
    await this.loadConnectionPreview();
    if (this.canAutoQuery) {
      await Promise.all([this.loadMetrics(), this.loadFS('first')]);
    } else {
      this.pageError = this.$t('deviceMonitorOfflineHint');
    }
  },
  methods: {
    diskBarColor(pct) {
      const n = Number(pct);
      if (Number.isNaN(n)) return 'primary';
      if (n >= 90) return 'error';
      if (n >= 75) return 'warning';
      return 'primary';
    },
    async loadDevice() {
      try {
        const { data } = await axios.get(this.apiBase);
        this.device = data;
      } catch (e) {
        this.pageError = (e.response && e.response.data && e.response.data.error)
          || e.message
          || this.$t('deviceMonitorLoadFailed');
      }
    },
    async loadConnectionPreview() {
      try {
        const { data } = await axios.get(`${this.apiBase}/winrm/connection-preview`, {
          params: { credential_mode: this.credentialMode },
        });
        this.connectionPreview = data;
      } catch (e) {
        this.connectionPreview = null;
      }
    },
    async onCredentialModeChange() {
      await this.loadConnectionPreview();
      if (this.canAutoQuery) {
        await Promise.all([this.loadMetrics(), this.loadFS('first')]);
      }
    },
    async probeDevice() {
      this.probing = true;
      try {
        await axios.post(`${this.apiBase}/probe`);
        await this.loadDevice();
        if (this.canAutoQuery) {
          await Promise.all([this.loadMetrics(), this.loadFS('first')]);
        }
      } catch (e) {
        this.pageError = (e.response && e.response.data && e.response.data.error) || e.message;
      } finally {
        this.probing = false;
      }
    },
    monitorParams(extra) {
      const params = {
        credential_mode: this.credentialMode,
        ...(extra || {}),
      };
      if (this.forceOffline) params.force_offline = '1';
      return params;
    },
    async loadMetrics() {
      this.metricsLoading = true;
      this.pageError = '';
      try {
        const { data } = await axios.get(`${this.apiBase}/monitor/metrics`, {
          params: this.monitorParams(),
        });
        this.metrics = data;
        if (!data.ok) {
          this.pageError = data.message || data.error || this.$t('deviceMonitorMetricsFailed');
        }
      } catch (e) {
        const body = e.response && e.response.data;
        this.pageError = (body && (body.message || body.error)) || e.message;
        if (body && body.cpu_percent !== undefined) this.metrics = body;
      } finally {
        this.metricsLoading = false;
      }
    },
    async loadFS(mode) {
      let cursor = '';
      if (mode === 'first') {
        this.cursorStack = [];
        this.page = 1;
        cursor = '';
      } else if (mode === 'refresh') {
        cursor = this.fsCursor || '';
      } else if (mode === 'next') {
        if (!this.hasNext || !this.nextCursor) return;
        this.cursorStack.push(this.fsCursor || '');
        cursor = this.nextCursor;
        this.page += 1;
      } else if (mode === 'prev') {
        if (this.page <= 1) return;
        cursor = this.cursorStack.length ? this.cursorStack.pop() : '';
        this.page = Math.max(1, this.page - 1);
      } else {
        cursor = '';
        this.cursorStack = [];
        this.page = 1;
      }

      this.fsLoading = true;
      this.pageError = '';
      try {
        const params = this.monitorParams({
          path: this.currentPath,
        });
        if (cursor) params.cursor = cursor;
        if (this.searchQuery) params.q = this.searchQuery;
        const { data } = await axios.get(`${this.apiBase}/monitor/fs`, { params });
        if (!data.ok) {
          this.pageError = data.message || data.error || this.$t('deviceMonitorFSFailed');
          this.entries = [];
          this.hasNext = false;
          this.nextCursor = '';
          return;
        }
        this.entries = data.entries || [];
        this.hasNext = !!data.has_next;
        this.nextCursor = data.next_cursor || '';
        this.fsCursor = cursor;
        this.currentPath = data.path || '';
        this.parentPath = this.computeParent(this.currentPath);
      } catch (e) {
        const body = e.response && e.response.data;
        this.pageError = (body && (body.message || body.error)) || e.message;
        this.entries = [];
        this.hasNext = false;
        this.nextCursor = '';
      } finally {
        this.fsLoading = false;
      }
    },
    applySearch() {
      const next = (this.searchInput || '').trim();
      if (next === this.searchQuery && this.page === 1 && this.fsCursor === '') {
        return;
      }
      this.searchQuery = next;
      this.loadFS('first');
    },
    clearSearch() {
      if (this.searchDebounceTimer) {
        clearTimeout(this.searchDebounceTimer);
        this.searchDebounceTimer = null;
      }
      this.searchInput = '';
      this.searchQuery = '';
      this.loadFS('first');
    },
    resetSearchAndLoad(path) {
      if (this.searchDebounceTimer) {
        clearTimeout(this.searchDebounceTimer);
        this.searchDebounceTimer = null;
      }
      this.currentPath = path || '';
      this.searchInput = '';
      this.searchQuery = '';
      this.loadFS('first');
    },
    computeParent(p) {
      if (!p) return '';
      const norm = p.replace(/\\+$/, '');
      const idx = norm.lastIndexOf('\\');
      if (idx <= 2) {
        // C:\Users -> C:\
        if (/^[A-Za-z]:$/i.test(norm) || /^[A-Za-z]:\\?$/i.test(p)) return '';
        return `${norm.slice(0, 2)}\\`;
      }
      return norm.slice(0, idx);
    },
    enterDir(item) {
      this.resetSearchAndLoad(item.path);
    },
    goParent() {
      this.resetSearchAndLoad(this.parentPath || '');
    },
    goRoots() {
      this.resetSearchAndLoad('');
    },
    goBreadcrumb(path) {
      this.resetSearchAndLoad(path || '');
    },
    async downloadFile(item) {
      this.downloadingPath = item.path;
      this.downloadPercent = item.size_bytes > 0 ? 0 : -1;
      this.pageError = '';
      const knownTotal = Number(item.size_bytes) || 0;
      try {
        const res = await axios.get(`${this.apiBase}/monitor/fs/download`, {
          params: this.monitorParams({ path: item.path }),
          responseType: 'blob',
          onDownloadProgress: (ev) => {
            const total = ev.total || knownTotal;
            if (total > 0) {
              this.downloadPercent = Math.min(100, Math.round((ev.loaded / total) * 100));
            } else if (ev.loaded > 0) {
              this.downloadPercent = -1;
            }
          },
        });
        const ct = res.headers['content-type'] || '';
        if (ct.includes('application/json')) {
          const text = await res.data.text();
          const body = JSON.parse(text);
          throw new Error(body.message || body.error || this.$t('deviceMonitorDownloadFailed'));
        }
        this.downloadPercent = 100;
        const blob = new Blob([res.data]);
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = item.name || 'download.bin';
        document.body.appendChild(a);
        a.click();
        a.remove();
        window.URL.revokeObjectURL(url);
      } catch (e) {
        let msg = e.message;
        if (e.response && e.response.data) {
          try {
            const text = await e.response.data.text();
            const body = JSON.parse(text);
            msg = body.message || body.error || msg;
          } catch (_) {
            /* keep msg */
          }
        }
        this.pageError = msg || this.$t('deviceMonitorDownloadFailed');
      } finally {
        this.downloadingPath = '';
        this.downloadPercent = -1;
      }
    },
    formatBytes(n) {
      if (n === null || n === undefined || Number.isNaN(Number(n))) return '—';
      let v = Number(n);
      const units = ['B', 'KB', 'MB', 'GB', 'TB'];
      let i = 0;
      while (v >= 1024 && i < units.length - 1) {
        v /= 1024;
        i += 1;
      }
      return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
    },
    formatPercent(n) {
      if (n === null || n === undefined || Number.isNaN(Number(n))) return '—';
      return `${Number(n).toFixed(1)}%`;
    },
    formatTime(v) {
      if (!v) return '—';
      const d = new Date(v);
      if (Number.isNaN(d.getTime())) return v;
      return d.toLocaleString();
    },
  },
};
</script>

<style scoped>
.monospace-path {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  word-break: break-all;
}
</style>
