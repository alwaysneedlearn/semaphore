<template>
  <div>
    <v-toolbar flat>
      <v-btn icon class="mr-4" @click="$router.push('/users')">
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
      <v-switch v-model="form.enabled" label="Enable TDengine status sync" />
      <v-text-field v-model="form.url" label="REST URL" hint="e.g. http://127.0.0.1:6041" persistent-hint outlined dense />
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
      <v-btn class="mt-2" :loading="testing" @click="testConnection">Test connection</v-btn>
      <p class="text--secondary mt-6 text-caption">
        After playbook bulk callback updates the DB, Semaphore writes a full
        project snapshot per device profile (healthy → online, other → offline).
        Table names are set per device profile under Project Settings.
      </p>
    </div>
  </div>
</template>

<script>
import axios from 'axios';

export default {
  data() {
    return {
      form: {
        enabled: false,
        url: '',
        user: '',
        password: '',
        database: 'semaphore_devices',
      },
      passwordSet: false,
      saving: false,
      testing: false,
      testOk: false,
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
    async load() {
      this.error = null;
      try {
        const { data } = await axios.get('/api/admin/tdengine');
        this.form.enabled = !!data.enabled;
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
    async testConnection() {
      this.testing = true;
      this.error = null;
      this.testOk = false;
      try {
        await axios.post('/api/admin/tdengine/test');
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
