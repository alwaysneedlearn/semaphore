<template>
  <div>
    <v-toolbar flat>
      <v-btn icon class="mr-4" @click="goBack">
        <v-icon>mdi-arrow-left</v-icon>
      </v-btn>
      <v-toolbar-title>TDengine</v-toolbar-title>
      <v-spacer />
      <v-btn color="primary" :loading="saving" @click="save">Save</v-btn>
    </v-toolbar>
    <v-divider />
    <div style="max-width: 640px; margin: 24px auto; padding: 0 16px;">
      <v-alert v-if="error" type="error" dense class="mb-4">{{ error }}</v-alert>
      <v-alert v-if="testOk" type="success" dense class="mb-4">Connection OK</v-alert>
      <v-alert v-if="publishOk" type="success" dense class="mb-4">
        Published {{ publishResult.rows }} row(s) across {{ publishResult.tables }} table(s).
      </v-alert>
      <v-switch v-model="form.enabled" label="Enable TDengine" />
      <v-switch
        v-model="form.auto_sync_on_bulk"
        label="Auto-sync after playbook bulk status callback"
        hint="When off, use Publish snapshot below after patrol/start templates update the DB"
        persistent-hint
        :disabled="!form.enabled"
        class="mt-0"
      />
      <v-text-field
        v-model="form.url"
        label="REST URL"
        hint="Base URL only, e.g. http://10.40.81.130:6041 (do not add /rest/sql; default REST port is 6041)"
        persistent-hint
        outlined
        dense
      />
      <v-text-field v-model="form.user" label="User" outlined dense />
      <v-text-field
        v-model="form.password"
        label="Password"
        type="password"
        :placeholder="passwordPlaceholder"
        outlined
        dense
      />
      <v-text-field v-model="form.database" label="Database" outlined dense />
      <v-btn class="mt-2 mr-2" :loading="testing" @click="testConnection">Test connection</v-btn>
      <v-btn
        class="mt-2"
        :loading="publishing"
        :disabled="!form.enabled"
        @click="publishSnapshot"
      >
        Publish snapshot (all projects)
      </v-btn>
      <p class="text--secondary mt-6 text-caption">
        Device status is stored in Semaphore when playbooks call
        PUT …/devices/status/bulk. TDengine is optional: run Publish snapshot to copy
        the current DB state (healthy → online). Table names are set per device type
        under Project → Device types → TDengine status table. Create database/stable
        in TDengine before first publish.
      </p>
    </div>
  </div>
</template>

<script>
import axios from 'axios';
import EventBus from '@/event-bus';

export default {
  data() {
    return {
      form: {
        enabled: false,
        auto_sync_on_bulk: false,
        url: '',
        user: '',
        password: '',
        database: 'semaphore_devices',
      },
      passwordSet: false,
      saving: false,
      testing: false,
      publishing: false,
      testOk: false,
      publishOk: false,
      publishResult: { rows: 0, tables: 0 },
      error: null,
    };
  },
  computed: {
    passwordPlaceholder() {
      return this.passwordSet ? '(unchanged — leave blank to keep)' : '';
    },
  },
  async created() {
    await this.load();
  },
  methods: {
    goBack() {
      // Same as Runners/Tokens/Users toolbar: return to last project, not /users.
      EventBus.$emit('i-open-last-project');
    },
    async load() {
      this.error = null;
      try {
        const { data } = await axios.get('/api/admin/tdengine');
        this.form.enabled = !!data.enabled;
        this.form.auto_sync_on_bulk = !!data.auto_sync_on_bulk;
        this.form.url = data.url || '';
        this.form.user = data.user || '';
        this.form.database = data.database || 'semaphore_devices';
        this.passwordSet = data.password === '********';
        this.form.password = '';
      } catch (e) {
        this.error = e?.response?.data?.error || e.message;
      }
    },
    async save() {
      this.saving = true;
      this.error = null;
      this.testOk = false;
      this.publishOk = false;
      try {
        const body = { ...this.form };
        if (this.passwordSet && !body.password) {
          body.password = '********';
        }
        await axios.put('/api/admin/tdengine', body);
        await this.load();
      } catch (e) {
        this.error = e?.response?.data?.error || e.message;
      } finally {
        this.saving = false;
      }
    },
    async publishSnapshot() {
      this.publishing = true;
      this.error = null;
      this.publishOk = false;
      try {
        const { data } = await axios.post('/api/admin/tdengine/publish', {});
        this.publishResult = data.result || { rows: 0, tables: 0 };
        this.publishOk = true;
      } catch (e) {
        const d = e?.response?.data;
        this.error = d?.error || e.message;
        if (d?.result) {
          this.publishResult = d.result;
        }
      } finally {
        this.publishing = false;
      }
    },
    async testConnection() {
      this.testing = true;
      this.error = null;
      this.testOk = false;
      try {
        const body = { ...this.form };
        if (this.passwordSet && !body.password) {
          body.password = '********';
        }
        await axios.post('/api/admin/tdengine/test', body);
        this.testOk = true;
      } catch (e) {
        this.error = e?.response?.data?.error || e.message;
      } finally {
        this.testing = false;
      }
    },
  },
};
</script>
