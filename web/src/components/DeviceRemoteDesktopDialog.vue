<template>
  <v-dialog v-model="dialog" :max-width="480" persistent>
    <v-card v-if="device" class="rdp-launch-card">
      <div class="rdp-launch-hero pa-6 pb-4">
        <div class="d-flex align-center mb-3">
          <v-avatar color="primary" size="44" class="mr-3 elevation-1">
            <v-icon dark>mdi-remote-desktop</v-icon>
          </v-avatar>
          <div>
            <div class="text-h6 font-weight-medium">{{ $t('deviceRemoteDesktop') }}</div>
            <div class="caption rdp-launch-muted">{{ $t('deviceRemoteDesktopLaunchSubtitle') }}</div>
          </div>
          <v-spacer />
          <v-btn icon small :disabled="status === 'launching'" @click="close">
            <v-icon>mdi-close</v-icon>
          </v-btn>
        </div>

        <div class="rdp-launch-device pa-3">
          <div class="font-weight-medium text-truncate">{{ device.hostname || '-' }}</div>
          <div class="caption rdp-launch-muted mt-1">
            {{ device.ip_address }}
            <span v-if="rdpPort"> · :{{ rdpPort }}</span>
            <span v-if="rdpUser"> · {{ rdpUser }}</span>
          </div>
        </div>
      </div>

      <v-card-text class="pt-2">
        <div v-if="operatorName" class="caption mb-3">
          <v-icon x-small class="mr-1">mdi-account</v-icon>
          {{ $t('deviceRemoteDesktopOperator') }}: <strong>{{ operatorName }}</strong>
          <span v-if="logId" class="rdp-launch-muted"> · #{{ logId }}</span>
        </div>

        <div class="d-flex align-start mb-2">
          <v-progress-circular
            v-if="status === 'launching'"
            indeterminate
            size="22"
            width="2"
            color="primary"
            class="mr-3 mt-1"
          />
          <v-icon v-else-if="status === 'opened'" color="success" class="mr-3">mdi-check-circle</v-icon>
          <v-icon v-else-if="status === 'helper_missing'" color="warning" class="mr-3">mdi-alert</v-icon>
          <v-icon v-else color="error" class="mr-3">mdi-alert-circle</v-icon>
          <div>
            <div class="subtitle-2">{{ statusTitle }}</div>
            <div class="body-2 rdp-launch-muted mt-1">{{ statusDetail }}</div>
          </div>
        </div>

        <v-alert
          v-if="status === 'helper_missing'"
          type="warning"
          dense
          text
          class="mt-3 mb-0"
        >
          {{ $t('deviceRemoteDesktopHelperFailed') }}
        </v-alert>
        <v-alert
          v-else-if="status === 'error' && errorText"
          type="error"
          dense
          text
          class="mt-3 mb-0"
        >
          {{ errorText }}
        </v-alert>
      </v-card-text>

      <v-card-actions class="px-4 pb-4">
        <v-spacer />
        <v-btn
          v-if="status === 'helper_missing' || status === 'error'"
          color="primary"
          depressed
          :loading="status === 'launching'"
          @click="retry"
        >
          {{ $t('deviceRemoteDesktopRetry') }}
        </v-btn>
        <v-btn text :disabled="status === 'launching'" @click="close">
          {{ status === 'opened' ? $t('close') : $t('deviceRemoteDesktopDismiss') }}
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
    projectId: { type: [Number, String], required: true },
    device: { type: Object, default: null },
  },

  data() {
    return {
      status: 'launching', // launching | opened | helper_missing | error
      errorText: '',
      operatorName: '',
      logId: null,
      rdpPort: null,
      rdpUser: '',
      launchSeq: 0,
    };
  },

  computed: {
    dialog: {
      get() { return this.value; },
      set(v) { this.$emit('input', v); },
    },
    statusTitle() {
      if (this.status === 'launching') return this.$t('deviceRemoteDesktopStatusLaunching');
      if (this.status === 'opened') return this.$t('deviceRemoteDesktopStatusOpened');
      if (this.status === 'helper_missing') return this.$t('deviceRemoteDesktopStatusHelperMissing');
      return this.$t('deviceRemoteDesktopStatusError');
    },
    statusDetail() {
      if (this.status === 'launching') return this.$t('deviceRemoteDesktopDetailLaunching');
      if (this.status === 'opened') return this.$t('deviceRemoteDesktopDetailOpened');
      if (this.status === 'helper_missing') return this.$t('deviceRemoteDesktopDetailHelperMissing');
      return this.$t('deviceRemoteDesktopDetailError');
    },
  },

  watch: {
    value(open) {
      if (open && this.device) {
        this.startLaunch();
      }
    },
  },

  methods: {
    close() {
      this.dialog = false;
    },
    retry() {
      this.startLaunch();
    },
    async startLaunch() {
      if (!this.device?.id) return;
      const seq = ++this.launchSeq;
      this.status = 'launching';
      this.errorText = '';
      this.operatorName = '';
      this.logId = null;
      this.rdpPort = this.device.rdp_port || 3389;
      this.rdpUser = this.device.rdp_user || '';

      try {
        const { data } = await axios.post(
          `/api/project/${this.projectId}/devices/${this.device.id}/rdp/launch`,
        );
        if (seq !== this.launchSeq) return;

        this.operatorName = (data && data.username) || '';
        this.logId = data && data.log_id;
        if (data && data.rdp_port) this.rdpPort = data.rdp_port;
        if (data && data.rdp_user) this.rdpUser = data.rdp_user;

        let helperUrl = (data && data.helper_url) || '';
        if (!helperUrl && data && data.token) {
          helperUrl = `semaphore-rdp://connect?token=${encodeURIComponent(data.token)}`;
        }
        if (!helperUrl) {
          throw new Error('empty helper_url');
        }
        const base = encodeURIComponent(window.location.origin);
        const sep = helperUrl.includes('?') ? '&' : '?';
        helperUrl = `${helperUrl}${sep}base=${base}`;

        let helperOpened = false;
        const onBlur = () => {
          helperOpened = true;
        };
        window.addEventListener('blur', onBlur);
        try {
          window.location.href = helperUrl;
        } catch (_) {
          helperOpened = false;
        }
        await new Promise((resolve) => {
          setTimeout(resolve, 1600);
        });
        window.removeEventListener('blur', onBlur);
        if (seq !== this.launchSeq) return;

        if (!helperOpened && document.hasFocus()) {
          this.status = 'helper_missing';
        } else {
          this.status = 'opened';
        }
      } catch (e) {
        if (seq !== this.launchSeq) return;
        this.status = 'error';
        this.errorText = getErrorMessage(e);
      }
    },
  },
};
</script>

<style scoped>
.rdp-launch-card {
  overflow: hidden;
}
.rdp-launch-hero {
  background: linear-gradient(135deg, rgba(25, 118, 210, 0.08), rgba(25, 118, 210, 0.02));
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}
.theme--dark .rdp-launch-hero {
  background: linear-gradient(135deg, rgba(100, 181, 246, 0.12), rgba(100, 181, 246, 0.03));
  border-bottom-color: rgba(255, 255, 255, 0.08);
}
.rdp-launch-device {
  background: rgba(255, 255, 255, 0.7);
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.06);
}
.theme--dark .rdp-launch-device {
  background: rgba(255, 255, 255, 0.04);
  border-color: rgba(255, 255, 255, 0.08);
}
.rdp-launch-muted {
  opacity: 0.72;
}
</style>
