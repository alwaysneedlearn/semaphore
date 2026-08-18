<template>
  <div>
    <v-alert v-if="formError" color="error" dense class="mb-2">{{ formError }}</v-alert>

    <v-alert type="info" dense class="mb-3" outlined>
      Templates belong under
      <strong>Devices → Device types</strong>.
      Periodic check-restart should be configured as a Semaphore Schedule.
      This form is kept only for migration compatibility.
    </v-alert>

    <p class="text--secondary mb-2">{{ $t('deviceSettingsHelp') }}</p>

    <v-row dense>
      <v-col cols="12" md="6" v-for="cfg in actions" :key="cfg.field">
        <v-autocomplete
          v-model="settings[cfg.field]"
          :items="templates"
          item-value="id"
          item-text="name"
          :label="$t(cfg.label)"
          clearable
          outlined
          dense
          :disabled="saving"
        />
      </v-col>
    </v-row>

    <v-divider class="my-4" />
    <p class="text--secondary mb-2">默认配置（全部设备）</p>
    <v-data-table
      :headers="defaultConfigHeaders"
      :items="defaultConfigItems"
      dense
      hide-default-footer
      :items-per-page="Number.MAX_VALUE"
    >
      <template v-slot:item.category="{ item }">
        <v-text-field
          v-model="item.category"
          hide-details
          dense
          outlined
          placeholder="SystemConfig"
          :disabled="saving"
        />
      </template>
      <template v-slot:item.key="{ item }">
        <v-text-field
          v-model="item.key"
          hide-details
          dense
          outlined
          :disabled="saving"
        />
      </template>
      <template v-slot:item.value="{ item }">
        <v-text-field
          v-model="item.value"
          hide-details
          dense
          outlined
          :disabled="saving"
        />
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

    <v-divider class="my-4" />

    <p class="text--secondary mb-2">{{ $t('deviceConnectionDefaultsHelp') }}</p>
    <v-row dense>
      <v-col cols="12" md="6">
        <v-text-field
          v-model="settings.default_ansible_user"
          :label="$t('deviceAnsibleUser')"
          :disabled="saving"
          outlined
          dense
        />
      </v-col>
      <v-col cols="12" md="6">
        <v-text-field
          v-model="settings.default_ansible_password"
          :label="$t('deviceAnsiblePassword')"
          :disabled="saving"
          outlined
          dense
          :type="showDefaultAnsiblePassword ? 'text' : 'password'"
          autocomplete="new-password"
          :append-icon="showDefaultAnsiblePassword ? 'mdi-eye-off' : 'mdi-eye'"
          @click:append="showDefaultAnsiblePassword = !showDefaultAnsiblePassword"
        />
      </v-col>
      <v-col cols="12" md="6">
        <v-text-field
          v-model="settings.default_ansible_connection"
          :label="$t('deviceAnsibleConnection')"
          :disabled="saving"
          outlined
          dense
        />
      </v-col>
      <v-col cols="12" md="6">
        <v-text-field
          v-model="settings.default_ansible_winrm_transport"
          :label="$t('deviceAnsibleWinrmTransport')"
          :disabled="saving"
          outlined
          dense
        />
      </v-col>
      <v-col cols="12" md="6">
        <v-text-field
          v-model="settings.default_ansible_winrm_scheme"
          :label="$t('deviceAnsibleWinrmScheme')"
          :disabled="saving"
          outlined
          dense
        />
      </v-col>
      <v-col cols="12" md="6">
        <v-text-field
          v-model.number="settings.default_ansible_port"
          :label="$t('deviceAnsiblePort')"
          :disabled="saving"
          outlined
          dense
          type="number"
        />
      </v-col>
      <v-col cols="12">
        <v-text-field
          v-model="settings.default_ansible_winrm_server_cert_validation"
          :label="$t('deviceAnsibleWinrmCertValidation')"
          :disabled="saving"
          outlined
          dense
        />
      </v-col>
    </v-row>

    <div v-if="!hideActions" class="d-flex justify-end mt-3">
      <v-btn color="primary" depressed :loading="saving" @click="save">
        {{ $t('save') }}
      </v-btn>
    </div>
  </div>
</template>
<script>
import axios from 'axios';
import EventBus from '@/event-bus';
import { getErrorMessage } from '@/lib/error';

const ACTIONS = [
  { field: 'discover_template_id', label: 'deviceTemplateDiscover' },
  { field: 'restart_template_id', label: 'deviceTemplateRestart' },
  { field: 'status_template_id', label: 'deviceTemplateStatus' },
];

export default {
  props: {
    projectId: { type: [Number, String], required: true },
    hideActions: { type: Boolean, default: false },
  },

  data() {
    return {
      actions: ACTIONS,
      templates: [],
      settings: {
        discover_template_id: null,
        restart_template_id: null,
        status_template_id: null,
        default_config_json: '',
        default_ansible_user: '',
        default_ansible_password: '',
        default_ansible_connection: 'winrm',
        default_ansible_winrm_transport: 'basic',
        default_ansible_winrm_scheme: 'http',
        default_ansible_port: 5985,
        default_ansible_winrm_server_cert_validation: 'ignore',
      },
      saving: false,
      formError: null,
      showDefaultAnsiblePassword: false,
      defaultConfigItems: [],
      defaultConfigHeaders: [
        { text: this.$i18n.t('deviceConfigCategory'), value: 'category', width: '25%' },
        { text: this.$i18n.t('deviceConfigKey'), value: 'key', width: '30%' },
        { text: this.$i18n.t('deviceConfigValue'), value: 'value', width: '40%' },
        { value: 'actions', sortable: false, width: '5%' },
      ],
    };
  },

  async created() {
    await Promise.all([this.loadTemplates(), this.loadSettings()]);
  },

  methods: {
    async loadTemplates() {
      try {
        const res = await axios.get(`/api/project/${this.projectId}/templates`);
        this.templates = res.data || [];
      } catch (e) {
        this.formError = getErrorMessage(e);
      }
    },
    async loadSettings() {
      try {
        const res = await axios.get(`/api/project/${this.projectId}/devices/settings`);
        // copy fields the server returned, falling back to defaults
        this.settings = { ...this.settings, ...(res.data || {}) };
        this.defaultConfigItems = this.parseDefaultConfigJson(this.settings.default_config_json);
        this.showDefaultAnsiblePassword = false;
      } catch (e) {
        this.formError = getErrorMessage(e);
      }
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
    addDefaultConfigRow() {
      this.defaultConfigItems.push({ category: 'SystemConfig', key: '', value: '' });
    },
    removeDefaultConfigRow(index) {
      this.defaultConfigItems.splice(index, 1);
    },
    async save() {
      this.saving = true;
      this.formError = null;
      try {
        const payload = { ...this.settings };
        payload.default_config_json = this.buildDefaultConfigJson();
        payload.default_ansible_port = Number(payload.default_ansible_port) || 5985;
        await axios.put(`/api/project/${this.projectId}/devices/settings`, payload);
        EventBus.$emit('i-snackbar', { color: 'success', text: this.$i18n.t('deviceSettingsSaved') });
        this.$emit('saved');
        return true;
      } catch (e) {
        this.formError = getErrorMessage(e);
        EventBus.$emit('i-snackbar', { color: 'error', text: this.formError });
        this.$emit('save-failed', e);
        return false;
      } finally {
        this.saving = false;
      }
    },
  },
};
</script>
