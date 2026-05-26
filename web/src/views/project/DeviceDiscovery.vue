<template>
  <div>
    <v-toolbar flat>
      <v-app-bar-nav-icon @click="showDrawer()" />
      <v-toolbar-title>{{ $t('deviceDiscoveryTitle') }}</v-toolbar-title>
      <v-spacer />
      <v-btn
        v-if="can(USER_PERMISSIONS.manageProjectResources)"
        text
        small
        class="mr-2"
        @click="openSettingsDialog"
      >
        <v-icon left small>mdi-cog</v-icon>
        {{ $t('deviceDiscoverySettingsBtn') }}
      </v-btn>
    </v-toolbar>

    <v-dialog v-model="settingsDialog" max-width="560" persistent>
      <v-card>
        <v-card-title>{{ $t('deviceDiscoverySettingsTitle') }}</v-card-title>
        <v-card-text>
          <p class="text--secondary mb-4">
            {{ $t('deviceDiscoverySettingsHelp') }}
          </p>
          <v-select
            v-model="settingsDraft.discover_template_id"
            :items="templateItems"
            item-text="name"
            item-value="id"
            :label="$t('deviceTemplateDiscover')"
            clearable
            outlined
            dense
            class="mb-2"
          />
          <v-select
            v-model="settingsDraft.default_inventory_id"
            :items="inventoryItems"
            item-text="name"
            item-value="id"
            :label="$t('deviceDiscoveryDefaultInventory')"
            clearable
            outlined
            dense
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text @click="settingsDialog = false">{{ $t('cancel') }}</v-btn>
          <v-btn color="primary" depressed :loading="savingSettings" @click="saveSettings">
            {{ $t('save') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-card flat class="mx-4">
      <v-card-text>
        <p class="text--secondary">
          {{ $t('deviceDiscoveryHelp') }}
        </p>
        <v-alert
          v-if="!settings.discover_template_id"
          dense
          type="info"
          class="mb-3"
        >
          {{ $t('deviceDiscoveryTemplateRequired') }}
          <v-btn
            v-if="can(USER_PERMISSIONS.manageProjectResources)"
            x-small
            text
            class="ml-1"
            @click="openSettingsDialog"
          >
            {{ $t('deviceDiscoverySettingsBtn') }}
          </v-btn>
        </v-alert>

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
          class="mb-3 mr-2"
          :loading="discovering"
          :disabled="!settings.discover_template_id"
          @click="runDiscoverTemplate"
        >
          {{ $t('deviceDiscoverRunTemplate') }}
        </v-btn>
        <v-btn
          v-if="discoveryPollTaskId"
          text
          class="mb-3"
          :loading="discoveryPolling"
          @click="refreshDiscoveryResultsNow"
        >
          {{ $t('deviceDiscoveryRefreshResults') }}
        </v-btn>

        <v-alert dense type="warning" v-if="discoveryWarning" class="mb-3">
          {{ discoveryWarningText }}
        </v-alert>
        <v-alert dense type="info" class="mb-3">
          {{ $t('deviceDiscoveryListNotDeviceList') }}
        </v-alert>
        <v-alert dense type="error" v-if="discoveryError">{{ discoveryError }}</v-alert>

        <v-data-table
          :headers="discoveryHeaders"
          :items="discoveredDevices"
          item-key="ip_address"
          show-select
          v-model="selectedDiscovered"
          dense
          class="mb-4"
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
          <template v-slot:item.api_status="{ item }">
            <v-chip x-small :color="statusColor(item.api_status)" dark>
              {{ item.api_status }}
            </v-chip>
          </template>
        </v-data-table>

        <v-select
          v-model="importProfileId"
          :items="profileItems"
          item-text="name"
          item-value="id"
          :label="$t('deviceDiscoveryImportProfile')"
          :hint="$t('deviceDiscoveryImportProfileHint')"
          persistent-hint
          outlined
          dense
          class="mb-4"
          :disabled="selectedDiscovered.length === 0"
        />

        <v-btn
          color="primary"
          :loading="importingDiscovery"
          :disabled="!importProfileId || selectedDiscovered.length === 0"
          @click="importSelectedDiscovery"
        >
          {{ $t('deviceImportSelected') }}
        </v-btn>
      </v-card-text>
    </v-card>
  </div>
</template>

<script>
import axios from 'axios';
import EventBus from '@/event-bus';
import PermissionsCheck from '@/components/PermissionsCheck';
import ProjectMixin from '@/components/ProjectMixin';
import { USER_PERMISSIONS } from '@/lib/constants';
import { getErrorMessage } from '@/lib/error';

export default {
  mixins: [PermissionsCheck, ProjectMixin],

  props: {
    projectId: Number,
    userPermissions: Number,
    isAdmin: Boolean,
  },

  data() {
    return {
      USER_PERMISSIONS,
      settings: {
        discover_template_id: null,
        default_inventory_id: null,
      },
      settingsDraft: {
        discover_template_id: null,
        default_inventory_id: null,
      },
      settingsDialog: false,
      templates: [],
      inventories: [],
      profiles: [],
      savingSettings: false,
      discovering: false,
      discoveryPolling: false,
      discoveryPollTaskId: null,
      discoveryPollTimer: null,
      discoverySubnet: '',
      discoveryWarning: '',
      discoveryError: '',
      discoveredDevices: [],
      selectedDiscovered: [],
      importingDiscovery: false,
      importProfileId: null,
      discoveryHeaders: [
        { text: 'IP', value: 'ip_address' },
        { text: 'Hostname', value: 'hostname' },
        { text: 'Status', value: 'device_status' },
        { text: 'RDP', value: 'rdp_status' },
        { text: 'WinRM', value: 'winrm_status' },
        { text: 'API', value: 'api_status' },
      ],
    };
  },

  computed: {
    templateItems() {
      return (this.templates || []).map((t) => ({ id: t.id, name: t.name }));
    },
    inventoryItems() {
      return (this.inventories || []).map((i) => ({ id: i.id, name: i.name }));
    },
    profileItems() {
      return (this.profiles || []).map((p) => ({
        id: p.id,
        name: p.name || p.key || `Profile #${p.id}`,
      }));
    },
    devicesApiBase() {
      return `/api/project/${this.projectId}/devices`;
    },
    discoveryWarningText() {
      if (this.discoveryWarning === 'missing_semaphore_api_token') {
        return this.$t('deviceDiscoveryMissingApiToken');
      }
      return this.discoveryWarning;
    },
  },

  async created() {
    await Promise.all([
      this.loadSettings(),
      this.loadTemplates(),
      this.loadInventories(),
      this.loadProfiles(),
      this.loadPersistedDiscovery(),
    ]);
  },

  beforeDestroy() {
    this.stopDiscoveryPoll();
  },

  methods: {
    showDrawer() {
      EventBus.$emit('i-show-drawer');
    },

    openSettingsDialog() {
      this.settingsDraft = {
        discover_template_id: this.settings.discover_template_id,
        default_inventory_id: this.settings.default_inventory_id,
      };
      this.settingsDialog = true;
    },

    statusColor(status) {
      if (status === 'healthy' || status === 'online') {
        return 'success';
      }
      if (status === 'checking') {
        return 'info';
      }
      if (status === 'unhealthy' || status === 'offline') {
        return 'error';
      }
      return 'grey';
    },

    async loadSettings() {
      const { data } = await axios.get(`${this.devicesApiBase}/discovery/settings`);
      this.settings = {
        discover_template_id: data.discover_template_id || null,
        default_inventory_id: data.default_inventory_id || null,
      };
    },

    async loadTemplates() {
      const { data } = await axios.get(`/api/project/${this.projectId}/templates`);
      this.templates = data || [];
    },

    async loadInventories() {
      const { data } = await axios.get(`/api/project/${this.projectId}/inventory`);
      this.inventories = data || [];
    },

    async loadProfiles() {
      const { data } = await axios.get(`${this.devicesApiBase}/profiles`);
      this.profiles = (data || []).map((row) => ({
        id: row.id,
        name: row.name || row.profile_key || row.key || `Type #${row.id}`,
      }));
    },

    async saveSettings() {
      this.savingSettings = true;
      try {
        await axios.put(`${this.devicesApiBase}/discovery/settings`, {
          discover_template_id: this.settingsDraft.discover_template_id,
          default_inventory_id: this.settingsDraft.default_inventory_id,
        });
        this.settings = { ...this.settingsDraft };
        this.settingsDialog = false;
        EventBus.$emit('i-snackbar', { color: 'success', text: this.$t('save') });
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.savingSettings = false;
      }
    },

    /** Snackbar with clickable task id + open task log dialog (same as device list actions). */
    notifyDeviceTaskQueued(taskId) {
      const id = taskId != null ? Number(taskId) : null;
      if (!id) {
        return;
      }
      EventBus.$emit('i-snackbar', {
        color: 'success',
        textPrefix: this.$t('deviceTaskQueuedPrefix'),
        textSuffix: this.$t('deviceTaskQueuedSuffix'),
        taskId: id,
      });
      EventBus.$emit('i-show-task', { taskId: id });
    },

    stopDiscoveryPoll() {
      if (this.discoveryPollTimer) {
        clearTimeout(this.discoveryPollTimer);
        this.discoveryPollTimer = null;
      }
      this.discoveryPollTaskId = null;
      this.discoveryPolling = false;
    },

    startDiscoveryPoll(taskId) {
      this.stopDiscoveryPoll();
      this.discoveryPollTaskId = taskId;
      this.scheduleDiscoveryPoll(taskId, 60);
    },

    scheduleDiscoveryPoll(taskId, attemptsLeft) {
      this.discoveryPollTimer = setTimeout(async () => {
        await this.pollDiscoveryOnce(taskId, attemptsLeft);
      }, 2000);
    },

    async refreshDiscoveryResultsNow() {
      if (!this.discoveryPollTaskId) {
        await this.loadPersistedDiscovery();
        return;
      }
      this.discoveryPolling = true;
      try {
        await this.pollDiscoveryOnce(this.discoveryPollTaskId, 1);
      } finally {
        this.discoveryPolling = false;
      }
    },

    async pollDiscoveryOnce(taskId, attemptsLeft) {
      if (!this.discoveryPollTaskId || this.discoveryPollTaskId !== taskId) {
        return;
      }
      let taskStatus = '';
      try {
        const taskRes = await axios.get(`/api/project/${this.projectId}/tasks/${taskId}`);
        taskStatus = (taskRes.data && taskRes.data.status) || '';
      } catch (e) {
        this.discoveryError = getErrorMessage(e);
        this.stopDiscoveryPoll();
        return;
      }

      if (taskStatus === 'error' || taskStatus === 'stopped') {
        this.discoveryError = this.$i18n.t('deviceDiscoveryTaskFailed');
        this.stopDiscoveryPoll();
        return;
      }

      try {
        let results = await this.fetchDiscoveryResults(taskId);
        const trySync = taskStatus === 'success'
          && (results.devices || []).length === 0;
        if (trySync) {
          results = await this.fetchDiscoveryResults(taskId, true);
        }
        if (results.callback_hint === 'missing_semaphore_api_token') {
          this.discoveryWarning = 'missing_semaphore_api_token';
        }
        if (results.status === 'ready' || (results.devices || []).length > 0) {
          if ((results.devices || []).length > 0) {
            this.applyDiscoveredDevicesFromApi(results.devices);
            this.discoveryError = '';
            EventBus.$emit('i-snackbar', {
              color: 'success',
              text: this.$i18n.t('deviceDiscoveryLoaded', { count: this.discoveredDevices.length }),
            });
            this.stopDiscoveryPoll();
            return;
          }
          if (results.status === 'ready') {
            this.discoveryError = this.$i18n.t('deviceDiscoveryCallbackEmpty');
            this.stopDiscoveryPoll();
            return;
          }
        }
      } catch (e) {
        if (attemptsLeft <= 1) {
          this.discoveryError = getErrorMessage(e);
          this.stopDiscoveryPoll();
          return;
        }
      }

      if (attemptsLeft <= 1) {
        if (taskStatus === 'success') {
          this.discoveryError = this.$i18n.t('deviceDiscoveryCallbackMissing');
        } else {
          this.discoveryError = this.$i18n.t('deviceDiscoveryTaskTimeout');
        }
        this.stopDiscoveryPoll();
        return;
      }

      this.scheduleDiscoveryPoll(taskId, attemptsLeft - 1);
    },

    async discoverDevices() {
      const sub = String(this.discoverySubnet || '').trim();
      if (!sub) {
        EventBus.$emit('i-snackbar', {
          color: 'error',
          text: this.$i18n.t('deviceDiscoverySubnetRequired'),
        });
        return;
      }
      this.discovering = true;
      this.discoveryError = '';
      try {
        const res = await axios.post(`${this.devicesApiBase}/discover`, { subnet: sub });
        const taskId = res.data && res.data.id;
        this.discoveryWarning = res.data?.discovery_warning || '';
        if (taskId) {
          this.notifyDeviceTaskQueued(taskId);
          this.startDiscoveryPoll(taskId);
        }
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.discovering = false;
      }
    },

    applyDiscoveredDevicesFromApi(devices) {
      this.discoveredDevices = (devices || [])
        .map((x) => ({
          hostname: (x.hostname || '').trim(),
          ip_address: (x.ip_address || x.ip || '').trim(),
          device_status: x.device_status || x.status || 'unhealthy',
          rdp_status: x.rdp_status || 'offline',
          winrm_status: x.winrm_status || 'offline',
          api_status: x.api_status || 'offline',
          abnormal_reason: x.abnormal_reason || null,
        }))
        .filter((x) => x.ip_address || x.hostname);
      this.selectedDiscovered = [...this.discoveredDevices];
    },

    async loadPersistedDiscovery() {
      try {
        const data = await this.fetchDiscoveryResults();
        if ((data.devices || []).length > 0) {
          this.applyDiscoveredDevicesFromApi(data.devices);
        }
        if (this.discoveryPollTaskId && (data.devices || []).length === 0) {
          const synced = await this.fetchDiscoveryResults(this.discoveryPollTaskId, true);
          if ((synced.devices || []).length > 0) {
            this.applyDiscoveredDevicesFromApi(synced.devices);
          }
        }
      } catch (e) {
        // ignore — empty table until first scan
      }
    },

    async fetchDiscoveryResults(taskId, syncFromLog = false) {
      const params = {};
      if (taskId) {
        params.task_id = taskId;
      }
      if (syncFromLog) {
        params.sync = '1';
      }
      const { data } = await axios.get(`${this.devicesApiBase}/discovery/results`, { params });
      return data || {};
    },

    async runDiscoverTemplate() {
      await this.discoverDevices();
    },

    async importSelectedDiscovery() {
      if (!this.importProfileId) {
        EventBus.$emit('i-snackbar', {
          color: 'error',
          text: this.$i18n.t('deviceDiscoveryImportProfileRequired'),
        });
        return;
      }
      this.importingDiscovery = true;
      this.discoveryError = '';
      try {
        const payload = {
          devices: this.discoveredDevices,
          selected_ips: this.selectedDiscovered
            .map((x) => String(x.ip_address || x.ip || '').trim())
            .filter((ip) => ip),
          device_profile_id: this.importProfileId,
        };
        const res = await axios.post(`${this.devicesApiBase}/discovery/import`, payload);
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceImportSaved', { count: res.data.saved_count || 0 }),
        });
        await this.$router.push(`/project/${this.projectId}/devices/list`);
      } catch (e) {
        this.discoveryError = getErrorMessage(e);
      } finally {
        this.importingDiscovery = false;
      }
    },
  },
};
</script>
