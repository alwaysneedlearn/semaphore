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
      v-model="item.rdp_user"
      :label="$t('deviceRdpUser')"
      :disabled="formSaving"
      outlined
      dense
      clearable
      @click:clear="item.rdp_user = ''"
    ></v-text-field>

    <v-text-field
      v-model="item.rdp_password"
      :label="$t('deviceRdpPassword')"
      :disabled="formSaving"
      outlined
      dense
      :type="showRdpPassword ? 'text' : 'password'"
      autocomplete="new-password"
      clearable
      :append-icon="showRdpPassword ? 'mdi-eye-off' : 'mdi-eye'"
      @click:append="showRdpPassword = !showRdpPassword"
      @click:clear="item.rdp_password = ''"
    ></v-text-field>

    <v-text-field
      v-model.number="item.rdp_port"
      :label="$t('deviceRdpPort')"
      :disabled="formSaving"
      outlined
      dense
      type="number"
      min="1"
      max="65535"
      placeholder="3389"
    ></v-text-field>

    <v-text-field
      v-model="item.ansible_user"
      :label="$t('deviceAnsibleUser')"
      :disabled="formSaving"
      outlined
      dense
      clearable
      placeholder="winrmuser"
      @click:clear="item.ansible_user = ''"
    ></v-text-field>

    <v-text-field
      v-model="item.ansible_password"
      :label="$t('deviceAnsiblePassword')"
      :disabled="formSaving"
      outlined
      dense
      :type="showAnsiblePassword ? 'text' : 'password'"
      autocomplete="new-password"
      clearable
      placeholder="winrmpass"
      :append-icon="showAnsiblePassword ? 'mdi-eye-off' : 'mdi-eye'"
      @click:append="showAnsiblePassword = !showAnsiblePassword"
      @click:clear="item.ansible_password = ''"
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

  data() {
    return {
      showAnsiblePassword: false,
      showRdpPassword: false,
    };
  },

  methods: {
    beforeSave() {
      if (!this.item) {
        return;
      }
      // Cleared WinRM fields must be '' (not undefined) so PUT persists clears.
      this.item.ansible_user = this.item.ansible_user == null
        ? ''
        : String(this.item.ansible_user).trim();
      this.item.ansible_password = this.item.ansible_password == null
        ? ''
        : String(this.item.ansible_password).trim();
      this.item.rdp_user = this.item.rdp_user == null ? '' : String(this.item.rdp_user).trim();
      this.item.rdp_password = this.item.rdp_password == null
        ? ''
        : String(this.item.rdp_password).trim();
      const rp = parseInt(this.item.rdp_port, 10);
      if (!Number.isFinite(rp) || rp < 1 || rp > 65535) {
        this.item.rdp_port = 3389;
      } else {
        this.item.rdp_port = rp;
      }
    },
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
        rdp_user: '',
        rdp_password: '',
        ansible_user: '',
        ansible_password: '',
        ansible_connection: 'winrm',
        ansible_winrm_transport: 'basic',
        ansible_winrm_scheme: 'http',
        ansible_port: 5985,
        rdp_port: 3389,
        ansible_winrm_server_cert_validation: 'ignore',
        device_status: 'unhealthy',
      };
    },
    afterLoadData() {
      this.showAnsiblePassword = false;
      this.showRdpPassword = false;
      if (this.item && (this.item.rdp_port == null || this.item.rdp_port === '')) {
        this.item.rdp_port = 3389;
      }
    },
  },
};
</script>
