<template>
  <div>
    <p class="text--secondary mb-2">
      Device types (profiles). Each type has its own templates and TDengine status table.
      Existing devices use the default <strong>NEWARE</strong> profile.
    </p>
    <v-alert v-if="error" type="error" dense class="mb-2">{{ error }}</v-alert>
    <v-progress-linear v-if="loading" indeterminate class="mb-4" />
    <v-expansion-panels v-else accordion>
      <v-expansion-panel v-for="p in profiles" :key="p.id">
        <v-expansion-panel-header>{{ p.name }} ({{ p.profile_key }})</v-expansion-panel-header>
        <v-expansion-panel-content>
          <DeviceProfileSettingsEditor
            :project-id="projectId"
            :profile-id="p.id"
            @saved="load"
          />
        </v-expansion-panel-content>
      </v-expansion-panel>
    </v-expansion-panels>
  </div>
</template>

<script>
import axios from 'axios';
import DeviceProfileSettingsEditor from '@/components/DeviceProfileSettingsEditor.vue';

export default {
  components: { DeviceProfileSettingsEditor },
  props: {
    projectId: { type: Number, required: true },
  },
  data() {
    return {
      profiles: [],
      loading: false,
      error: null,
    };
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
  },
};
</script>
