<template>
  <div>
    <p class="text--secondary mb-2">
      Configure templates, connection defaults, and TDengine table per device type.
      <strong>NEWARE</strong> is the default type for existing devices.
    </p>
    <v-btn
      color="primary"
      small
      class="mb-4"
      :disabled="loading"
      @click="addDialog = true"
    >
      <v-icon left small>mdi-plus</v-icon>
      Add device type
    </v-btn>
    <v-dialog v-model="addDialog" max-width="480">
      <v-card>
        <v-card-title>Add device type</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="newProfileKey"
            label="Profile key"
            hint="Uppercase identifier, e.g. ACME"
            persistent-hint
            outlined
            dense
          />
          <v-text-field
            v-model="newProfileName"
            label="Display name"
            outlined
            dense
          />
          <v-alert v-if="addError" type="error" dense>{{ addError }}</v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text @click="addDialog = false">Cancel</v-btn>
          <v-btn color="primary" :loading="adding" @click="createProfile">Create</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
    <v-alert v-if="error" type="error" dense class="mb-2">{{ error }}</v-alert>
    <v-progress-linear v-if="loading" indeterminate class="mb-4" />
    <v-expansion-panels v-if="!loading" accordion>
      <v-expansion-panel v-for="p in profiles" :key="p.id">
        <v-expansion-panel-header>
          <span>{{ p.name }} ({{ p.profile_key }})</span>
          <v-chip x-small class="ml-2" v-if="p.device_count > 0">
            {{ p.device_count }} device(s)
          </v-chip>
          <v-spacer />
          <v-btn
            icon
            small
            :disabled="p.device_count > 0 || deleting"
            :title="p.device_count > 0
              ? 'Remove or reassign all devices before deleting this type'
              : 'Delete device type'"
            @click.stop="askDeleteProfile(p)"
          >
            <v-icon small>mdi-delete</v-icon>
          </v-btn>
        </v-expansion-panel-header>
        <v-expansion-panel-content>
          <DeviceProfileSettingsEditor
            :project-id="projectId"
            :profile-id="p.id"
            @saved="load"
          />
        </v-expansion-panel-content>
      </v-expansion-panel>
    </v-expansion-panels>

    <YesNoDialog
      title="Delete device type"
      :text="deleteConfirmText"
      v-model="deleteDialog"
      @yes="deleteProfileConfirmed()"
    />
  </div>
</template>

<script>
import axios from 'axios';
import DeviceProfileSettingsEditor from '@/components/DeviceProfileSettingsEditor.vue';
import YesNoDialog from '@/components/YesNoDialog.vue';

export default {
  components: { DeviceProfileSettingsEditor, YesNoDialog },
  props: {
    projectId: { type: Number, required: true },
  },
  data() {
    return {
      profiles: [],
      loading: false,
      error: null,
      addDialog: false,
      newProfileKey: '',
      newProfileName: '',
      adding: false,
      addError: null,
      deleteDialog: false,
      profileToDelete: null,
      deleting: false,
    };
  },
  computed: {
    deleteConfirmText() {
      if (!this.profileToDelete) {
        return '';
      }
      return `Delete device type "${this.profileToDelete.name}" (${this.profileToDelete.profile_key})?`;
    },
  },
  async created() {
    await this.load();
  },
  methods: {
    async load() {
      this.loading = true;
      this.error = null;
      try {
        const { data } = await axios.get(`/api/project/${this.projectId}/devices/profiles`);
        this.profiles = data || [];
      } catch (e) {
        this.error = e?.response?.data?.error || e.message;
      } finally {
        this.loading = false;
      }
    },
    askDeleteProfile(p) {
      if (p.device_count > 0) {
        return;
      }
      this.profileToDelete = p;
      this.deleteDialog = true;
    },
    async deleteProfileConfirmed() {
      const p = this.profileToDelete;
      this.profileToDelete = null;
      if (!p) {
        return;
      }
      this.deleting = true;
      this.error = null;
      try {
        await axios.delete(`/api/project/${this.projectId}/devices/profiles/${p.id}`);
        await this.load();
      } catch (e) {
        this.error = e?.response?.data?.error || e.message;
      } finally {
        this.deleting = false;
      }
    },
    async createProfile() {
      this.adding = true;
      this.addError = null;
      try {
        await axios.post(`/api/project/${this.projectId}/devices/profiles`, {
          profile_key: this.newProfileKey,
          name: this.newProfileName,
        });
        this.addDialog = false;
        this.newProfileKey = '';
        this.newProfileName = '';
        await this.load();
      } catch (e) {
        this.addError = e?.response?.data?.error || e.message;
      } finally {
        this.adding = false;
      }
    },
  },
};
</script>
