<template>
  <div>
    <v-toolbar flat>
      <v-app-bar-nav-icon @click="showDrawer()" />
      <v-toolbar-title>{{ $t('deviceDiscoveryTitle') }}</v-toolbar-title>
    </v-toolbar>

    <v-card flat class="mx-4 mb-4" v-if="can(USER_PERMISSIONS.manageProjectResources)">
      <v-card-title class="subtitle-1">{{ $t('deviceDiscoverySettingsTitle') }}</v-card-title>
      <v-card-text>
        <p class="text--secondary mb-4">
          {{ $t('deviceDiscoverySettingsHelp') }}
        </p>
        <v-row dense>
          <v-col cols="12" md="6">
            <v-select
              v-model="settings.discover_template_id"
              :items="templateItems"
              item-text="name"
              item-value="id"
              :label="$t('deviceTemplateDiscover')"
              clearable
              outlined
              dense
            />
          </v-col>
          <v-col cols="12" md="6">
            <v-select
              v-model="settings.default_inventory_id"
              :items="inventoryItems"
              item-text="name"
              item-value="id"
              :label="$t('deviceDiscoveryDefaultInventory')"
              clearable
              outlined
              dense
            />
          </v-col>
        </v-row>
        <v-btn color="primary" depressed :loading="savingSettings" @click="saveSettings">
          {{ $t('save') }}
        </v-btn>
      </v-card-text>
    </v-card>

    <v-card flat class="mx-4">
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
          :disabled="!settings.discover_template_id"
          @click="runDiscoverTemplate"
        >
          {{ $t('deviceDiscoverRunTemplate') }}
        </v-btn>
        <v-alert v-if="!settings.discover_template_id" dense type="info" class="mb-3">
          {{ $t('deviceDiscoveryTemplateRequired') }}
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
import { getErrorMessage } from '@/lib/error';
import { extractJsonArrayFromText, normalizeDiscoveredRows } from '@/lib/deviceDiscoveryParse';

export default {
  mixins: [PermissionsCheck, ProjectMixin],

  data() {
    return {
      settings: {
        discover_template_id: null,
        default_inventory_id: null,
      },
      templates: [],
      inventories: [],
      profiles: [],
      savingSettings: false,
      discovering: false,
      discoverySubnet: '',
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
  },

  async created() {
    await Promise.all([
      this.loadSettings(),
      this.loadTemplates(),
      this.loadInventories(),
      this.loadProfiles(),
    ]);
  },

  methods: {
    showDrawer() {
      EventBus.$emit('i-show-drawer');
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
          discover_template_id: this.settings.discover_template_id,
          default_inventory_id: this.settings.default_inventory_id,
        });
        EventBus.$emit('i-snackbar', { color: 'success', text: this.$t('save') });
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.savingSettings = false;
      }
    },

    notifyDeviceTaskQueued(taskId) {
      EventBus.$emit('i-snackbar', {
        color: 'info',
        text: this.$i18n.t('deviceTaskQueued', { id: taskId }),
      });
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
        if (taskId) {
          this.notifyDeviceTaskQueued(taskId);
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

    async loadDiscoveryResultFromTask(taskId) {
      const outputRes = await axios.get(`/api/project/${this.projectId}/tasks/${taskId}/output`);
      const lines = outputRes.data || [];
      const merged = lines.map((line) => line.output || '').join('\n');
      let parsed = extractJsonArrayFromText(merged);

      if (!parsed) {
        try {
          const rawRes = await axios.get(
            `/api/project/${this.projectId}/tasks/${taskId}/raw_output`,
            { responseType: 'text' },
          );
          parsed = extractJsonArrayFromText(rawRes.data || '');
        } catch (e) {
          // ignore
        }
      }

      if (!parsed) {
        this.discoveryError = this.$i18n.t('deviceDiscoveryResultNotFound');
        return;
      }
      this.discoveredDevices = normalizeDiscoveredRows(parsed);
      this.selectedDiscovered = [...this.discoveredDevices];
      EventBus.$emit('i-snackbar', {
        color: 'success',
        text: this.$i18n.t('deviceDiscoveryLoaded', { count: this.discoveredDevices.length }),
      });
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
