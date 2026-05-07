<template>
  <div>
    <v-alert v-if="formError" color="error" dense class="mb-2">{{ formError }}</v-alert>

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

    <div class="d-flex justify-end mt-3">
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
  { field: 'start_template_id', label: 'deviceTemplateStart' },
  { field: 'stop_template_id', label: 'deviceTemplateStop' },
  { field: 'restart_template_id', label: 'deviceTemplateRestart' },
  { field: 'status_template_id', label: 'deviceTemplateStatus' },
  { field: 'config_template_id', label: 'deviceTemplateConfig' },
];

export default {
  props: {
    projectId: { type: [Number, String], required: true },
  },

  data() {
    return {
      actions: ACTIONS,
      templates: [],
      settings: {
        discover_template_id: null,
        start_template_id: null,
        stop_template_id: null,
        restart_template_id: null,
        status_template_id: null,
        config_template_id: null,
        status_refresh_interval_min: 0,
      },
      saving: false,
      formError: null,
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
        if (this.settings.status_refresh_interval_min == null) {
          this.settings.status_refresh_interval_min = 0;
        }
      } catch (e) {
        this.formError = getErrorMessage(e);
      }
    },
    async save() {
      this.saving = true;
      this.formError = null;
      try {
        const payload = { ...this.settings };
        payload.status_refresh_interval_min = Number(payload.status_refresh_interval_min) || 0;
        await axios.put(`/api/project/${this.projectId}/devices/settings`, payload);
        EventBus.$emit('i-snackbar', { color: 'success', text: this.$i18n.t('deviceSettingsSaved') });
      } catch (e) {
        this.formError = getErrorMessage(e);
        EventBus.$emit('i-snackbar', { color: 'error', text: this.formError });
      } finally {
        this.saving = false;
      }
    },
  },
};
</script>
