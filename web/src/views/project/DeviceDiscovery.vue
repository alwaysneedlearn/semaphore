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
          v-if="settingsReady && !settings.discover_template_id"
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
          class="mb-3 mr-2"
          :loading="discoveryPolling"
          @click="refreshDiscoveryResultsNow"
        >
          {{ $t('deviceDiscoveryRefreshResults') }}
        </v-btn>
        <v-btn
          text
          class="mb-3 mr-2"
          color="error"
          :disabled="discoveredDevices.length === 0"
          :loading="clearingDiscovery"
          @click="confirmClearDiscovery"
        >
          <v-icon left small>mdi-delete-sweep</v-icon>
          {{ $t('deviceDiscoveryClearList') }}
        </v-btn>

        <v-alert dense type="warning" v-if="discoveryWarning" class="mb-3">
          {{ discoveryWarningText }}
        </v-alert>
        <p class="text--secondary caption mb-3">
          {{ $t('deviceDiscoveryListNotDeviceList') }}
        </p>
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
          <template v-slot:item.hostname="{ item }">
            <v-text-field
              v-model="item.hostname"
              dense
              outlined
              hide-details
              class="mt-1 mb-1"
              style="max-width: 220px"
              :disabled="importingDiscovery"
              @blur="saveDiscoveryHostname(item)"
            />
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
          <template v-slot:item.import_profile_id="{ item }">
            <v-select
              v-model="item.import_profile_id"
              :items="profileItems"
              item-text="name"
              item-value="id"
              :label="$t('deviceDiscoveryImportProfileColumn')"
              dense
              outlined
              hide-details
              class="mt-1 mb-1"
              style="max-width: 220px"
              :disabled="importingDiscovery"
            />
          </template>
        </v-data-table>

        <v-row dense class="mb-3 align-center">
          <v-col cols="12" md="6">
            <v-select
              v-model="bulkImportProfileId"
              :items="profileItems"
              item-text="name"
              item-value="id"
              :label="$t('deviceDiscoveryDefaultImportProfile')"
              :hint="$t('deviceDiscoveryDefaultImportProfileHint')"
              persistent-hint
              outlined
              dense
              clearable
            />
          </v-col>
          <v-col cols="12" md="6" class="d-flex align-center">
            <v-btn
              text
              small
              :disabled="!bulkImportProfileId || selectedDiscovered.length === 0"
              @click="applyBulkImportProfileToSelected"
            >
              {{ $t('deviceDiscoveryApplyDefaultType') }}
            </v-btn>
          </v-col>
        </v-row>

        <v-btn
          color="primary"
          :loading="importingDiscovery"
          :disabled="!canImportSelected"
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
      settingsReady: false,
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
      clearingDiscovery: false,
      bulkImportProfileId: null,
    };
  },

  computed: {
    discoveryHeaders() {
      return [
        { text: 'IP', value: 'ip_address' },
        { text: 'Hostname', value: 'hostname', sortable: false, width: '240px' },
        { text: 'RDP', value: 'rdp_status' },
        { text: 'WinRM', value: 'winrm_status' },
        { text: 'API', value: 'api_status' },
        {
          text: this.$t('deviceDiscoveryImportProfileColumn'),
          value: 'import_profile_id',
          sortable: false,
          width: '240px',
        },
      ];
    },
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
    defaultImportProfileId() {
      const list = this.profiles || [];
      const neware = list.find((p) => {
        const key = String(p.profile_key || p.key || '').toUpperCase();
        return key === 'NEWARE';
      });
      return (neware && neware.id) || (list[0] && list[0].id) || null;
    },
    canImportSelected() {
      if (this.selectedDiscovered.length === 0) {
        return false;
      }
      return this.selectedDiscovered.every((row) => Number(row.import_profile_id) > 0);
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
      try {
        const { data } = await axios.get(`${this.devicesApiBase}/discovery/settings`);
        this.settings = {
          discover_template_id: data.discover_template_id || null,
          default_inventory_id: data.default_inventory_id || null,
        };
      } finally {
        this.settingsReady = true;
      }
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
        profile_key: row.profile_key || row.key,
      }));
      if (!this.bulkImportProfileId && this.defaultImportProfileId) {
        this.bulkImportProfileId = this.defaultImportProfileId;
      }
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
      this.discoveryPolling = true;
      try {
        const syncId = this.discoveryPollTaskId || null;
        if (syncId) {
          await this.fetchDiscoveryResults(syncId, true);
        }
        await this.loadPersistedDiscovery();
      } catch (e) {
        this.discoveryError = getErrorMessage(e);
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
        } else if ((results.devices || []).length > 0) {
          this.discoveryWarning = '';
        }
        const runReady = results.status === 'ready';
        const taskHasRows = (results.devices || []).length > 0;
        if (runReady || taskHasRows) {
          await this.loadPersistedDiscovery();
          if (this.discoveredDevices.length > 0) {
            this.discoveryError = '';
            if (taskStatus === 'success' || runReady) {
              EventBus.$emit('i-snackbar', {
                color: 'success',
                text: this.$i18n.t('deviceDiscoveryLoaded', { count: this.discoveredDevices.length }),
              });
            }
            if (taskStatus === 'success' || runReady) {
              this.stopDiscoveryPoll();
              return;
            }
            this.scheduleDiscoveryPoll(taskId, attemptsLeft - 1);
            return;
          }
          if (taskStatus === 'success' || runReady) {
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
          await this.loadPersistedDiscovery();
          if (this.discoveredDevices.length > 0) {
            this.discoveryError = '';
            EventBus.$emit('i-snackbar', {
              color: 'success',
              text: this.$i18n.t('deviceDiscoveryLoaded', { count: this.discoveredDevices.length }),
            });
            this.stopDiscoveryPoll();
            return;
          }
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

    applyDiscoveredDevicesFromApi(devices, { preserveProfiles = false } = {}) {
      const defaultProfile = this.bulkImportProfileId || this.defaultImportProfileId;
      const profileByIP = {};
      if (preserveProfiles) {
        (this.discoveredDevices || []).forEach((row) => {
          const ip = String(row.ip_address || '').trim();
          if (ip && row.import_profile_id) {
            profileByIP[ip] = row.import_profile_id;
          }
        });
      }
      this.discoveredDevices = (devices || [])
        .map((x) => {
          const ip = (x.ip_address || x.ip || '').trim();
          return {
            hostname: (x.hostname || '').trim(),
            ip_address: ip,
            rdp_status: x.rdp_status || 'offline',
            winrm_status: x.winrm_status || 'offline',
            api_status: x.api_status || 'offline',
            abnormal_reason: x.abnormal_reason || null,
            import_profile_id: profileByIP[ip] || x.import_profile_id || defaultProfile || null,
          };
        })
        .filter((x) => x.ip_address || x.hostname);
      this.selectedDiscovered = [...this.discoveredDevices];
    },

    applyBulkImportProfileToSelected() {
      const pid = this.bulkImportProfileId;
      if (!pid) {
        return;
      }
      const selectedIPs = new Set(
        this.selectedDiscovered.map((r) => String(r.ip_address || '').trim()).filter(Boolean),
      );
      this.discoveredDevices = this.discoveredDevices.map((row) => {
        const ip = String(row.ip_address || '').trim();
        if (!selectedIPs.has(ip)) {
          return row;
        }
        return { ...row, import_profile_id: pid };
      });
      this.selectedDiscovered = this.discoveredDevices.filter((row) => {
        const ip = String(row.ip_address || '').trim();
        return selectedIPs.has(ip);
      });
    },

    removeImportedFromDiscoveryList(importedIPs) {
      const imported = new Set((importedIPs || []).map((ip) => String(ip).trim()).filter(Boolean));
      if (imported.size === 0) {
        return;
      }
      this.discoveredDevices = this.discoveredDevices.filter(
        (row) => !imported.has(String(row.ip_address || '').trim()),
      );
      this.selectedDiscovered = this.selectedDiscovered.filter(
        (row) => !imported.has(String(row.ip_address || '').trim()),
      );
    },

    async loadPersistedDiscovery() {
      try {
        const data = await this.fetchDiscoveryResults();
        this.applyDiscoveredDevicesFromApi(data.devices || [], { preserveProfiles: true });
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

    async saveDiscoveryHostname(item) {
      const ip = String(item.ip_address || '').trim();
      const hostname = String(item.hostname || '').trim();
      if (!ip) {
        return;
      }
      try {
        await axios.patch(`${this.devicesApiBase}/discovery/host`, {
          ip_address: ip,
          hostname: hostname || ip,
        });
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      }
    },

    confirmClearDiscovery() {
      if (this.discoveredDevices.length === 0) {
        return;
      }
      // eslint-disable-next-line no-alert
      if (!window.confirm(this.$t('deviceDiscoveryClearConfirm'))) {
        return;
      }
      this.clearDiscoveryList();
    },

    async clearDiscoveryList() {
      this.clearingDiscovery = true;
      this.discoveryError = '';
      try {
        await axios.delete(`${this.devicesApiBase}/discovery/results`);
        this.discoveredDevices = [];
        this.selectedDiscovered = [];
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$t('deviceDiscoveryClearDone'),
        });
      } catch (e) {
        this.discoveryError = getErrorMessage(e);
      } finally {
        this.clearingDiscovery = false;
      }
    },

    async importSelectedDiscovery() {
      if (!this.canImportSelected) {
        EventBus.$emit('i-snackbar', {
          color: 'error',
          text: this.$i18n.t('deviceDiscoveryImportProfileRequired'),
        });
        return;
      }
      this.importingDiscovery = true;
      this.discoveryError = '';
      try {
        const imports = this.selectedDiscovered.map((row) => ({
          ip_address: String(row.ip_address || row.ip || '').trim(),
          device_profile_id: Number(row.import_profile_id),
        })).filter((x) => x.ip_address && x.device_profile_id > 0);
        const selectedIPs = new Set(imports.map((x) => x.ip_address));
        const payload = {
          devices: this.discoveredDevices
            .filter((row) => selectedIPs.has(String(row.ip_address || '').trim()))
            .map((row) => ({
              hostname: String(row.hostname || '').trim(),
              ip_address: String(row.ip_address || '').trim(),
              rdp_status: row.rdp_status,
              winrm_status: row.winrm_status,
              api_status: row.api_status,
              api_port: row.api_port,
            })),
          imports,
        };
        const res = await axios.post(`${this.devicesApiBase}/discovery/import`, payload);
        const importedIPs = res.data.imported_ips || imports.map((x) => x.ip_address);
        this.removeImportedFromDiscoveryList(importedIPs);
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceImportSaved', { count: res.data.saved_count || 0 }),
        });
      } catch (e) {
        this.discoveryError = getErrorMessage(e);
      } finally {
        this.importingDiscovery = false;
      }
    },
  },
};
</script>
