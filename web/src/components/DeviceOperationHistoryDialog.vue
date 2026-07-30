<template>
  <v-dialog v-model="dialog" :max-width="960" scrollable>
    <v-card v-if="dialog && device">
      <v-card-title class="d-flex align-center">
        <v-icon class="mr-2">mdi-history</v-icon>
        {{ $t('deviceDetailTitle') }} &mdash; {{ device.hostname }} ({{ device.ip_address }})
        <v-spacer />
        <v-btn icon small :loading="loading" @click="load">
          <v-icon>mdi-refresh</v-icon>
        </v-btn>
      </v-card-title>

      <v-tabs v-model="tab" background-color="transparent" class="px-4">
        <v-tab>{{ $t('deviceOperationHistoryTabOps') }}</v-tab>
        <v-tab>{{ $t('deviceOperationHistoryTabRdp') }}</v-tab>
      </v-tabs>

      <v-card-text>
        <v-alert v-if="error" type="error" dense>{{ error }}</v-alert>

        <v-progress-linear v-if="loading" indeterminate class="mb-2" />

        <v-tabs-items v-model="tab">
          <v-tab-item>
            <div class="caption grey--text mb-3">{{ $t('deviceOperationHistoryHint') }}</div>

            <v-expansion-panels v-if="logs.length" accordion>
              <v-expansion-panel v-for="log in logs" :key="log.id">
                <v-expansion-panel-header>
                  <div class="d-flex align-center flex-wrap" style="gap: 8px;">
                    <v-chip x-small :color="operationColor(log.operation)" dark>
                      {{ operationLabel(log.operation) }}
                    </v-chip>
                    <v-chip x-small :color="resultColor(log.result)" dark>
                      {{ resultLabel(log.result) }}
                    </v-chip>
                    <span class="caption">{{ formatTime(log.created) }}</span>
                    <span v-if="log.task_id" class="caption grey--text">#{{ log.task_id }}</span>
                  </div>
                </v-expansion-panel-header>
                <v-expansion-panel-content>
                  <div class="mb-2">
                    <strong>{{ $t('deviceOperationSummary') }}:</strong>
                    {{ log.summary || '-' }}
                  </div>
                  <v-simple-table dense>
                    <thead>
                      <tr>
                        <th>{{ $t('deviceOperationStep') }}</th>
                        <th>{{ $t('deviceOperationStepResult') }}</th>
                        <th>{{ $t('deviceOperationStepDetail') }}</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="(step, idx) in (log.steps || [])" :key="idx">
                        <td>{{ step.step }}</td>
                        <td>
                          <v-chip x-small :color="stepResultColor(step.result)" dark>
                            {{ step.result }}
                          </v-chip>
                        </td>
                        <td class="text--secondary">{{ step.detail || '-' }}</td>
                      </tr>
                    </tbody>
                  </v-simple-table>
                </v-expansion-panel-content>
              </v-expansion-panel>
            </v-expansion-panels>

            <div v-else-if="!loading" class="text--secondary py-4 text-center">
              {{ $t('deviceOperationHistoryEmpty') }}
            </div>
          </v-tab-item>

          <v-tab-item>
            <div class="caption grey--text mb-3">{{ $t('deviceRdpLaunchHistoryHint') }}</div>

            <v-simple-table v-if="rdpLogs.length" dense>
              <thead>
                <tr>
                  <th>{{ $t('deviceRdpLaunchTime') }}</th>
                  <th>{{ $t('deviceRdpLaunchUser') }}</th>
                  <th>{{ $t('deviceRdpLaunchPhase') }}</th>
                  <th>{{ $t('deviceRdpLaunchTarget') }}</th>
                  <th>{{ $t('deviceRdpLaunchClientIp') }}</th>
                  <th>{{ $t('deviceRdpLaunchHelperAt') }}</th>
                  <th>{{ $t('deviceRdpLaunchMstscStarted') }}</th>
                  <th>{{ $t('deviceRdpLaunchMstscExited') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in rdpLogs" :key="row.id">
                  <td class="caption">{{ formatTime(row.created) }}</td>
                  <td>{{ row.username || ('#' + row.user_id) }}</td>
                  <td>
                    <v-chip x-small :color="rdpPhaseColor(row.phase)" dark>
                      {{ rdpPhaseLabel(row.phase) }}
                    </v-chip>
                  </td>
                  <td class="caption">
                    {{ row.host }}:{{ row.rdp_port }}
                    <span v-if="row.rdp_user" class="grey--text"> ({{ row.rdp_user }})</span>
                  </td>
                  <td class="caption">{{ row.client_ip || '-' }}</td>
                  <td class="caption">{{ formatTime(row.helper_fetched_at) }}</td>
                  <td class="caption">{{ formatTime(row.mstsc_started_at) }}</td>
                  <td class="caption">{{ formatTime(row.mstsc_exited_at) }}</td>
                </tr>
              </tbody>
            </v-simple-table>

            <div v-else-if="!loading" class="text--secondary py-4 text-center">
              {{ $t('deviceRdpLaunchHistoryEmpty') }}
            </div>
          </v-tab-item>
        </v-tabs-items>
      </v-card-text>

      <v-card-actions>
        <v-spacer />
        <v-btn text @click="dialog = false">{{ $t('close') }}</v-btn>
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
      loading: false,
      error: null,
      tab: 0,
      logs: [],
      total: 0,
      rdpLogs: [],
      rdpTotal: 0,
    };
  },

  computed: {
    dialog: {
      get() { return this.value; },
      set(v) { this.$emit('input', v); },
    },
  },

  watch: {
    value(open) {
      if (open && this.device) {
        this.tab = 0;
        this.load();
      }
    },
    device() {
      if (this.dialog && this.device) {
        this.load();
      }
    },
  },

  methods: {
    async load() {
      if (!this.device?.id) return;
      this.loading = true;
      this.error = null;
      try {
        const base = `/api/project/${this.projectId}/devices/${this.device.id}`;
        const [opsRes, rdpRes] = await Promise.all([
          axios.get(`${base}/operations`, { params: { limit: 100, offset: 0 } }),
          axios.get(`${base}/rdp/launch-logs`, { params: { limit: 100, offset: 0 } }),
        ]);
        this.logs = opsRes.data.logs || [];
        this.total = opsRes.data.total || 0;
        this.rdpLogs = rdpRes.data.logs || [];
        this.rdpTotal = rdpRes.data.total || 0;
      } catch (e) {
        this.error = getErrorMessage(e);
        this.logs = [];
        this.rdpLogs = [];
      } finally {
        this.loading = false;
      }
    },

    formatTime(t) {
      if (!t) return '-';
      try {
        return new Date(t).toLocaleString();
      } catch (e) {
        return t;
      }
    },

    operationLabel(op) {
      if (op === 'redeploy') return this.$t('deviceRedeploy');
      if (op === 'status') return this.$t('deviceOperationStatus');
      if (op === 'resend_data') return this.$t('deviceResendData');
      return this.$t('deviceRestart');
    },

    operationColor(op) {
      if (op === 'redeploy') return 'deep-purple';
      if (op === 'status') return 'orange';
      if (op === 'resend_data') return 'teal';
      return 'primary';
    },

    resultLabel(result) {
      return result === 'success' ? this.$t('deviceOperationSuccess') : this.$t('deviceOperationFailed');
    },

    resultColor(result) {
      return result === 'success' ? 'success' : 'error';
    },

    stepResultColor(result) {
      const r = (result || '').toLowerCase();
      if (r === 'ok' || r === 'success') return 'success';
      if (r === 'skipped') return 'grey';
      return 'error';
    },

    rdpPhaseLabel(phase) {
      if (phase === 'mstsc_exited') return this.$t('deviceRdpLaunchPhaseExited');
      if (phase === 'mstsc_started') return this.$t('deviceRdpLaunchPhaseStarted');
      if (phase === 'helper_fetched') return this.$t('deviceRdpLaunchPhaseFetched');
      return this.$t('deviceRdpLaunchPhaseRequested');
    },

    rdpPhaseColor(phase) {
      if (phase === 'mstsc_exited') return 'grey';
      if (phase === 'mstsc_started') return 'success';
      if (phase === 'helper_fetched') return 'primary';
      return 'info';
    },
  },
};
</script>
