<template>
  <v-dialog v-model="dialog" :max-width="1100" scrollable persistent>
    <v-card v-if="dialog && device">
      <v-card-title class="d-flex align-center">
        <v-icon class="mr-2">mdi-console</v-icon>
        {{ $t('deviceWinrmConsole') }} &mdash; {{ device.hostname }} ({{ device.ip_address }})
        <v-spacer />
        <v-chip x-small :color="winrmStatusColor" dark class="mr-2">
          WinRM: {{ device.winrm_status || 'unknown' }}
        </v-chip>
        <v-btn icon small :loading="probing" @click="probeDevice" :title="$t('deviceProbe')">
          <v-icon>mdi-radar</v-icon>
        </v-btn>
      </v-card-title>

      <v-card-text>
        <v-alert v-if="formError" type="error" dense>{{ formError }}</v-alert>

        <div class="caption mb-2" v-if="connectionPreview">
          {{ connectionPreview.endpoint }}
        </div>

        <v-radio-group
          v-model="credentialMode"
          row
          dense
          class="mt-0"
          @change="loadConnectionPreview"
        >
          <v-radio :label="$t('deviceWinrmCredentialWinrm')" value="winrm" />
          <v-radio
            :label="$t('deviceWinrmCredentialRdp')"
            value="rdp"
            :disabled="!rdpCredentialAvailable"
          />
        </v-radio-group>
        <div v-if="!rdpCredentialAvailable" class="caption grey--text mb-2">
          {{ $t('deviceWinrmRdpCredentialDisabled') }}
        </div>

        <div class="mb-2">
          <span class="subtitle-2 mr-2">{{ $t('deviceWinrmExamples') }}</span>
          <v-chip
            v-for="group in exampleGroups"
            :key="group.key"
            small
            class="mr-1 mb-1"
            :color="group.key === 'top5' ? 'primary' : undefined"
            :outlined="group.key === 'top5'"
            @click="showExampleMenu(group)"
          >
            {{ $t(group.labelKey) }}
          </v-chip>
          <div class="caption grey--text mt-1">
            {{ $t('deviceWinrmExamplesTop5Hint') }}
          </div>
          <v-menu v-model="exampleMenu" offset-y :close-on-content-click="true">
            <template v-slot:activator="{ on, attrs }">
              <span v-bind="attrs" v-on="on" ref="exampleMenuActivator" />
            </template>
            <v-list dense>
              <v-list-item
                v-for="(cmd, idx) in exampleMenuCommands"
                :key="idx"
                @click="command = cmd"
              >
                <v-list-item-title class="monospace-caption winrm-console-font">
                  {{ truncate(cmd, 80) }}
                </v-list-item-title>
              </v-list-item>
            </v-list>
          </v-menu>
        </div>

        <v-textarea
          v-model="command"
          :label="$t('deviceWinrmCommand')"
          rows="4"
          auto-grow
          outlined
          dense
          class="monospace-field winrm-console-font"
          :disabled="executing"
        />

        <div class="d-flex align-center mb-4">
          <v-btn
            color="primary"
            depressed
            :loading="executing"
            :disabled="!commandTrimmed"
            @click="askExecute"
          >
            {{ $t('deviceWinrmExecute') }}
          </v-btn>
          <v-btn text class="ml-2" :disabled="executing" @click="command = ''">
            {{ $t('deviceWinrmClearCommand') }}
          </v-btn>
          <v-checkbox
            v-model="forceOffline"
            dense
            hide-details
            class="ml-4 mt-0"
            :label="$t('deviceWinrmForceOffline')"
          />
        </div>

        <div v-if="lastOutput !== null" class="mb-4">
          <div class="subtitle-2 mb-1">{{ $t('deviceWinrmOutput') }}</div>
          <v-alert
            v-if="lastOutput.message && !lastOutput.ok"
            type="warning"
            dense
            text
            class="mb-2"
          >
            {{ lastOutput.message }}
          </v-alert>
          <pre class="winrm-output winrm-console-font">{{ formatOutput(lastOutput) }}</pre>
          <div class="caption" v-if="lastOutput.duration_ms != null">
            exit={{ lastOutput.exit_code }} duration={{ lastOutput.duration_ms }}ms
          </div>
        </div>

        <v-divider class="my-3" />

        <div class="d-flex align-center mb-2">
          <span class="subtitle-2">{{ $t('deviceWinrmHistory') }}</span>
          <span class="caption grey--text ml-2">{{ $t('deviceAuditLogRetainHint') }}</span>
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
          class="winrm-history-table"
        >
          <template v-slot:item.created="{ item }">
            {{ formatTime(item.created) }}
          </template>
          <template v-slot:item.ok="{ item }">
            <v-icon small :color="item.ok ? 'success' : 'error'">
              {{ item.ok ? 'mdi-check' : 'mdi-close' }}
            </v-icon>
          </template>
          <template v-slot:item.command="{ item }">
            <span class="monospace-caption">{{ truncate(item.command, 60) }}</span>
          </template>
          <template v-slot:item.actions="{ item }">
            <v-btn icon x-small @click="viewLog(item)" :title="$t('deviceWinrmViewLog')">
              <v-icon small>mdi-eye</v-icon>
            </v-btn>
          </template>
        </v-data-table>
      </v-card-text>

      <v-card-actions>
        <v-spacer />
        <v-btn text @click="dialog = false">{{ $t('close') }}</v-btn>
      </v-card-actions>
    </v-card>

    <YesNoDialog
      v-model="riskDialog"
      :title="$t('deviceWinrmRiskTitle')"
      :text="$t('deviceWinrmRiskText')"
      @yes="executeCommand"
    />

    <v-dialog v-model="viewLogDialog" max-width="900">
      <v-card v-if="viewingLog">
        <v-card-title>{{ $t('deviceWinrmViewLog') }} #{{ viewingLog.id }}</v-card-title>
        <v-card-text>
          <div class="caption mb-2">
            <span v-if="viewingLog.username">{{ viewingLog.username }} · </span>
            {{ viewingLog.command }}
          </div>
          <pre class="winrm-output">{{ formatLogDetail(viewingLog) }}</pre>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text @click="viewLogDialog = false">{{ $t('close') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-dialog>
</template>

<script>
import axios from 'axios';
import { getErrorMessage } from '@/lib/error';
import DEVICE_WINRM_EXAMPLE_GROUPS from '@/lib/deviceWinrmExamples';
import YesNoDialog from '@/components/YesNoDialog.vue';

export default {
  components: { YesNoDialog },

  props: {
    value: Boolean,
    projectId: [Number, String],
    device: { type: Object, default: null },
  },

  data() {
    return {
      credentialMode: 'winrm',
      command: '',
      forceOffline: false,
      executing: false,
      probing: false,
      formError: null,
      connectionPreview: null,
      lastOutput: null,
      historyLogs: [],
      historyLoading: false,
      riskDialog: false,
      viewLogDialog: false,
      viewingLog: null,
      exampleGroups: DEVICE_WINRM_EXAMPLE_GROUPS,
      exampleMenu: false,
      exampleMenuCommands: [],
    };
  },

  computed: {
    dialog: {
      get() { return this.value; },
      set(v) { this.$emit('input', v); },
    },
    commandTrimmed() {
      return (this.command || '').trim();
    },
    rdpCredentialAvailable() {
      return Boolean(this.device && String(this.device.rdp_user || '').trim());
    },
    winrmStatusColor() {
      const s = this.device && this.device.winrm_status;
      if (s === 'online') return 'success';
      if (s === 'offline') return 'error';
      return 'grey';
    },
    historyHeaders() {
      return [
        { text: this.$t('deviceLastUpdated'), value: 'created', width: '140px' },
        { text: this.$t('user'), value: 'username', width: '100px' },
        { text: 'OK', value: 'ok', width: '48px' },
        { text: this.$t('deviceWinrmCommand'), value: 'command' },
        {
          text: '', value: 'actions', sortable: false, width: '48px',
        },
      ];
    },
    apiBase() {
      return `/api/project/${this.projectId}/devices/${this.device.id}`;
    },
  },

  watch: {
    async value(open) {
      if (open && this.device) {
        this.formError = null;
        this.lastOutput = null;
        this.command = '';
        this.credentialMode = 'winrm';
        this.forceOffline = false;
        await this.probeDevice();
        await this.loadConnectionPreview();
        await this.loadHistory();
      }
    },
  },

  methods: {
    truncate(s, n) {
      const t = String(s || '');
      return t.length > n ? `${t.slice(0, n)}…` : t;
    },
    formatTime(v) {
      if (!v) return '-';
      try {
        return new Date(v).toLocaleString();
      } catch (e) {
        return String(v);
      }
    },
    showExampleMenu(group) {
      const commands = group.commands || [];
      if (commands.length === 1) {
        this.command = commands[0];
        this.exampleMenu = false;
        return;
      }
      this.exampleMenuCommands = commands;
      this.exampleMenu = true;
    },
    async loadConnectionPreview() {
      if (!this.device) return;
      try {
        const res = await axios.get(`${this.apiBase}/winrm/connection-preview`, {
          params: { credential_mode: this.credentialMode },
        });
        this.connectionPreview = res.data;
      } catch (e) {
        this.connectionPreview = null;
      }
    },
    async probeDevice() {
      if (!this.device) return;
      this.probing = true;
      try {
        const res = await axios.post(`${this.apiBase}/probe`);
        Object.assign(this.device, res.data);
        this.$emit('device-updated', res.data);
      } catch (e) {
        this.formError = getErrorMessage(e);
      } finally {
        this.probing = false;
      }
    },
    askExecute() {
      this.riskDialog = true;
    },
    async executeCommand() {
      this.riskDialog = false;
      this.executing = true;
      this.formError = null;
      try {
        if (!this.forceOffline) {
          await this.probeDevice();
        }
        const res = await axios.post(`${this.apiBase}/winrm/exec`, {
          credential_mode: this.credentialMode,
          command: this.commandTrimmed,
          shell: 'powershell',
          timeout_seconds: 60,
          force_offline: this.forceOffline,
        });
        this.lastOutput = res.data;
        if (res.data && res.data.error) {
          this.formError = res.data.message || res.data.error;
        }
        await this.loadHistory();
      } catch (e) {
        if (e.response && e.response.data) {
          this.lastOutput = e.response.data;
        }
        this.formError = getErrorMessage(e);
      } finally {
        this.executing = false;
      }
    },
    formatOutput(o) {
      if (!o) return '';
      const parts = [];
      if (o.stdout) parts.push(o.stdout);
      if (o.stderr) parts.push(`[stderr]\n${o.stderr}`);
      if (o.message && o.error) parts.push(`[error] ${o.message}`);
      return parts.join('\n') || '(empty)';
    },
    formatLogDetail(log) {
      const parts = [];
      if (log.stdout) parts.push(log.stdout);
      if (log.stderr) parts.push(`[stderr]\n${log.stderr}`);
      if (log.error_message) parts.push(`[error] ${log.error_message}`);
      return parts.join('\n') || '(empty)';
    },
    async loadHistory() {
      if (!this.device) return;
      this.historyLoading = true;
      try {
        const res = await axios.get(`${this.apiBase}/winrm/exec-logs`, { params: { limit: 10 } });
        this.historyLogs = (res.data && res.data.logs) || [];
      } catch (e) {
        this.formError = getErrorMessage(e);
      } finally {
        this.historyLoading = false;
      }
    },
    viewLog(item) {
      this.viewingLog = item;
      this.viewLogDialog = true;
    },
  },
};
</script>

<style scoped>
.winrm-console-font {
  font-family: ui-monospace, 'Cascadia Code', 'Cascadia Mono', 'Segoe UI Mono', 'JetBrains Mono',
    'Fira Code', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 13px;
  line-height: 1.5;
  letter-spacing: 0.01em;
  -webkit-font-smoothing: antialiased;
}
.winrm-output {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: 4px;
  max-height: 280px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
}
.monospace-field >>> textarea,
.monospace-field >>> .v-text-field__slot textarea {
  font-family: ui-monospace, 'Cascadia Code', 'Cascadia Mono', 'Segoe UI Mono', 'JetBrains Mono',
    'Fira Code', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 13px;
  line-height: 1.5;
}
.monospace-caption {
  font-family: ui-monospace, 'Cascadia Code', 'Cascadia Mono', 'Segoe UI Mono', 'JetBrains Mono',
    'Fira Code', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 12px;
  line-height: 1.45;
  white-space: normal;
  word-break: break-all;
}
.winrm-history-table >>> .monospace-caption {
  font-size: 11px;
}
</style>
