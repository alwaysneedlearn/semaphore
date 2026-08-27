<template>
  <v-dialog v-model="dialog" :max-width="520" persistent>
    <v-card>
      <v-card-title class="d-flex align-center">
        <v-icon class="mr-2">mdi-database-sync</v-icon>
        {{ $t('deviceResendData') }}
      </v-card-title>
      <v-card-text>
        <v-alert type="warning" dense prominent class="mb-3">
          {{ $t('deviceActionRiskWarning') }}
        </v-alert>
        <p v-if="deviceSummary" class="caption mb-3">{{ deviceSummary }}</p>
        <v-alert v-if="mixedProfilesHint" type="warning" dense text class="mb-3">
          {{ mixedProfilesHint }}
        </v-alert>

        <v-row dense>
          <v-col cols="12" sm="6">
            <v-text-field
              v-model="form.start"
              :label="$t('deviceResendStart')"
              type="datetime-local"
              outlined
              dense
              hide-details="auto"
              :disabled="loading"
            />
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field
              v-model="form.end"
              :label="$t('deviceResendEnd')"
              type="datetime-local"
              outlined
              dense
              hide-details="auto"
              :disabled="loading"
            />
          </v-col>
        </v-row>
        <v-alert v-if="error" type="error" dense class="mt-3 mb-0">{{ error }}</v-alert>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn text :disabled="loading" @click="dialog = false">{{ $t('cancel') }}</v-btn>
        <v-btn color="primary" :loading="loading" @click="submit">{{ $t('confirmTask') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
function pad2(n) {
  return String(n).padStart(2, '0');
}

function toDatetimeLocalValue(d) {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

function toApiTime(localValue) {
  if (!localValue) {
    return '';
  }
  return `${localValue.replace('T', ' ')}:00`;
}

export default {
  props: {
    value: { type: Boolean, default: false },
    devices: { type: Array, default: () => [] },
    profileById: { type: Object, default: () => ({}) },
    loading: { type: Boolean, default: false },
  },
  data() {
    const end = new Date();
    const start = new Date(end.getTime() - 24 * 60 * 60 * 1000);
    return {
      form: {
        start: toDatetimeLocalValue(start),
        end: toDatetimeLocalValue(end),
      },
      error: null,
    };
  },
  computed: {
    dialog: {
      get() { return this.value; },
      set(v) { this.$emit('input', v); },
    },
    profileKeys() {
      const keys = new Set();
      (this.devices || []).forEach((d) => {
        const k = this.profileById[d.device_profile_id];
        if (k) keys.add(String(k).toUpperCase());
      });
      return [...keys];
    },
    deviceSummary() {
      const n = (this.devices || []).length;
      if (n === 0) return '';
      if (n === 1) {
        const d = this.devices[0];
        return `${d.hostname || '—'} (${d.ip_address || '—'})`;
      }
      return this.$t('deviceResendBulkSummary', { count: n });
    },
    mixedProfilesHint() {
      if (this.profileKeys.length <= 1) {
        return '';
      }
      return this.$t('deviceResendMixedProfiles', { types: this.profileKeys.join(', ') });
    },
  },
  watch: {
    value(open) {
      if (open) {
        this.error = null;
        const end = new Date();
        const start = new Date(end.getTime() - 24 * 60 * 60 * 1000);
        this.form.start = toDatetimeLocalValue(start);
        this.form.end = toDatetimeLocalValue(end);
      }
    },
  },
  methods: {
    submit() {
      this.error = null;
      if (!this.form.start || !this.form.end) {
        this.error = this.$t('deviceResendTimeRequired');
        return;
      }
      const startMs = new Date(this.form.start).getTime();
      const endMs = new Date(this.form.end).getTime();
      if (Number.isNaN(startMs) || Number.isNaN(endMs)) {
        this.error = this.$t('deviceResendTimeInvalid');
        return;
      }
      if (endMs < startMs) {
        this.error = this.$t('deviceResendEndBeforeStart');
        return;
      }
      this.$emit('submit', {
        start: toApiTime(this.form.start),
        end: toApiTime(this.form.end),
      });
    },
  },
};
</script>
