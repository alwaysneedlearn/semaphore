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
    <v-text-field
      v-model="settings.tdengine_status_table"
      label="TDengine status table"
      hint="NEWARE default: status"
      persistent-hint
      outlined
      dense
      :disabled="saving"
    />
    <v-text-field
      v-model.number="settings.status_refresh_interval_min"
      label="Status refresh interval (minutes)"
      type="number"
      min="0"
      outlined
      dense
      :disabled="saving"
    />
    <v-btn color="primary" small :loading="saving" @click="save">Save profile settings</v-btn>
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
      saving: false,
      actions: [
        { field: 'discover_template_id', label: 'Discover template' },
        { field: 'start_template_id', label: 'Start template' },
        { field: 'stop_template_id', label: 'Stop template' },
        { field: 'restart_template_id', label: 'Restart template' },
        { field: 'status_template_id', label: 'Status / Patrol template' },
        { field: 'config_template_id', label: 'Config template' },
      ],
    };
  },
  async created() {
    await Promise.all([this.loadSettings(), this.loadTemplates()]);
  },
  methods: {
    async loadTemplates() {
      const { data } = await axios.get(`/api/project/${this.projectId}/templates`);
      this.templates = data || [];
    },
    async loadSettings() {
      const { data } = await axios.get(
        `/api/project/${this.projectId}/devices/profiles/${this.profileId}/settings`,
      );
      this.settings = { ...data };
    },
    async save() {
      this.saving = true;
      try {
        await axios.put(
          `/api/project/${this.projectId}/devices/profiles/${this.profileId}/settings`,
          this.settings,
        );
        this.$emit('saved');
      } finally {
        this.saving = false;
      }
    },
  },
};
</script>
