<template>
  <div class="device-config-items-editor">
    <p v-if="hint" class="text--secondary caption mb-2">{{ hint }}</p>

    <div v-if="layout === 'cards'" class="device-config-items-editor__cards">
      <v-card
        v-for="(item, index) in items"
        :key="index"
        outlined
        class="device-config-items-editor__card mb-2"
      >
        <v-card-text class="py-3">
          <v-row dense>
            <v-col cols="12" sm="4">
              <v-text-field
                v-model="item.category"
                :label="$t('deviceConfigCategory')"
                hide-details="auto"
                dense
                outlined
                :disabled="disabled"
              />
            </v-col>
            <v-col cols="12" sm="8">
              <v-text-field
                v-model="item.key"
                :label="$t('deviceConfigKey')"
                hide-details="auto"
                dense
                outlined
                :disabled="disabled"
              />
            </v-col>
            <v-col cols="12">
              <v-textarea
                v-model="item.value"
                :label="$t('deviceConfigValue')"
                hide-details="auto"
                dense
                outlined
                auto-grow
                rows="1"
                :disabled="disabled"
              />
            </v-col>
            <v-col cols="12">
              <v-textarea
                v-model="item.remark"
                :label="$t('deviceConfigRemark')"
                :placeholder="$t('deviceConfigRemarkPlaceholder')"
                hide-details="auto"
                dense
                outlined
                auto-grow
                rows="1"
                :disabled="disabled"
              />
            </v-col>
          </v-row>
        </v-card-text>
        <v-card-actions class="pt-0 pb-2 px-3">
          <v-spacer />
          <v-btn
            icon
            small
            :title="$t('delete')"
            :disabled="disabled"
            @click="$emit('remove-row', index)"
          >
            <v-icon small>mdi-close</v-icon>
          </v-btn>
        </v-card-actions>
      </v-card>
      <v-alert v-if="items.length === 0" type="info" outlined dense class="mb-0">
        {{ $t('deviceConfigEmpty') }}
      </v-alert>
    </div>

    <div v-else class="device-config-items-editor__table-wrap">
      <v-data-table
        :headers="tableHeaders"
        :items="items"
        dense
        hide-default-footer
        :items-per-page="-1"
        class="elevation-0 device-config-items-editor__table"
        mobile-breakpoint="0"
      >
        <template v-slot:item.category="{ item }">
          <v-text-field
            v-model="item.category"
            hide-details
            dense
            outlined
            :disabled="disabled"
          />
        </template>
        <template v-slot:item.key="{ item }">
          <v-text-field
            v-model="item.key"
            hide-details
            dense
            outlined
            :disabled="disabled"
          />
        </template>
        <template v-slot:item.value="{ item }">
          <v-textarea
            v-model="item.value"
            hide-details
            dense
            outlined
            auto-grow
            rows="1"
            :disabled="disabled"
          />
        </template>
        <template v-slot:item.remark="{ item }">
          <v-textarea
            v-model="item.remark"
            hide-details
            dense
            outlined
            auto-grow
            rows="1"
            :placeholder="$t('deviceConfigRemarkPlaceholder')"
            :disabled="disabled"
          />
        </template>
        <template v-slot:item.actions="{ index }">
          <v-btn icon small :disabled="disabled" @click="$emit('remove-row', index)">
            <v-icon>mdi-close</v-icon>
          </v-btn>
        </template>
      </v-data-table>
    </div>

    <v-btn small text class="mt-2" :disabled="disabled" @click="$emit('add-row')">
      <v-icon left>mdi-plus</v-icon>
      {{ $t('deviceConfigAddRow') }}
    </v-btn>
  </div>
</template>

<script>
export default {
  props: {
    items: { type: Array, required: true },
    disabled: { type: Boolean, default: false },
    /** cards: stacked rows (profile editor); table: wide scrollable table (device dialog) */
    layout: {
      type: String,
      default: 'cards',
      validator: (v) => ['cards', 'table'].includes(v),
    },
    hint: { type: String, default: '' },
  },
  computed: {
    tableHeaders() {
      return [
        { text: this.$t('deviceConfigCategory'), value: 'category', width: '14%' },
        { text: this.$t('deviceConfigKey'), value: 'key', width: '18%' },
        { text: this.$t('deviceConfigValue'), value: 'value', width: '30%' },
        { text: this.$t('deviceConfigRemark'), value: 'remark', width: '33%' },
        { value: 'actions', sortable: false, width: '5%' },
      ];
    },
  },
};
</script>

<style scoped>
.device-config-items-editor__table-wrap {
  overflow-x: auto;
  margin: 0 -4px;
  padding: 0 4px;
}
.device-config-items-editor__table {
  min-width: 720px;
}
.device-config-items-editor__table >>> td {
  vertical-align: top;
  padding-top: 8px !important;
  padding-bottom: 8px !important;
}
.device-config-items-editor__card {
  background: rgba(0, 0, 0, 0.015);
}
</style>
