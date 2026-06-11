<template>
  <v-dialog v-model="open" max-width="900" scrollable>
    <v-card>
      <v-card-title>{{ $t('deviceImportExportTitle') }}</v-card-title>
      <v-card-text>
        <v-tabs v-model="tab" class="mb-4">
          <v-tab>{{ $t('deviceExport') }}</v-tab>
          <v-tab>{{ $t('deviceImport') }}</v-tab>
        </v-tabs>

        <v-tabs-items v-model="tab">
          <v-tab-item>
            <p class="text-body-2 mb-4">{{ $t('deviceExportHelp') }}</p>
            <v-radio-group v-model="exportScope" row dense>
              <v-radio
                :label="$t('deviceExportAll')"
                value="all"
              />
              <v-radio
                :label="$t('deviceExportSelected', { count: selectedCount })"
                value="selected"
                :disabled="selectedCount === 0"
              />
            </v-radio-group>
            <v-radio-group v-model="exportFormat" row dense class="mt-0">
              <v-radio label="JSON" value="json" />
              <v-radio label="CSV" value="csv" />
            </v-radio-group>
            <v-btn
              color="primary"
              depressed
              :loading="exporting"
              @click="runExport"
            >
              <v-icon left>mdi-download</v-icon>
              {{ $t('deviceExport') }}
            </v-btn>
          </v-tab-item>

          <v-tab-item>
            <p class="text-body-2 mb-4">{{ $t('deviceImportHelp') }}</p>
            <v-file-input
              v-model="importFile"
              accept=".json,.csv,application/json,text/csv"
              :label="$t('deviceImportFile')"
              outlined
              dense
              show-size
              prepend-icon="mdi-file-upload"
              @change="onImportFileChange"
            />
            <v-text-field
              v-model="defaultProfileKey"
              :label="$t('deviceImportDefaultProfileKey')"
              :hint="$t('deviceImportDefaultProfileKeyHint')"
              persistent-hint
              outlined
              dense
              clearable
              class="mt-2"
            />
            <v-alert v-if="importParseError" type="error" dense class="mt-2">
              {{ importParseError }}
            </v-alert>
            <v-alert v-if="previewRows.length" type="info" dense class="mt-2">
              {{ $t('deviceImportPreviewCount', { count: previewRows.length }) }}
            </v-alert>
            <v-data-table
              v-if="previewRows.length"
              :headers="previewHeaders"
              :items="previewRows.slice(0, 50)"
              dense
              hide-default-footer
              class="mt-2 elevation-0"
            />
            <div v-if="previewRows.length > 50" class="text-caption mt-1">
              {{ $t('deviceImportPreviewTruncated', { count: previewRows.length }) }}
            </div>
            <v-btn
              class="mt-4"
              color="primary"
              depressed
              :loading="importing"
              :disabled="!previewRows.length"
              @click="runImport"
            >
              <v-icon left>mdi-upload</v-icon>
              {{ $t('deviceImport') }}
            </v-btn>
          </v-tab-item>
        </v-tabs-items>

        <v-alert v-if="resultMessage" :type="resultType" dense class="mt-4">
          {{ resultMessage }}
        </v-alert>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn text @click="open = false">{{ $t('close') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
import axios from 'axios';
import { getErrorMessage } from '@/lib/error';

function parseCSVLine(line) {
  const out = [];
  let cur = '';
  let inQuotes = false;
  for (let i = 0; i < line.length; i += 1) {
    const ch = line[i];
    if (inQuotes) {
      if (ch === '"') {
        if (line[i + 1] === '"') {
          cur += '"';
          i += 1;
        } else {
          inQuotes = false;
        }
      } else {
        cur += ch;
      }
    } else if (ch === '"') {
      inQuotes = true;
    } else if (ch === ',') {
      out.push(cur);
      cur = '';
    } else {
      cur += ch;
    }
  }
  out.push(cur);
  return out;
}

function parseCSV(text) {
  const lines = String(text || '').split(/\r?\n/).filter((l) => l.trim() !== '');
  if (!lines.length) {
    return [];
  }
  const headers = parseCSVLine(lines[0]).map((h) => h.trim());
  const rows = [];
  for (let i = 1; i < lines.length; i += 1) {
    const cells = parseCSVLine(lines[i]);
    const row = {};
    headers.forEach((h, idx) => {
      row[h] = cells[idx] != null ? cells[idx] : '';
    });
    rows.push(normalizeImportRow(row));
  }
  return rows;
}

function normalizeImportRow(row) {
  const out = { ...row };
  const intFields = ['rdp_port', 'ansible_port', 'api_port', 'device_profile_id'];
  intFields.forEach((f) => {
    if (out[f] == null || out[f] === '') {
      return;
    }
    const n = parseInt(out[f], 10);
    if (Number.isFinite(n)) {
      out[f] = n;
    }
  });
  return out;
}

function parseImportPayload(text, filename) {
  const name = (filename || '').toLowerCase();
  if (name.endsWith('.csv')) {
    return parseCSV(text);
  }
  const trimmed = String(text || '').trim();
  if (!trimmed) {
    return [];
  }
  const data = JSON.parse(trimmed);
  if (Array.isArray(data)) {
    return data.map(normalizeImportRow);
  }
  if (data && Array.isArray(data.devices)) {
    return data.devices.map(normalizeImportRow);
  }
  throw new Error('Invalid JSON: expected { devices: [...] } or an array');
}

export default {
  name: 'DeviceImportExportDialog',

  props: {
    value: { type: Boolean, default: false },
    projectId: { type: [Number, String], required: true },
    selectedDeviceIds: { type: Array, default: () => [] },
  },

  data() {
    return {
      tab: 0,
      exportScope: 'all',
      exportFormat: 'json',
      exporting: false,
      importFile: null,
      defaultProfileKey: '',
      previewRows: [],
      importParseError: '',
      importing: false,
      resultMessage: '',
      resultType: 'info',
    };
  },

  computed: {
    open: {
      get() {
        return this.value;
      },
      set(v) {
        this.$emit('input', v);
      },
    },
    selectedCount() {
      return (this.selectedDeviceIds || []).length;
    },
    previewHeaders() {
      return [
        { text: this.$t('deviceIpAddress'), value: 'ip_address' },
        { text: this.$t('deviceHostname'), value: 'hostname' },
        { text: 'profile_key', value: 'profile_key' },
      ];
    },
  },

  watch: {
    value(v) {
      if (v) {
        this.resultMessage = '';
        this.importParseError = '';
        if (this.selectedCount > 0) {
          this.exportScope = 'selected';
        }
      }
    },
  },

  methods: {
    devicesUrl() {
      return `/api/project/${this.projectId}/devices`;
    },

    async runExport() {
      this.exporting = true;
      this.resultMessage = '';
      try {
        const params = { format: this.exportFormat };
        if (this.exportScope === 'selected' && this.selectedCount > 0) {
          params.ids = this.selectedDeviceIds.join(',');
        }
        const res = await axios.get(`${this.devicesUrl()}/export`, {
          params,
          responseType: this.exportFormat === 'csv' ? 'blob' : 'json',
        });
        const ext = this.exportFormat === 'csv' ? 'csv' : 'json';
        const mime = this.exportFormat === 'csv' ? 'text/csv' : 'application/json';
        let blob;
        if (this.exportFormat === 'csv') {
          blob = res.data;
        } else {
          blob = new Blob([JSON.stringify(res.data, null, 2)], { type: mime });
        }
        const a = document.createElement('a');
        a.href = URL.createObjectURL(blob);
        a.download = `devices_${this.projectId}_${Date.now()}.${ext}`;
        a.click();
        URL.revokeObjectURL(a.href);
        this.resultType = 'success';
        this.resultMessage = this.$t('deviceExportDone');
      } catch (e) {
        this.resultType = 'error';
        this.resultMessage = getErrorMessage(e);
      } finally {
        this.exporting = false;
      }
    },

    async onImportFileChange(file) {
      this.importParseError = '';
      this.previewRows = [];
      this.resultMessage = '';
      if (!file) {
        return;
      }
      try {
        const text = await file.text();
        this.previewRows = parseImportPayload(text, file.name);
        if (!this.previewRows.length) {
          this.importParseError = this.$t('deviceImportEmpty');
        }
      } catch (e) {
        this.importParseError = e.message || String(e);
      }
    },

    async runImport() {
      if (!this.previewRows.length) {
        return;
      }
      this.importing = true;
      this.resultMessage = '';
      try {
        const body = { devices: this.previewRows };
        if (this.defaultProfileKey && String(this.defaultProfileKey).trim()) {
          body.default_profile_key = String(this.defaultProfileKey).trim();
        }
        const { data } = await axios.post(`${this.devicesUrl()}/import`, body);
        const errCount = (data.errors || []).length;
        this.resultType = errCount ? 'warning' : 'success';
        this.resultMessage = this.$t('deviceImportResult', {
          saved: data.saved_count || 0,
          created: data.created || 0,
          updated: data.updated || 0,
          errors: errCount,
        });
        this.$emit('imported');
      } catch (e) {
        this.resultType = 'error';
        this.resultMessage = getErrorMessage(e);
      } finally {
        this.importing = false;
      }
    },
  },
};
</script>
