<template>
  <v-dialog v-model="dialog" :max-width="960" scrollable persistent>
    <v-card v-if="dialog && device">
      <v-card-title class="d-flex align-center">
        <v-icon class="mr-2">mdi-remote-desktop</v-icon>
        {{ $t('deviceRemoteDesktop') }}
        &mdash; {{ device.hostname }} ({{ device.ip_address }})
        <v-spacer />
        <v-chip
          x-small
          dark
          class="mr-2"
          :color="sessionChipColor"
        >
          {{ sessionChipLabel }}
        </v-chip>
        <v-btn icon small @click="close">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </v-card-title>

      <v-card-text>
        <v-alert v-if="formError" type="error" dense class="mb-3">
          {{ formError }}
        </v-alert>
        <v-alert
          v-if="status === 'helper_missing'"
          type="warning"
          dense
          text
          class="mb-3"
        >
          {{ $t('deviceRemoteDesktopHelperFailed') }}
        </v-alert>

        <div class="caption grey--text mb-2">
          {{ $t('deviceRemoteDesktopLaunchSubtitle') }}
          <span v-if="device.rdp_port">
            · RDP :{{ device.rdp_port || 3389 }}
          </span>
          <span v-if="device.rdp_user"> · {{ device.rdp_user }}</span>
        </div>
        <div class="caption mb-3">
          {{ $t('deviceRemoteDesktopConnectHint') }}
        </div>

        <div class="d-flex align-center flex-wrap mb-4" style="gap: 8px;">
          <v-btn
            color="primary"
            depressed
            :loading="status === 'connecting'"
            :disabled="status === 'connecting' || status === 'stopping'"
            @click="connect"
          >
            <v-icon left>mdi-lan-connect</v-icon>
            {{ $t('deviceRemoteDesktopConnect') }}
          </v-btn>
          <v-btn
            color="error"
            outlined
            :loading="status === 'stopping'"
            :disabled="!canStop || status === 'connecting'"
            @click="stop"
          >
            <v-icon left>mdi-stop</v-icon>
            {{ $t('deviceRemoteDesktopStop') }}
          </v-btn>
          <span v-if="lastOperator" class="caption grey--text ml-2">
            {{ $t('deviceRemoteDesktopOperator') }}:
            <strong>{{ lastOperator }}</strong>
            <span v-if="lastLogId"> · #{{ lastLogId }}</span>
          </span>
        </div>

        <v-divider class="my-3" />

        <div class="d-flex align-center mb-2">
          <span class="subtitle-2">{{ $t('deviceRdpLaunchHistory') }}</span>
          <span class="caption grey--text ml-2">
            {{ $t('deviceAuditLogRetainHint') }}
          </span>
          <v-spacer />
          <v-btn icon small :loading="historyLoading" @click="loadHistory">
            <v-icon>mdi-refresh</v-icon>
          </v-btn>
        </div>

        <v-data-table
          :headers="historyHeaders"
          :items="historyLogs"
          item-key="id"
          dense
          hide-default-footer
          :items-per-page="10"
          :loading="historyLoading"
          class="rdp-history-table"
        >
          <template v-slot:item.created="{ item }">
            {{ formatTime(item.created) }}
          </template>
          <template v-slot:item.phase="{ item }">
            <v-chip x-small dark :color="rdpPhaseColor(item.phase)">
              {{ rdpPhaseLabel(item.phase) }}
            </v-chip>
          </template>
          <template v-slot:item.target="{ item }">
            <span class="caption">
              {{ item.host }}:{{ item.rdp_port }}
              <span v-if="item.rdp_user" class="grey--text">
                ({{ item.rdp_user }})
              </span>
            </span>
          </template>
          <template v-slot:item.mstsc_started_at="{ item }">
            {{ formatTime(item.mstsc_started_at) }}
          </template>
          <template v-slot:item.mstsc_exited_at="{ item }">
            {{ formatTime(item.mstsc_exited_at) }}
          </template>
        </v-data-table>
      </v-card-text>

      <v-card-actions>
        <v-spacer />
        <v-btn text @click="close">{{ $t('close') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
import axios from 'axios';
import { getErrorMessage } from '@/lib/error';

export default {
  props: {
    value: Boolean,
    projectId: { type: [Number, String], required: true },
    device: { type: Object, default: null },
  },

  data() {
    return {
      // idle | connecting | connected | stopping | helper_missing | error
      status: 'idle',
      formError: null,
      lastOperator: '',
      lastLogId: null,
      historyLogs: [],
      historyLoading: false,
      launchSeq: 0,
      pollTimer: null,
    };
  },

  computed: {
    dialog: {
      get() { return this.value; },
      set(v) { this.$emit('input', v); },
    },
    apiBase() {
      return `/api/project/${this.projectId}/devices/${this.device.id}`;
    },
    canStop() {
      if (this.status === 'connected' || this.status === 'stopping') {
        return true;
      }
      const latest = this.historyLogs[0];
      return Boolean(
        latest
        && latest.phase === 'mstsc_started'
        && !latest.mstsc_exited_at,
      );
    },
    sessionChipColor() {
      if (this.status === 'connected' || this.canStop) return 'success';
      if (this.status === 'connecting' || this.status === 'stopping') {
        return 'info';
      }
      if (this.status === 'helper_missing') return 'warning';
      if (this.status === 'error') return 'error';
      return 'grey';
    },
    sessionChipLabel() {
      if (this.status === 'connecting') {
        return this.$t('deviceRemoteDesktopStatusLaunching');
      }
      if (this.status === 'stopping') {
        return this.$t('deviceRemoteDesktopStatusStopping');
      }
      if (this.status === 'connected' || this.canStop) {
        return this.$t('deviceRemoteDesktopStatusConnected');
      }
      if (this.status === 'helper_missing') {
        return this.$t('deviceRemoteDesktopStatusHelperMissing');
      }
      if (this.status === 'error') {
        return this.$t('deviceRemoteDesktopStatusError');
      }
      return this.$t('deviceRemoteDesktopStatusIdle');
    },
    historyHeaders() {
      return [
        {
          text: this.$t('deviceRdpLaunchTime'),
          value: 'created',
          width: '150px',
        },
        {
          text: this.$t('deviceRdpLaunchUser'),
          value: 'username',
          width: '100px',
        },
        {
          text: this.$t('deviceRdpLaunchPhase'),
          value: 'phase',
          width: '120px',
        },
        {
          text: this.$t('deviceRdpLaunchTarget'),
          value: 'target',
        },
        {
          text: this.$t('deviceRdpLaunchClientIp'),
          value: 'client_ip',
          width: '110px',
        },
        {
          text: this.$t('deviceRdpLaunchMstscStarted'),
          value: 'mstsc_started_at',
          width: '150px',
        },
        {
          text: this.$t('deviceRdpLaunchMstscExited'),
          value: 'mstsc_exited_at',
          width: '150px',
        },
      ];
    },
  },

  watch: {
    async value(open) {
      if (open && this.device) {
        // Idle only — do not call connect() / openHelperURL on open.
        this.resetState();
        await this.loadHistory();
        this.syncStatusFromHistory();
        this.startPolling();
      } else {
        this.stopPolling();
      }
    },
  },

  beforeDestroy() {
    this.stopPolling();
  },

  methods: {
    resetState() {
      this.status = 'idle';
      this.formError = null;
      this.lastOperator = '';
      this.lastLogId = null;
      this.launchSeq += 1;
    },
    close() {
      this.stopPolling();
      this.dialog = false;
    },
    startPolling() {
      this.stopPolling();
      this.pollTimer = setInterval(() => {
        if (this.dialog) {
          this.loadHistory(true);
        }
      }, 4000);
    },
    stopPolling() {
      if (this.pollTimer) {
        clearInterval(this.pollTimer);
        this.pollTimer = null;
      }
    },
    formatTime(t) {
      if (!t) return '-';
      try {
        return new Date(t).toLocaleString();
      } catch (e) {
        return String(t);
      }
    },
    rdpPhaseLabel(phase) {
      if (phase === 'mstsc_exited') {
        return this.$t('deviceRdpLaunchPhaseExited');
      }
      if (phase === 'mstsc_started') {
        return this.$t('deviceRdpLaunchPhaseStarted');
      }
      if (phase === 'helper_fetched') {
        return this.$t('deviceRdpLaunchPhaseFetched');
      }
      return this.$t('deviceRdpLaunchPhaseRequested');
    },
    rdpPhaseColor(phase) {
      if (phase === 'mstsc_exited') return 'grey';
      if (phase === 'mstsc_started') return 'success';
      if (phase === 'helper_fetched') return 'primary';
      return 'info';
    },
    syncStatusFromHistory() {
      if (this.status === 'connecting' || this.status === 'stopping') {
        return;
      }
      const latest = this.historyLogs[0];
      if (
        latest
        && latest.phase === 'mstsc_started'
        && !latest.mstsc_exited_at
      ) {
        this.status = 'connected';
        this.lastOperator = latest.username || '';
        this.lastLogId = latest.id;
      } else if (this.status === 'connected') {
        this.status = 'idle';
      }
    },
    async loadHistory(silent) {
      if (!this.device?.id) return;
      if (!silent) this.historyLoading = true;
      try {
        const res = await axios.get(`${this.apiBase}/rdp/launch-logs`, {
          params: { limit: 10 },
        });
        this.historyLogs = (res.data && res.data.logs) || [];
        this.syncStatusFromHistory();
      } catch (e) {
        if (!silent) this.formError = getErrorMessage(e);
      } finally {
        if (!silent) this.historyLoading = false;
      }
    },
    openHelperURL(helperUrl) {
      let helperOpened = false;
      const onBlur = () => {
        helperOpened = true;
      };
      window.addEventListener('blur', onBlur);
      try {
        // Do not assign window.location — custom-protocol navigation can replace
        // the page with the browser interstitial
        // ("this site wants to open this application").
        // A transient <a> click keeps the RDP history dialog in place.
        const a = document.createElement('a');
        a.href = helperUrl;
        a.rel = 'noopener noreferrer';
        a.style.display = 'none';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
      } catch (_) {
        helperOpened = false;
      }
      return new Promise((resolve) => {
        setTimeout(() => {
          window.removeEventListener('blur', onBlur);
          resolve(!helperOpened && document.hasFocus());
        }, 1600);
      });
    },
    async connect() {
      if (!this.device?.id) return;
      this.launchSeq += 1;
      const seq = this.launchSeq;
      this.status = 'connecting';
      this.formError = null;
      try {
        const { data } = await axios.post(`${this.apiBase}/rdp/launch`);
        if (seq !== this.launchSeq) return;

        this.lastOperator = (data && data.username) || '';
        this.lastLogId = data && data.log_id;

        let helperUrl = (data && data.helper_url) || '';
        if (!helperUrl && data && data.token) {
          helperUrl = `semaphore-rdp://connect?token=${encodeURIComponent(data.token)}`;
        }
        if (!helperUrl) {
          throw new Error('empty helper_url');
        }
        const base = encodeURIComponent(window.location.origin);
        const sep = helperUrl.includes('?') ? '&' : '?';
        helperUrl = `${helperUrl}${sep}base=${base}`;

        const missing = await this.openHelperURL(helperUrl);
        if (seq !== this.launchSeq) return;
        if (missing) {
          this.status = 'helper_missing';
        } else {
          this.status = 'connected';
        }
        await this.loadHistory();
      } catch (e) {
        if (seq !== this.launchSeq) return;
        this.status = 'error';
        this.formError = getErrorMessage(e);
      }
    },
    async stop() {
      if (!this.device?.id) return;
      this.status = 'stopping';
      this.formError = null;
      const base = encodeURIComponent(window.location.origin);
      const helperUrl = `semaphore-rdp://stop?device_id=${encodeURIComponent(this.device.id)}&base=${base}`;
      const missing = await this.openHelperURL(helperUrl);
      if (missing) {
        this.status = 'helper_missing';
        this.formError = this.$t('deviceRemoteDesktopStopHelperMissing');
      } else {
        this.status = 'idle';
      }
      await this.loadHistory();
    },
  },
};
</script>

<style scoped>
.rdp-history-table >>> .caption {
  font-size: 12px;
  line-height: 1.35;
}
</style>
