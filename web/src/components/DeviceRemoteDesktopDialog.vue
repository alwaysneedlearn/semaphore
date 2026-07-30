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
        <div class="caption grey--text mb-2">
          {{ $t('deviceRemoteDesktopLaunchSubtitle') }}
          <span v-if="device.rdp_port">
            · RDP :{{ device.rdp_port || 3389 }}
          </span>
          <span v-if="device.rdp_user"> · {{ device.rdp_user }}</span>
        </div>

        <div class="d-flex align-center flex-wrap mb-4" style="gap: 8px;">
          <v-btn
            color="primary"
            depressed
            :loading="busy === 'connect'"
            :disabled="busy !== ''"
            @click="connect"
          >
            <v-icon left>mdi-lan-connect</v-icon>
            {{ $t('deviceRemoteDesktopConnect') }}
          </v-btn>
          <v-btn
            color="error"
            outlined
            :loading="busy === 'stop'"
            :disabled="!canStop || busy === 'connect'"
            @click="stop"
          >
            <v-icon left>mdi-stop</v-icon>
            {{ $t('deviceRemoteDesktopStop') }}
          </v-btn>
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

export default {
  props: {
    value: Boolean,
    projectId: { type: [Number, String], required: true },
    device: { type: Object, default: null },
  },

  data() {
    return {
      busy: '', // '' | connect | stop
      historyLogs: [],
      historyLoading: false,
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
    activeSession() {
      const latest = this.historyLogs[0];
      return Boolean(
        latest
        && latest.phase === 'mstsc_started'
        && !latest.mstsc_exited_at,
      );
    },
    canStop() {
      return this.activeSession || this.busy === 'stop';
    },
    sessionChipColor() {
      if (this.activeSession) return 'success';
      if (this.busy) return 'info';
      return 'grey';
    },
    sessionChipLabel() {
      if (this.busy === 'connect') {
        return this.$t('deviceRemoteDesktopStatusLaunching');
      }
      if (this.busy === 'stop') {
        return this.$t('deviceRemoteDesktopStatusStopping');
      }
      if (this.activeSession) {
        return this.$t('deviceRemoteDesktopStatusConnected');
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
        this.busy = '';
        await this.loadHistory();
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
    async loadHistory(silent) {
      if (!this.device?.id) return;
      if (!silent) this.historyLoading = true;
      try {
        const res = await axios.get(`${this.apiBase}/rdp/launch-logs`, {
          params: { limit: 10 },
        });
        this.historyLogs = (res.data && res.data.logs) || [];
      } catch (e) {
        // History refresh stays quiet — connection itself is Helper-driven.
      } finally {
        if (!silent) this.historyLoading = false;
      }
    },
    openHelperURL(helperUrl) {
      try {
        window.location.href = helperUrl;
      } catch (_) {
        // Ignore protocol handler errors; Helper absence is not prompted here.
      }
    },
    async connect() {
      if (!this.device?.id || this.busy) return;
      this.busy = 'connect';
      try {
        const { data } = await axios.post(`${this.apiBase}/rdp/launch`);
        let helperUrl = (data && data.helper_url) || '';
        if (!helperUrl && data && data.token) {
          helperUrl = `semaphore-rdp://connect?token=${encodeURIComponent(data.token)}`;
        }
        if (helperUrl) {
          const base = encodeURIComponent(window.location.origin);
          const sep = helperUrl.includes('?') ? '&' : '?';
          this.openHelperURL(`${helperUrl}${sep}base=${base}`);
        }
      } catch (_) {
        // No popup on connect failure; user can retry.
      } finally {
        this.busy = '';
        // History appears after Helper redeem / mstsc callbacks.
        setTimeout(() => this.loadHistory(true), 1500);
      }
    },
    async stop() {
      if (!this.device?.id || this.busy === 'connect') return;
      this.busy = 'stop';
      const base = encodeURIComponent(window.location.origin);
      this.openHelperURL(
        `semaphore-rdp://stop?device_id=${encodeURIComponent(this.device.id)}&base=${base}`,
      );
      this.busy = '';
      setTimeout(() => this.loadHistory(true), 1500);
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
