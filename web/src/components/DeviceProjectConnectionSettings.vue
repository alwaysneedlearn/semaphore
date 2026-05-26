<template>
  <div class="mb-6">
    <p class="subtitle-2 mb-1">{{ $t('deviceConnectionDefaultsTitle') }}</p>
    <p class="text--secondary mb-3">{{ $t('deviceConnectionDefaultsHelp') }}</p>
    <v-row dense>
      <v-col cols="12" md="6">
        <v-text-field
          v-model="form.default_ansible_user"
          :label="$t('deviceAnsibleUser')"
          outlined
          dense
          :disabled="saving"
        />
      </v-col>
      <v-col cols="12" md="6">
        <v-text-field
          v-model="form.default_ansible_password"
          :label="$t('deviceAnsiblePassword')"
          outlined
          dense
          :type="showPassword ? 'text' : 'password'"
          autocomplete="new-password"
          :append-icon="showPassword ? 'mdi-eye-off' : 'mdi-eye'"
          @click:append="showPassword = !showPassword"
          :disabled="saving"
        />
      </v-col>
      <v-col cols="12" md="6">
        <v-text-field
          v-model="form.default_ansible_connection"
          :label="$t('deviceAnsibleConnection')"
          outlined
          dense
          :disabled="saving"
        />
      </v-col>
      <v-col cols="12" md="6">
        <v-text-field
          v-model="form.default_ansible_winrm_transport"
          :label="$t('deviceAnsibleWinrmTransport')"
          outlined
          dense
          :disabled="saving"
        />
      </v-col>
      <v-col cols="12" md="6">
        <v-text-field
          v-model="form.default_ansible_winrm_scheme"
          :label="$t('deviceAnsibleWinrmScheme')"
          outlined
          dense
          :disabled="saving"
        />
      </v-col>
      <v-col cols="12" md="6">
        <v-text-field
          v-model.number="form.default_ansible_port"
          :label="$t('deviceAnsiblePort')"
          type="number"
          outlined
          dense
          :disabled="saving"
        />
      </v-col>
      <v-col cols="12">
        <v-text-field
          v-model="form.default_ansible_winrm_server_cert_validation"
          :label="$t('deviceAnsibleWinrmCertValidation')"
          outlined
          dense
          :disabled="saving"
        />
      </v-col>
    </v-row>
    <v-btn color="primary" small :loading="saving" @click="save">
      {{ $t('save') }}
    </v-btn>
  </div>
</template>

<script>
import axios from 'axios';
import EventBus from '@/event-bus';
import { getErrorMessage } from '@/lib/error';

const emptyForm = () => ({
  default_ansible_user: '',
  default_ansible_password: '',
  default_ansible_connection: 'winrm',
  default_ansible_winrm_transport: 'ntlm',
  default_ansible_winrm_scheme: 'http',
  default_ansible_port: 5985,
  default_ansible_winrm_server_cert_validation: 'ignore',
});

export default {
  props: {
    projectId: { type: Number, required: true },
  },
  data() {
    return {
      form: emptyForm(),
      saving: false,
      showPassword: false,
    };
  },
  async created() {
    await this.load();
  },
  methods: {
    async load() {
      const { data } = await axios.get(
        `/api/project/${this.projectId}/devices/settings/connection`,
      );
      this.form = {
        ...emptyForm(),
        ...data,
      };
      this.form.default_ansible_port = Number(this.form.default_ansible_port) || 5985;
      this.showPassword = false;
    },
    async save() {
      this.saving = true;
      try {
        const payload = {
          ...this.form,
          default_ansible_port: Number(this.form.default_ansible_port) || 5985,
        };
        await axios.put(
          `/api/project/${this.projectId}/devices/settings/connection`,
          payload,
        );
        EventBus.$emit('i-snackbar', { color: 'success', text: this.$t('save') });
        this.$emit('saved');
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.saving = false;
      }
    },
  },
};
</script>
