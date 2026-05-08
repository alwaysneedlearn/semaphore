<template>
  <div v-if="items != null">
    <v-dialog v-model="deviceSettingsDialog" :max-width="900">
      <v-card>
        <v-card-title>{{ $t('deviceSettingsTitle') }}</v-card-title>
        <v-card-text>
          <DeviceSettingsForm :project-id="projectId" />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text @click="deviceSettingsDialog = false">{{ $t('close') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="discoveryDialog" :max-width="900">
      <v-card>
        <v-card-title>{{ $t('deviceDiscoveryTitle') }}</v-card-title>
        <v-card-text>
          <p class="text--secondary">
            {{ $t('deviceDiscoveryHelp') }}
          </p>
          <v-text-field
            v-model="discoverySubnet"
            :label="$t('deviceDiscoverySubnet')"
            :hint="$t('deviceDiscoverySubnetHint')"
            persistent-hint
            outlined
            dense
            class="mb-2"
            placeholder="192.168.1.0/24"
          />
          <v-btn
            color="primary"
            class="mb-3"
            :loading="discovering"
            @click="runDiscoverTemplate"
          >
            {{ $t('deviceDiscoverRunTemplate') }}
          </v-btn>
          <v-alert dense type="error" v-if="discoveryError">{{ discoveryError }}</v-alert>
          <v-data-table
            :headers="discoveryHeaders"
            :items="discoveredDevices"
            item-key="hostname"
            show-select
            v-model="selectedDiscovered"
            dense
          >
            <template v-slot:item.device_status="{ item }">
              <v-chip x-small :color="statusColor(item.device_status)" dark>
                {{ item.device_status }}
              </v-chip>
            </template>
            <template v-slot:item.rdp_status="{ item }">
              <v-chip x-small :color="statusColor(item.rdp_status)" dark>
                {{ item.rdp_status }}
              </v-chip>
            </template>
            <template v-slot:item.winrm_status="{ item }">
              <v-chip x-small :color="statusColor(item.winrm_status)" dark>
                {{ item.winrm_status }}
              </v-chip>
            </template>
          </v-data-table>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text @click="discoveryDialog = false">{{ $t('cancel') }}</v-btn>
          <v-btn color="primary" :loading="importingDiscovery" @click="importSelectedDiscovery">
            {{ $t('deviceImportSelected') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <EditDialog
      v-model="editDialog"
      :save-button-text="itemId === 'new' ? $t('create') : $t('save')"
      :icon="'mdi-server-network'"
      icon-color="primary"
      :title="`${itemId === 'new' ? $t('nnew') : $t('edit')} ${$t('device')}`"
      :max-width="500"
      @save="loadItems"
    >
      <template v-slot:form="{ onSave, onError, needSave, needReset }">
        <DeviceForm
          :project-id="projectId"
          :item-id="itemId"
          @save="onSave"
          @error="onError"
          :need-save="needSave"
          :need-reset="needReset"
        />
      </template>
    </EditDialog>

    <YesNoDialog
      :title="$t('deleteDevice')"
      :text="$t('askDeleteDevice')"
      v-model="deleteItemDialog"
      @yes="deleteItem(itemId)"
    />

    <DeviceConfigDialog
      v-model="configDialog"
      :project-id="projectId"
      :device-id="configDeviceId"
      :device-name="configDeviceName"
      @saved="loadItems"
    />
    <v-dialog v-model="reasonDialog" max-width="700">
      <v-card>
        <v-card-title>
          {{ $t('deviceAbnormalReason') }} - {{ reasonDeviceHostname }}
        </v-card-title>
        <v-card-text>
          <v-alert dense type="warning" v-if="reasonError">{{ reasonError }}</v-alert>
          <div class="mb-2">
            <strong>{{ $t('deviceStatus') }}:</strong> {{ reasonData.device_status || '-' }}
          </div>
          <div class="mb-4">
            <strong>{{ $t('deviceAbnormalReason') }}:</strong>
            {{ reasonData.abnormal_reason || $t('deviceReasonEmpty') }}
          </div>
          <v-data-table
            :headers="reasonHeaders"
            :items="reasonData.logs || []"
            dense
            hide-default-footer
            :items-per-page="Number.MAX_VALUE"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text @click="reasonDialog = false">{{ $t('close') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-toolbar flat>
      <v-app-bar-nav-icon @click="showDrawer()"></v-app-bar-nav-icon>
      <v-toolbar-title>{{ $t('devices') }}</v-toolbar-title>
      <v-spacer></v-spacer>
      <v-btn
        v-if="can(USER_PERMISSIONS.manageProjectResources)"
        text
        class="mr-2"
        @click="deviceSettingsDialog = true"
      >
        <v-icon left>mdi-cog</v-icon>
        {{ $t('deviceSettingsTitle') }}
      </v-btn>
      <v-btn
        v-if="can(USER_PERMISSIONS.manageProjectResources)"
        text
        class="mr-2"
        @click="discoveryDialog = true"
      >
        <v-icon left>mdi-radar</v-icon>
        {{ $t('deviceDiscover') }}
      </v-btn>
      <v-btn
        v-if="can(USER_PERMISSIONS.manageProjectResources)"
        text
        class="mr-2"
        :loading="patrolling"
        @click="runPatrol"
      >
        <v-icon left>mdi-stethoscope</v-icon>
        {{ $t('devicePatrolAll') }}
      </v-btn>
      <v-btn
        v-if="can(USER_PERMISSIONS.manageProjectResources)"
        color="primary"
        depressed
        @click="editItem('new')"
      >
        <v-icon left>mdi-plus</v-icon>
        {{ $t('newDevice') }}
      </v-btn>
      <v-menu offset-y v-if="can(USER_PERMISSIONS.manageProjectResources)">
        <template v-slot:activator="{ on, attrs }">
          <v-btn
            class="ml-2"
            outlined
            v-bind="attrs"
            v-on="on"
            :disabled="selectedDevices.length === 0 || bulkLoading"
          >
            <v-icon left>mdi-playlist-check</v-icon>
            {{ $t('deviceBulkOps') }} ({{ selectedDevices.length }})
          </v-btn>
        </template>
        <v-list dense>
          <v-list-item @click="runBulkProbe()">
            <v-list-item-icon><v-icon>mdi-radar</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceProbe') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('start')">
            <v-list-item-icon><v-icon>mdi-play</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceStart') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('stop')">
            <v-list-item-icon><v-icon>mdi-stop</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceStop') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('restart')">
            <v-list-item-icon><v-icon>mdi-restart</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceRestart') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('status')">
            <v-list-item-icon><v-icon>mdi-stethoscope</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceStatusCheck') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('config')">
            <v-list-item-icon><v-icon>mdi-cloud-upload</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceConfigPush') }}</v-list-item-title>
          </v-list-item>
          <v-divider />
          <v-list-item @click="bulkDeleteDialog = true">
            <v-list-item-icon><v-icon color="error">mdi-delete</v-icon></v-list-item-icon>
            <v-list-item-title class="error--text">{{ $t('delete') }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
    </v-toolbar>

    <v-divider />

    <YesNoDialog
      :title="$t('deleteDevice')"
      :text="$t('deviceBulkDeleteConfirm', { count: selectedDevices.length })"
      v-model="bulkDeleteDialog"
      @yes="bulkDelete()"
    />

    <v-row class="ma-4" dense>
      <v-col cols="6" sm="3">
        <v-card outlined>
          <v-card-text class="pa-3">
            <div class="text-overline">{{ $t('deviceTotal') }}</div>
            <div class="text-h4">{{ stats.total }}</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="6" sm="3">
        <v-card outlined>
          <v-card-text class="pa-3">
            <div class="text-overline">
              <v-icon small color="success">mdi-check-circle</v-icon>
              {{ $t('deviceHealthy') }}
            </div>
            <div class="text-h4">{{ stats.healthy }}</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="6" sm="3">
        <v-card outlined>
          <v-card-text class="pa-3">
            <div class="text-overline">
              <v-icon small color="error">mdi-alert-circle</v-icon>
              {{ $t('deviceUnhealthy') }}
            </div>
            <div class="text-h4">{{ stats.unhealthy }}</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="6" sm="3">
        <v-card outlined>
          <v-card-text class="pa-3">
            <div class="text-overline">
              <v-icon small color="warning">mdi-progress-clock</v-icon>
              {{ $t('deviceChecking') }}
            </div>
            <div class="text-h4">{{ stats.checking }}</div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-data-table
      :headers="headers"
      :items="filteredItems"
      item-key="id"
      show-select
      v-model="selectedDevices"
      hide-default-footer
      class="mt-2"
      :items-per-page="Number.MAX_VALUE"
      style="max-width: calc(var(--breakpoint-xl) - var(--nav-drawer-width) - 200px); margin: auto;"
    >
      <template v-slot:top>
        <v-row dense class="px-4 pt-3">
          <v-col cols="12" md="3">
            <v-text-field
              v-model.trim="filters.hostname"
              :label="$t('deviceFilterHostname')"
              clearable
              dense
              outlined
            />
          </v-col>
          <v-col cols="12" md="3">
            <v-text-field
              v-model.trim="filters.ip"
              :label="$t('deviceFilterIp')"
              clearable
              dense
              outlined
            />
          </v-col>
          <v-col cols="12" md="2">
            <v-select
              v-model="filters.deviceStatus"
              :items="statusFilterOptions"
              :label="$t('deviceStatus')"
              clearable
              dense
              outlined
            />
          </v-col>
          <v-col cols="12" md="2">
            <v-select
              v-model="filters.rdpStatus"
              :items="protocolFilterOptions"
              :label="$t('deviceRdpStatus')"
              clearable
              dense
              outlined
            />
          </v-col>
          <v-col cols="12" md="2">
            <v-select
              v-model="filters.winrmStatus"
              :items="protocolFilterOptions"
              :label="$t('deviceWinrmStatus')"
              clearable
              dense
              outlined
            />
          </v-col>
        </v-row>
      </template>
      <template v-slot:item.device_status="{ item }">
        <v-chip x-small :color="statusColor(item.device_status)" dark>
          {{ item.device_status }}
        </v-chip>
      </template>
      <template v-slot:item.rdp_status="{ item }">
        <v-chip x-small :color="statusColor(item.rdp_status)" dark>
          {{ item.rdp_status }}
        </v-chip>
      </template>
      <template v-slot:item.winrm_status="{ item }">
        <v-chip x-small :color="statusColor(item.winrm_status)" dark>
          {{ item.winrm_status }}
        </v-chip>
      </template>
      <template v-slot:item.last_updated="{ item }">
        {{ formatTime(item.last_updated) }}
      </template>
      <template v-slot:item.actions="{ item }">
        <v-btn-toggle dense :value-comparator="() => false">
          <v-btn
            :title="$t('deviceProbe')"
            :loading="busyId === item.id"
            @click="probeDevice(item)"
          >
            <v-icon>mdi-radar</v-icon>
          </v-btn>
          <v-menu offset-y>
            <template v-slot:activator="{ on, attrs }">
              <v-btn :title="$t('deviceActions')" v-bind="attrs" v-on="on">
                <v-icon>mdi-lightning-bolt</v-icon>
              </v-btn>
            </template>
            <v-list dense>
              <v-list-item @click="runAction(item, 'start')">
                <v-list-item-icon><v-icon>mdi-play</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceStart') }}</v-list-item-title>
              </v-list-item>
              <v-list-item @click="runAction(item, 'stop')">
                <v-list-item-icon><v-icon>mdi-stop</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceStop') }}</v-list-item-title>
              </v-list-item>
              <v-list-item @click="runAction(item, 'restart')">
                <v-list-item-icon><v-icon>mdi-restart</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceRestart') }}</v-list-item-title>
              </v-list-item>
              <v-list-item @click="runAction(item, 'status')">
                <v-list-item-icon><v-icon>mdi-stethoscope</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceStatusCheck') }}</v-list-item-title>
              </v-list-item>
              <v-list-item @click="runAction(item, 'config')">
                <v-list-item-icon><v-icon>mdi-cloud-upload</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceConfigPush') }}</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-menu>
          <v-btn :title="$t('deviceConfig')" @click="openConfigDialog(item)">
            <v-icon>mdi-cog</v-icon>
          </v-btn>
          <v-btn
            :title="$t('deviceAbnormalReason')"
            :disabled="item.device_status === 'healthy' || item.device_status === 'unknown'"
            @click="openReasonDialog(item)"
          >
            <v-icon>mdi-alert-circle-outline</v-icon>
          </v-btn>
          <v-btn :title="$t('edit')" @click="editItem(item.id)">
            <v-icon>mdi-pencil</v-icon>
          </v-btn>
          <v-btn :title="$t('delete')" @click="askDeleteItem(item.id)">
            <v-icon>mdi-delete</v-icon>
          </v-btn>
        </v-btn-toggle>
      </template>
    </v-data-table>
  </div>
</template>
<script>
import axios from 'axios';
import EventBus from '@/event-bus';
import ItemListPageBase from '@/components/ItemListPageBase';
import DeviceForm from '@/components/DeviceForm.vue';
import DeviceConfigDialog from '@/components/DeviceConfigDialog.vue';
import DeviceSettingsForm from '@/components/DeviceSettingsForm.vue';
import { getErrorMessage } from '@/lib/error';

export default {
  mixins: [ItemListPageBase],
  components: { DeviceForm, DeviceConfigDialog, DeviceSettingsForm },

  data() {
    return {
      stats: {
        total: 0, healthy: 0, unhealthy: 0, checking: 0, unknown: 0,
      },
      discovering: false,
      patrolling: false,
      deviceSettingsDialog: false,
      busyId: null,
      configDialog: false,
      configDeviceId: null,
      configDeviceName: '',
      reasonDialog: false,
      reasonError: '',
      reasonDeviceHostname: '',
      reasonData: {},
      reasonHeaders: [
        { text: this.$i18n.t('deviceLastUpdated'), value: 'created' },
        { text: this.$i18n.t('deviceStatus'), value: 'status' },
        { text: this.$i18n.t('deviceRdpStatus'), value: 'rdp_status' },
        { text: this.$i18n.t('deviceWinrmStatus'), value: 'winrm_status' },
        { text: this.$i18n.t('deviceAbnormalReason'), value: 'abnormal_reason' },
      ],
      discoveryDialog: false,
      discoverySubnet: '',
      discoveryError: '',
      discoveredDevices: [],
      selectedDiscovered: [],
      selectedDevices: [],
      importingDiscovery: false,
      bulkLoading: false,
      bulkDeleteDialog: false,
      filters: {
        hostname: '',
        ip: '',
        deviceStatus: null,
        rdpStatus: null,
        winrmStatus: null,
      },
      discoveryHeaders: [
        { text: 'Hostname', value: 'hostname' },
        { text: 'IP', value: 'ip_address' },
        { text: 'Status', value: 'device_status' },
        { text: 'RDP', value: 'rdp_status' },
        { text: 'WinRM', value: 'winrm_status' },
      ],
    };
  },
  computed: {
    statusFilterOptions() {
      return ['healthy', 'unhealthy', 'checking', 'unknown'];
    },
    protocolFilterOptions() {
      return ['online', 'offline', 'unknown'];
    },
    filteredItems() {
      return (this.items || []).filter((it) => {
        if (this.filters.hostname && !String(it.hostname || '').toLowerCase().includes(this.filters.hostname.toLowerCase())) {
          return false;
        }
        if (this.filters.ip && !String(it.ip_address || '').toLowerCase().includes(this.filters.ip.toLowerCase())) {
          return false;
        }
        if (this.filters.deviceStatus && it.device_status !== this.filters.deviceStatus) {
          return false;
        }
        if (this.filters.rdpStatus && it.rdp_status !== this.filters.rdpStatus) {
          return false;
        }
        if (this.filters.winrmStatus && it.winrm_status !== this.filters.winrmStatus) {
          return false;
        }
        return true;
      });
    },
  },

  // ItemListPageBase has askDeleteItem call /refs which we don't expose;
  // override to skip the refs check and go straight to the confirm dialog.
  methods: {
    async askDeleteItem(itemId) {
      this.itemId = itemId;
      this.deleteItemDialog = true;
    },

    statusColor(s) {
      if (s === 'healthy' || s === 'online') return 'success';
      if (s === 'unhealthy' || s === 'offline') return 'error';
      if (s === 'checking') return 'warning';
      return 'grey';
    },

    formatTime(t) {
      if (!t) return '\u2014';
      try {
        return new Date(t).toLocaleString();
      } catch (_) {
        return t;
      }
    },

    getHeaders() {
      return [
        { text: this.$i18n.t('deviceIpAddress'), value: 'ip_address', width: '18%' },
        { text: this.$i18n.t('deviceHostname'), value: 'hostname', width: '25%' },
        { text: this.$i18n.t('deviceStatus'), value: 'device_status', width: '12%' },
        { text: this.$i18n.t('deviceRdpStatus'), value: 'rdp_status', width: '9%' },
        { text: this.$i18n.t('deviceWinrmStatus'), value: 'winrm_status', width: '9%' },
        { text: this.$i18n.t('deviceLastUpdated'), value: 'last_updated', width: '15%' },
        { value: 'actions', sortable: false, width: '0%' },
      ];
    },

    getItemsUrl() {
      return `/api/project/${this.projectId}/devices`;
    },
    getSingleItemUrl() {
      return `/api/project/${this.projectId}/devices/${this.itemId}`;
    },
    getEventName() {
      return 'i-device';
    },

    async loadItems() {
      const [devicesRes, statsRes] = await Promise.all([
        axios.get(this.getItemsUrl()),
        axios.get(`${this.getItemsUrl()}/stats`),
      ]);
      this.items = devicesRes.data || [];
      this.stats = statsRes.data || this.stats;
      const selectedIds = new Set(this.selectedDevices.map((x) => x.id));
      this.selectedDevices = this.items.filter((x) => selectedIds.has(x.id));
    },

    async probeDevice(device) {
      this.busyId = device.id;
      try {
        await axios.post(`${this.getItemsUrl()}/${device.id}/probe`);
        await this.loadItems();
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.busyId = null;
      }
    },

    async runAction(device, action) {
      this.busyId = device.id;
      try {
        const res = await axios.post(`${this.getItemsUrl()}/actions/bulk`, {
          action,
          device_ids: [device.id],
        });
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceTaskQueued', { id: res.data && res.data.id }),
        });
        if (res.data && res.data.id) {
          EventBus.$emit('i-show-task', { taskId: res.data.id });
        }
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.busyId = null;
      }
    },

    async discoverDevices() {
      this.discovering = true;
      try {
        const payload = {
          subnet: this.discoverySubnet || null,
        };
        const res = await axios.post(`${this.getItemsUrl()}/discover`, payload);
        const taskId = res.data && res.data.id;
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceTaskQueued', { id: taskId }),
        });
        if (taskId) {
          EventBus.$emit('i-show-task', { taskId });
          await this.waitAndLoadDiscoveryResult(taskId);
        }
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.discovering = false;
      }
    },
    async waitAndLoadDiscoveryResult(taskId, attemptsLeft = 60) {
      const taskRes = await axios.get(`/api/project/${this.projectId}/tasks/${taskId}`);
      const status = (taskRes.data && taskRes.data.status) || '';
      if (status === 'success') {
        await this.loadDiscoveryResultFromTask(taskId);
        return;
      }
      if (status === 'error' || status === 'stopped') {
        this.discoveryError = this.$i18n.t('deviceDiscoveryTaskFailed');
        return;
      }
      if (attemptsLeft <= 1) {
        this.discoveryError = this.$i18n.t('deviceDiscoveryTaskTimeout');
        return;
      }
      await new Promise((resolve) => {
        setTimeout(resolve, 2000);
      });
      await this.waitAndLoadDiscoveryResult(taskId, attemptsLeft - 1);
    },
    tryParseJsonArray(candidate) {
      if (!candidate) {
        return null;
      }
      try {
        const parsed = JSON.parse(candidate.trim());
        return Array.isArray(parsed) ? parsed : null;
      } catch (e) {
        return null;
      }
    },
    stripAnsiCodes(input) {
      if (!input) {
        return '';
      }
      const ESC = '\u001b';
      let i = 0;
      let out = '';
      while (i < input.length) {
        if (input[i] === ESC && input[i + 1] === '[') {
          i += 2;
          while (i < input.length) {
            const ch = input[i];
            if ((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')) {
              i += 1;
              break;
            }
            i += 1;
          }
        } else {
          out += input[i];
          i += 1;
        }
      }
      return out;
    },
    extractJsonArrayFromText(text) {
      if (!text) {
        return null;
      }

      const clean = this.stripAnsiCodes(text);

      const direct = this.tryParseJsonArray(clean);
      if (direct) {
        return direct;
      }

      const escapedFieldRegex = /"(stdout|msg)"\s*:\s*"((?:\\.|[^"\\])*)"/g;
      let escapedFieldMatch = escapedFieldRegex.exec(clean);
      while (escapedFieldMatch !== null) {
        try {
          const unescaped = JSON.parse(`"${escapedFieldMatch[2]}"`);
          const fromField = this.tryParseJsonArray(unescaped);
          if (fromField) {
            return fromField;
          }
        } catch (e) {
          // continue
        }
        escapedFieldMatch = escapedFieldRegex.exec(clean);
      }

      const arrayLikeMatches = clean.match(/\[\s*\{[\s\S]*?\}\s*\]/g) || [];
      for (let i = arrayLikeMatches.length - 1; i >= 0; i -= 1) {
        const parsed = this.tryParseJsonArray(arrayLikeMatches[i]);
        if (parsed) {
          return parsed;
        }
      }

      return null;
    },
    async loadDiscoveryResultFromTask(taskId) {
      const outputRes = await axios.get(`/api/project/${this.projectId}/tasks/${taskId}/output`);
      const lines = outputRes.data || [];
      const merged = lines.map((line) => line.output || '').join('\n');
      let parsed = this.extractJsonArrayFromText(merged);

      if (!parsed) {
        try {
          const rawRes = await axios.get(
            `/api/project/${this.projectId}/tasks/${taskId}/raw_output`,
            { responseType: 'text' },
          );
          parsed = this.extractJsonArrayFromText(rawRes.data || '');
        } catch (e) {
          // ignore raw output fallback errors and keep original message below
        }
      }

      if (!parsed) {
        this.discoveryError = this.$i18n.t('deviceDiscoveryResultNotFound');
        return;
      }
      this.applyDiscoveredDevices(parsed);
      EventBus.$emit('i-snackbar', {
        color: 'success',
        text: this.$i18n.t('deviceDiscoveryLoaded', { count: this.discoveredDevices.length }),
      });
    },
    applyDiscoveredDevices(parsed) {
      this.discoveredDevices = (parsed || [])
        .map((x) => ({
          hostname: (x.hostname || '').trim(),
          ip_address: (x.ip_address || x.ip || '').trim(),
          device_status: x.device_status || x.status || 'unknown',
          rdp_status: x.rdp_status || 'unknown',
          winrm_status: x.winrm_status || 'unknown',
          abnormal_reason: x.abnormal_reason || null,
        }))
        .filter((x) => x.hostname);
      this.selectedDiscovered = [...this.discoveredDevices];
    },
    async runDiscoverTemplate() {
      await this.discoverDevices();
    },
    async importSelectedDiscovery() {
      this.importingDiscovery = true;
      this.discoveryError = '';
      try {
        const payload = {
          devices: this.discoveredDevices,
          selected_hostnames: this.selectedDiscovered.map((x) => x.hostname),
        };
        const res = await axios.post(`${this.getItemsUrl()}/discovery/import`, payload);
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceImportSaved', { count: res.data.saved_count || 0 }),
        });
        this.discoveryDialog = false;
        await this.loadItems();
      } catch (e) {
        this.discoveryError = getErrorMessage(e);
      } finally {
        this.importingDiscovery = false;
      }
    },
    async runPatrol() {
      this.patrolling = true;
      try {
        const res = await axios.post(`${this.getItemsUrl()}/patrol`);
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceTaskQueued', { id: res.data && res.data.id }),
        });
        if (res.data && res.data.id) {
          EventBus.$emit('i-show-task', { taskId: res.data.id });
        }
        await this.loadItems();
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.patrolling = false;
      }
    },
    async runBulkProbe() {
      this.bulkLoading = true;
      try {
        await Promise.all(this.selectedDevices.map((d) => axios.post(`${this.getItemsUrl()}/${d.id}/probe`)));
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceBulkDone', { count: this.selectedDevices.length }),
        });
        await this.loadItems();
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.bulkLoading = false;
      }
    },
    async runBulkAction(action) {
      this.bulkLoading = true;
      try {
        const res = await axios.post(`${this.getItemsUrl()}/actions/bulk`, {
          action,
          device_ids: this.selectedDevices.map((d) => d.id),
        });
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceTaskQueued', { id: res.data && res.data.id }),
        });
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.bulkLoading = false;
      }
    },
    async bulkDelete() {
      this.bulkLoading = true;
      try {
        await Promise.all(this.selectedDevices.map((d) => axios.delete(`${this.getItemsUrl()}/${d.id}`)));
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceBulkDone', { count: this.selectedDevices.length }),
        });
        this.selectedDevices = [];
        await this.loadItems();
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.bulkLoading = false;
      }
    },

    openConfigDialog(device) {
      this.configDeviceId = device.id;
      this.configDeviceName = device.hostname;
      this.configDialog = true;
    },
    async openReasonDialog(device) {
      this.reasonDialog = true;
      this.reasonDeviceHostname = device.hostname;
      this.reasonError = '';
      this.reasonData = {};
      try {
        const res = await axios.get(`${this.getItemsUrl()}/${device.id}/status/reason`);
        this.reasonData = res.data || {};
      } catch (e) {
        this.reasonError = getErrorMessage(e);
      }
    },
  },
};
</script>
