<template>
  <div class="profile-settings-editor">
    <v-alert v-if="saveError" type="error" dense dismissible class="mb-3" @input="saveError = null">
      {{ saveError }}
    </v-alert>

    <v-card outlined class="mb-3">
      <v-card-subtitle class="pb-0 font-weight-medium">
        Playbook templates
      </v-card-subtitle>
      <v-card-text>
        <v-row dense>
          <v-col cols="12" sm="6" v-for="cfg in templateActions" :key="cfg.field">
            <v-autocomplete
              v-model="settings[cfg.field]"
              :items="templates"
              item-value="id"
              item-text="name"
              :label="cfg.label"
              clearable
              outlined
              dense
              hide-details="auto"
              :disabled="saving"
            />
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <v-card outlined class="mb-3">
      <v-card-subtitle class="pb-0 font-weight-medium">
        Scheduled status refresh
      </v-card-subtitle>
      <v-card-text>
        <p class="text--secondary caption mb-3">
          When interval &gt; 0, Semaphore runs the check-restart-redeploy template on a timer
          (falls back to Status / Patrol template if not set).
        </p>
        <v-row dense>
          <v-col cols="12" sm="6">
            <v-text-field
              v-model.number="settings.status_refresh_interval_min"
              :label="$t('deviceRefreshIntervalMinutes')"
              :hint="$t('deviceRefreshIntervalHelp')"
              persistent-hint
              type="number"
              min="0"
              outlined
              dense
              :disabled="saving"
            />
          </v-col>
          <v-col cols="12" sm="6">
            <v-autocomplete
              v-model="settings.check_restart_redeploy_template_id"
              :items="templates"
              item-value="id"
              item-text="name"
              label="Check-restart-redeploy template"
              clearable
              outlined
              dense
              hide-details="auto"
              :disabled="saving"
            />
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <v-card outlined class="mb-3">
      <v-card-subtitle class="pb-0 font-weight-medium">
        Inventory
      </v-card-subtitle>
      <v-card-text>
        <v-autocomplete
          v-model="settings.default_inventory_id"
          :items="inventories"
          item-value="id"
          item-text="name"
          label="Default inventory"
          clearable
          outlined
          dense
          hide-details="auto"
          :disabled="saving"
        />
      </v-card-text>
    </v-card>

    <v-card outlined class="mb-3">
      <v-card-subtitle class="pb-0 font-weight-medium">
        Default configuration (devices of this type)
      </v-card-subtitle>
      <v-card-text>
        <v-data-table
          :headers="defaultConfigHeaders"
          :items="defaultConfigItems"
          dense
          hide-default-footer
          :items-per-page="-1"
          class="elevation-0"
        >
          <template v-slot:item.category="{ item }">
            <v-text-field v-model="item.category" hide-details dense outlined :disabled="saving" />
          </template>
          <template v-slot:item.key="{ item }">
            <v-text-field v-model="item.key" hide-details dense outlined :disabled="saving" />
          </template>
          <template v-slot:item.value="{ item }">
            <v-text-field v-model="item.value" hide-details dense outlined :disabled="saving" />
          </template>
          <template v-slot:item.actions="{ index }">
            <v-btn icon small @click="removeDefaultConfigRow(index)" :disabled="saving">
              <v-icon>mdi-close</v-icon>
            </v-btn>
          </template>
        </v-data-table>
        <v-btn small text class="mt-2" @click="addDefaultConfigRow" :disabled="saving">
          <v-icon left>mdi-plus</v-icon>
          {{ $t('deviceConfigAddRow') }}
        </v-btn>
      </v-card-text>
    </v-card>

    <div class="profile-settings-editor__actions">
      <v-btn text small :disabled="saving" @click="loadSettings">
        <v-icon left small>mdi-refresh</v-icon>
        Reset
      </v-btn>
      <v-btn color="primary" :loading="saving" @click="save">
        <v-icon left>mdi-content-save</v-icon>
        {{ $t('save') }}
      </v-btn>
    </div>
  </div>
</template>

<script>
import axios from 'axios';
import EventBus from '@/event-bus';
import { getErrorMessage } from '@/lib/error';

const TEMPLATE_ID_FIELDS = [
  'start_template_id',
  'stop_template_id',
  'restart_template_id',
  'status_template_id',
  'check_restart_redeploy_template_id',
];

export default {
  props: {
    projectId: { type: Number, required: true },
    profileId: { type: Number, required: true },
  },
  data() {
    return {
      settings: {},
      templates: [],
      inventories: [],
      saving: false,
      saveError: null,
      defaultConfigItems: [],
      defaultConfigHeaders: [
        { text: this.$i18n.t('deviceConfigCategory'), value: 'category', width: '25%' },
        { text: this.$i18n.t('deviceConfigKey'), value: 'key', width: '30%' },
        { text: this.$i18n.t('deviceConfigValue'), value: 'value', width: '40%' },
        { value: 'actions', sortable: false, width: '5%' },
      ],
      templateActions: [
        { field: 'start_template_id', label: 'Start template' },
        { field: 'stop_template_id', label: 'Stop template' },
        { field: 'restart_template_id', label: 'Restart template' },
        { field: 'status_template_id', label: 'Status / Patrol template' },
      ],
    };
  },
  watch: {
    profileId: {
      immediate: false,
      handler() {
        this.loadSettings();
      },
    },
  },
  async created() {
    await Promise.all([
      this.loadSettings(),
      this.loadTemplates(),
      this.loadInventories(),
    ]);
  },
  methods: {
    async loadTemplates() {
      const { data } = await axios.get(`/api/project/${this.projectId}/templates`);
      this.templates = data || [];
    },
    async loadInventories() {
      const { data } = await axios.get(`/api/project/${this.projectId}/inventory`);
      this.inventories = data || [];
    },
    normalizeTemplateId(value) {
      if (value === '' || value === undefined || value === null) {
        return null;
      }
      const n = Number(value);
      return Number.isFinite(n) && n > 0 ? n : null;
    },
    applySettingsFromApi(data) {
      const next = { ...data };
      TEMPLATE_ID_FIELDS.forEach((field) => {
        next[field] = this.normalizeTemplateId(next[field]);
      });
      next.default_inventory_id = this.normalizeTemplateId(next.default_inventory_id);
      next.status_refresh_interval_min = Number(next.status_refresh_interval_min) || 0;
      this.settings = next;
      this.defaultConfigItems = this.parseDefaultConfigJson(next.default_config_json);
    },
    async loadSettings() {
      this.saveError = null;
      const { data } = await axios.get(
        `/api/project/${this.projectId}/devices/profiles/${this.profileId}/settings`,
      );
      this.applySettingsFromApi(data);
    },
    parseDefaultConfigJson(raw) {
      if (!raw || String(raw).trim() === '') {
        return [];
      }
      try {
        const parsed = JSON.parse(raw);
        const rows = [];
        Object.keys(parsed || {}).forEach((category) => {
          const group = parsed[category] || {};
          Object.keys(group).forEach((key) => {
            rows.push({
              category,
              key,
              value: String(group[key] == null ? '' : group[key]),
            });
          });
        });
        return rows;
      } catch (e) {
        return [];
      }
    },
    buildDefaultConfigJson() {
      const categorized = {};
      this.defaultConfigItems.forEach((item) => {
        const category = (item.category || '').trim();
        const key = (item.key || '').trim();
        if (!category || !key) {
          return;
        }
        if (!categorized[category]) {
          categorized[category] = {};
        }
        categorized[category][key] = item.value == null ? '' : String(item.value);
      });
      return JSON.stringify(categorized);
    },
    buildSavePayload() {
      const payload = { ...this.settings };
      payload.default_config_json = this.buildDefaultConfigJson();
      payload.status_refresh_interval_min = Number(payload.status_refresh_interval_min) || 0;
      TEMPLATE_ID_FIELDS.forEach((field) => {
        payload[field] = this.normalizeTemplateId(payload[field]);
      });
      payload.default_inventory_id = this.normalizeTemplateId(payload.default_inventory_id);
      delete payload.default_ansible_user;
      delete payload.default_ansible_password;
      delete payload.default_ansible_connection;
      delete payload.default_ansible_winrm_transport;
      delete payload.default_ansible_winrm_scheme;
      delete payload.default_ansible_port;
      delete payload.default_ansible_winrm_server_cert_validation;
      delete payload.config_template_id;
      delete payload.discover_template_id;
      return payload;
    },
    addDefaultConfigRow() {
      this.defaultConfigItems.push({ category: 'SystemConfig', key: '', value: '' });
    },
    removeDefaultConfigRow(index) {
      this.defaultConfigItems.splice(index, 1);
    },
    async save() {
      this.saving = true;
      this.saveError = null;
      try {
        const { data } = await axios.put(
          `/api/project/${this.projectId}/devices/profiles/${this.profileId}/settings`,
          this.buildSavePayload(),
        );
        this.applySettingsFromApi(data);
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: 'Device type settings saved',
        });
        this.$emit('saved');
      } catch (e) {
        this.saveError = getErrorMessage(e);
        EventBus.$emit('i-snackbar', { color: 'error', text: this.saveError });
      } finally {
        this.saving = false;
      }
    },
  },
};
</script>

<style scoped>
.profile-settings-editor__actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  padding: 12px 0 4px;
  border-top: 1px solid rgba(0, 0, 0, 0.08);
}
</style>
