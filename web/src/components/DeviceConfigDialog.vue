<template>
  <v-dialog
    v-model="dialog"
    :max-width="1000"
    persistent
  >
    <v-card v-if="dialog">
      <v-card-title>
        <v-icon class="mr-2">mdi-cog</v-icon>
        {{ $t('deviceConfig') }} &mdash; {{ deviceName }}
      </v-card-title>

      <v-card-text>
        <v-alert v-if="formError" color="error" dense>{{ formError }}</v-alert>

        <DeviceConfigItemsEditor
          :items="items"
          :disabled="saving"
          layout="table"
          :hint="$t('deviceConfigHelp')"
          @add-row="addRow"
          @remove-row="removeRow"
        />
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
import DeviceConfigItemsEditor from '@/components/DeviceConfigItemsEditor.vue';

export default {
  components: { DeviceConfigItemsEditor },
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
      this.items.push({
        category: 'default', key: '', value: '', remark: '',
      });
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
