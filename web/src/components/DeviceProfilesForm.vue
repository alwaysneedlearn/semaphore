<template>
  <div class="device-profiles-form">
    <v-card outlined class="mb-4">
      <v-card-title class="text-h6 py-3">
        <v-icon left color="primary">mdi-shape-outline</v-icon>
        Device types
      </v-card-title>
      <v-card-text class="pt-0">
        <p class="text--secondary body-2 mb-0">
          Bind playbook templates and refresh policy per device type.
          TDengine credentials stay on template Variable Groups
          (<code>docs/tdengine-setup.md</code>).
          <strong>NEWARE</strong> is the default type for existing devices.
        </p>
      </v-card-text>
    </v-card>

    <DeviceProjectConnectionSettings :project-id="projectId" class="mb-4" />

    <div class="d-flex align-center mb-3">
      <v-btn color="primary" depressed :disabled="loading" @click="addDialog = true">
        <v-icon left>mdi-plus</v-icon>
        Add device type
      </v-btn>
      <v-spacer />
      <v-btn icon :loading="loading" title="Refresh list" @click="load">
        <v-icon>mdi-refresh</v-icon>
      </v-btn>
    </div>

    <v-dialog v-model="addDialog" max-width="480">
      <v-card>
        <v-card-title class="text-h6">Add device type</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="newProfileKey"
            label="Profile key"
            hint="Uppercase identifier, e.g. LAND"
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
          <v-alert v-if="addError" type="error" dense class="mt-2">{{ addError }}</v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text @click="addDialog = false">Cancel</v-btn>
          <v-btn color="primary" depressed :loading="adding" @click="createProfile">Create</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-alert v-if="error" type="error" dense dismissible class="mb-3" @input="error = null">
      {{ error }}
    </v-alert>

    <v-progress-linear v-if="loading" indeterminate class="mb-4" />

    <v-expansion-panels v-if="!loading && profiles.length > 0" accordion multiple>
      <v-expansion-panel v-for="p in profiles" :key="p.id">
        <v-expansion-panel-header class="profile-panel-header">
          <div class="d-flex align-center flex-grow-1 pr-2">
            <v-icon class="mr-2" color="primary">mdi-layers</v-icon>
            <div>
              <div class="font-weight-medium">{{ p.name }}</div>
              <div class="text--secondary caption">{{ p.profile_key }}</div>
            </div>
            <v-chip x-small class="ml-3" color="primary" outlined>
              {{ p.device_count }} device(s)
            </v-chip>
          </div>
          <v-btn
            icon
            small
            class="mr-2"
            :disabled="p.device_count > 0 || deleting"
            :title="p.device_count > 0
              ? 'Remove or reassign all devices before deleting this type'
              : 'Delete device type'"
            @click.stop="askDeleteProfile(p)"
          >
            <v-icon small>mdi-delete</v-icon>
          </v-btn>
        </v-expansion-panel-header>
        <v-expansion-panel-content class="profile-panel-content">
          <DeviceProfileSettingsEditor
            :project-id="projectId"
            :profile-id="p.id"
            @saved="load"
          />
        </v-expansion-panel-content>
      </v-expansion-panel>
    </v-expansion-panels>

    <v-alert v-else-if="!loading" type="info" outlined dense class="mt-2">
      No device types yet. Click <strong>Add device type</strong> to create one.
    </v-alert>

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
import DeviceProjectConnectionSettings from '@/components/DeviceProjectConnectionSettings.vue';
import YesNoDialog from '@/components/YesNoDialog.vue';

export default {
  components: { DeviceProfileSettingsEditor, DeviceProjectConnectionSettings, YesNoDialog },
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

<style scoped>
.profile-panel-header {
  min-height: 56px;
}
.profile-panel-content {
  background: rgba(0, 0, 0, 0.02);
  padding-top: 8px !important;
}
</style>
