<template>
  <v-dialog
    v-model="dialog"
    :max-width="900"
    persistent
  >
    <v-card v-if="dialog">
      <v-card-title>
        <v-icon class="mr-2">mdi-cog</v-icon>
        {{ $t('deviceConfig') }} &mdash; {{ deviceName }}
      </v-card-title>

      <v-card-text>
        <v-alert v-if="formError" color="error" dense>{{ formError }}</v-alert>

        <p class="text--secondary mb-2">{{ $t('deviceConfigHelp') }}</p>

        <v-data-table
          :headers="headers"
          :items="items"
          dense
          hide-default-footer
          :items-per-page="Number.MAX_VALUE"
        >
          <template v-slot:item.category="{ item }">
            <v-text-field
              v-model="item.category"
              hide-details
              dense
              outlined
              placeholder="default"
            />
          </template>
          <template v-slot:item.key="{ item }">
            <v-text-field
              v-model="item.key"
              hide-details
              dense
              outlined
              :rules="[v => !!v || $t('keyIsRequired')]"
            />
          </template>
          <template v-slot:item.value="{ item }">
            <v-text-field
              v-model="item.value"
              hide-details
              dense
              outlined
            />
          </template>
          <template v-slot:item.remark="{ item }">
            <v-text-field
              v-model="item.remark"
              hide-details
              dense
              outlined
              :placeholder="$t('deviceConfigRemarkPlaceholder')"
            />
          </template>
          <template v-slot:item.actions="{ index }">
            <v-btn icon small @click="removeRow(index)">
              <v-icon>mdi-close</v-icon>
            </v-btn>
          </template>
        </v-data-table>

        <v-btn small text class="mt-2" @click="addRow">
          <v-icon left>mdi-plus</v-icon>
          {{ $t('deviceConfigAddRow') }}
        </v-btn>
      </v-card-text>

      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn text @click="dialog = false" :disabled="saving">
          {{ $t('cancel') }}
        </v-btn>
        <v-btn color="primary" depressed :loading="saving" @click="save">
          {{ $t('save') }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>
<script>
import axios from 'axios';
import { getErrorMessage } from '@/lib/error';

export default {
  props: {
    value: Boolean,
    projectId: [Number, String],
    deviceId: [Number, String],
    deviceName: String,
  },

  data() {
    return {
      items: [],
      saving: false,
      formError: null,
      headers: [
        { text: this.$i18n.t('deviceConfigCategory'), value: 'category', width: '18%' },
        { text: this.$i18n.t('deviceConfigKey'), value: 'key', width: '22%' },
        { text: this.$i18n.t('deviceConfigValue'), value: 'value', width: '28%' },
        { text: this.$i18n.t('deviceConfigRemark'), value: 'remark', width: '27%' },
        { value: 'actions', sortable: false, width: '5%' },
      ],
    };
  },

  computed: {
    dialog: {
      get() { return this.value; },
      set(v) { this.$emit('input', v); },
    },
  },

  watch: {
    async value(open) {
      if (open && this.deviceId) {
        await this.loadItems();
      }
    },
  },

  methods: {
    async loadItems() {
      this.formError = null;
      try {
        const res = await axios.get(`/api/project/${this.projectId}/devices/${this.deviceId}/config`);
        this.items = (res.data || []).map((it) => ({ ...it }));
      } catch (e) {
        this.formError = getErrorMessage(e);
      }
    },

    addRow() {
      this.items.push({ category: 'default', key: '', value: '', remark: '' });
    },

    removeRow(index) {
      this.items.splice(index, 1);
    },

    async save() {
      this.saving = true;
      this.formError = null;
      try {
        const payload = this.items
          .filter((it) => it.key && it.key.trim() !== '')
          .map((it) => ({
            category: (it.category || '').trim(),
            key: it.key.trim(),
            value: it.value || '',
            remark: (it.remark || '').trim(),
          }));
        await axios.put(
          `/api/project/${this.projectId}/devices/${this.deviceId}/config`,
          payload,
        );
        this.$emit('saved');
        this.dialog = false;
      } catch (e) {
        this.formError = getErrorMessage(e);
      } finally {
        this.saving = false;
      }
    },
  },
};
</script>
