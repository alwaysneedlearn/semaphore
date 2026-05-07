<template>
  <v-form
    v-if="item"
    ref="form"
    lazy-validation
    v-model="formValid"
  >
    <v-alert
      :value="formError"
      color="error"
      class="pb-2"
    >{{ formError }}
    </v-alert>

    <v-text-field
      v-model="item.ip_address"
      :label="$t('deviceIpAddress')"
      :rules="[v => !v || /^[0-9a-fA-F:.]+$/.test(v) || $t('deviceIpInvalid')]"
      :disabled="formSaving"
      outlined
      dense
      placeholder="10.0.0.5"
    ></v-text-field>

    <v-text-field
      v-model="item.hostname"
      :label="$t('deviceHostname')"
      :rules="[v => !!v || $t('deviceHostnameRequired')]"
      :disabled="formSaving"
      outlined
      dense
      placeholder="server01.example.com"
    ></v-text-field>

    <v-text-field
      v-model="item.ansible_user"
      :label="$t('deviceAnsibleUser')"
      :disabled="formSaving"
      outlined
      dense
      placeholder="winrmuser"
    ></v-text-field>

    <v-text-field
      v-model="item.ansible_password"
      :label="$t('deviceAnsiblePassword')"
      :disabled="formSaving"
      outlined
      dense
      type="password"
      placeholder="winrmpass"
    ></v-text-field>

    <v-text-field
      v-model="item.ansible_connection"
      :label="$t('deviceAnsibleConnection')"
      :disabled="formSaving"
      outlined
      dense
      placeholder="winrm"
    ></v-text-field>

    <v-text-field
      v-model="item.ansible_winrm_transport"
      :label="$t('deviceAnsibleWinrmTransport')"
      :disabled="formSaving"
      outlined
      dense
      placeholder="basic"
    ></v-text-field>

    <v-text-field
      v-model="item.ansible_winrm_scheme"
      :label="$t('deviceAnsibleWinrmScheme')"
      :disabled="formSaving"
      outlined
      dense
      placeholder="http"
    ></v-text-field>

    <v-text-field
      v-model.number="item.ansible_port"
      :label="$t('deviceAnsiblePort')"
      :disabled="formSaving"
      outlined
      dense
      type="number"
      min="1"
      placeholder="5985"
    ></v-text-field>

    <v-text-field
      v-model="item.ansible_winrm_server_cert_validation"
      :label="$t('deviceAnsibleWinrmCertValidation')"
      :disabled="formSaving"
      outlined
      dense
      placeholder="ignore"
    ></v-text-field>
  </v-form>
</template>
<script>
import ItemFormBase from '@/components/ItemFormBase';

export default {
  mixins: [ItemFormBase],

  methods: {
    getItemsUrl() {
      return `/api/project/${this.projectId}/devices`;
    },
    getSingleItemUrl() {
      return `/api/project/${this.projectId}/devices/${this.itemId}`;
    },
    getNewItem() {
      return {
        ip_address: '',
        hostname: '',
        ansible_user: '',
        ansible_password: '',
        ansible_connection: 'winrm',
        ansible_winrm_transport: 'basic',
        ansible_winrm_scheme: 'http',
        ansible_port: 5985,
        ansible_winrm_server_cert_validation: 'ignore',
        device_status: 'unknown',
      };
    },
  },
};
</script>
