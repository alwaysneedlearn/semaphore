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
        device_status: 'unknown',
      };
    },
  },
};
</script>
