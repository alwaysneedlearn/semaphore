<template>
  <div>
    <v-row dense>
      <v-col cols="12" md="6" v-for="cfg in actions" :key="cfg.field">
        <v-autocomplete
          v-model="settings[cfg.field]"
          :items="templates"
          item-value="id"
          item-text="name"
          :label="cfg.label"
          clearable
          outlined
          dense
          :disabled="saving"
        />
      </v-col>
    </v-row>
    <v-autocomplete
      v-model="settings.default_inventory_id"
      :items="inventories"
      item-value="id"
      item-text="name"
      label="Default inventory"
      clearable
      outlined
      dense
      class="mb-2"
      :disabled="saving"
    />
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

    <v-divider class="my-4" />
    <p class="text--secondary mb-2">默认配置（该类型下设备）</p>
    <v-data-table
      :headers="defaultConfigHeaders"
      :items="defaultConfigItems"
      dense
      hide-default-footer
      :items-per-page="-1"
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

    <v-btn color="primary" small class="mt-4" :loading="saving" @click="save">
      {{ $t('save') }}
    </v-btn>
  </div>
</template>

<script>
import axios from 'axios';

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
      defaultConfigItems: [],
      defaultConfigHeaders: [
        { text: this.$i18n.t('deviceConfigCategory'), value: 'category', width: '25%' },
        { text: this.$i18n.t('deviceConfigKey'), value: 'key', width: '30%' },
        { text: this.$i18n.t('deviceConfigValue'), value: 'value', width: '40%' },
        { value: 'actions', sortable: false, width: '5%' },
      ],
      actions: [
        { field: 'start_template_id', label: 'Start template' },
        { field: 'stop_template_id', label: 'Stop template' },
        { field: 'restart_template_id', label: 'Restart template' },
        { field: 'status_template_id', label: 'Status / Patrol template' },
        { field: 'check_restart_redeploy_template_id', label: 'Check-restart-redeploy template (for status refresh interval)' },
      ],
    };
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
    async loadSettings() {
      const { data } = await axios.get(
        `/api/project/${this.projectId}/devices/profiles/${this.profileId}/settings`,
      );
      this.settings = { ...data };
      this.defaultConfigItems = this.parseDefaultConfigJson(this.settings.default_config_json);
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
      try {
        const payload = { ...this.settings };
        payload.default_config_json = this.buildDefaultConfigJson();
        payload.status_refresh_interval_min = Number(payload.status_refresh_interval_min) || 0;
        delete payload.default_ansible_user;
        delete payload.default_ansible_password;
        delete payload.default_ansible_connection;
        delete payload.default_ansible_winrm_transport;
        delete payload.default_ansible_winrm_scheme;
        delete payload.default_ansible_port;
        delete payload.default_ansible_winrm_server_cert_validation;
        delete payload.config_template_id;
        delete payload.discover_template_id;
        await axios.put(
          `/api/project/${this.projectId}/devices/profiles/${this.profileId}/settings`,
          payload,
        );
        this.$emit('saved');
      } finally {
        this.saving = false;
      }
    },
  },
};
</script>
