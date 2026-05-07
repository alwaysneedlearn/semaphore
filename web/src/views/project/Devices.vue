<template>
  <div v-if="items != null">
    <v-dialog v-model="discoveryDialog" :max-width="900">
      <v-card>
        <v-card-title>{{ $t('deviceDiscoveryTitle') }}</v-card-title>
        <v-card-text>
          <p class="text--secondary">
            {{ $t('deviceDiscoveryHelp') }}
          </p>
          <v-textarea
            v-model="discoveryJson"
            :label="$t('deviceDiscoveryJson')"
            outlined
            rows="6"
            auto-grow
          />
          <div class="d-flex mb-2">
            <v-btn small text @click="parseDiscoveryJson">{{ $t('parse') }}</v-btn>
            <v-btn small text class="ml-2" @click="runDiscoverTemplate">
              {{ $t('deviceDiscoverRunTemplate') }}
            </v-btn>
          </div>
          <v-alert dense type="error" v-if="discoveryError">{{ discoveryError }}</v-alert>
          <v-data-table
            :headers="discoveryHeaders"
            :items="discoveredDevices"
            item-key="hostname"
            show-select
            v-model="selectedDiscovered"
            dense
          >
            <template v-slot:item.device_status="{ item }">
              <v-chip x-small :color="statusColor(item.device_status)" dark>
                {{ item.device_status }}
              </v-chip>
            </template>
          </v-data-table>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text @click="discoveryDialog = false">{{ $t('cancel') }}</v-btn>
          <v-btn color="primary" :loading="importingDiscovery" @click="importSelectedDiscovery">
            {{ $t('deviceImportSelected') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <EditDialog
      v-model="editDialog"
      :save-button-text="itemId === 'new' ? $t('create') : $t('save')"
      :icon="'mdi-server-network'"
      icon-color="primary"
      :title="`${itemId === 'new' ? $t('nnew') : $t('edit')} ${$t('device')}`"
      :max-width="500"
      @save="loadItems"
    >
      <template v-slot:form="{ onSave, onError, needSave, needReset }">
        <DeviceForm
          :project-id="projectId"
          :item-id="itemId"
          @save="onSave"
          @error="onError"
          :need-save="needSave"
          :need-reset="needReset"
        />
      </template>
    </EditDialog>

    <YesNoDialog
      :title="$t('deleteDevice')"
      :text="$t('askDeleteDevice')"
      v-model="deleteItemDialog"
      @yes="deleteItem(itemId)"
    />

    <DeviceConfigDialog
      v-model="configDialog"
      :project-id="projectId"
      :device-id="configDeviceId"
      :device-name="configDeviceName"
      @saved="loadItems"
    />

    <v-toolbar flat>
      <v-app-bar-nav-icon @click="showDrawer()"></v-app-bar-nav-icon>
      <v-toolbar-title>{{ $t('devices') }}</v-toolbar-title>
      <v-spacer></v-spacer>
      <v-btn
        v-if="can(USER_PERMISSIONS.manageProjectResources)"
        text
        class="mr-2"
        @click="discoveryDialog = true"
      >
        <v-icon left>mdi-radar</v-icon>
        {{ $t('deviceDiscover') }}
      </v-btn>
      <v-btn
        v-if="can(USER_PERMISSIONS.manageProjectResources)"
        text
        class="mr-2"
        :loading="patrolling"
        @click="runPatrol"
      >
        <v-icon left>mdi-stethoscope</v-icon>
        {{ $t('devicePatrolAll') }}
      </v-btn>
      <v-btn
        v-if="can(USER_PERMISSIONS.manageProjectResources)"
        color="primary"
        depressed
        @click="editItem('new')"
      >
        <v-icon left>mdi-plus</v-icon>
        {{ $t('newDevice') }}
      </v-btn>
    </v-toolbar>

    <v-divider />

    <v-row class="ma-4" dense>
      <v-col cols="6" sm="3">
        <v-card outlined>
          <v-card-text class="pa-3">
            <div class="text-overline">{{ $t('deviceTotal') }}</div>
            <div class="text-h4">{{ stats.total }}</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="6" sm="3">
        <v-card outlined>
          <v-card-text class="pa-3">
            <div class="text-overline">
              <v-icon small color="success">mdi-check-circle</v-icon>
              {{ $t('deviceHealthy') }}
            </div>
            <div class="text-h4">{{ stats.healthy }}</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="6" sm="3">
        <v-card outlined>
          <v-card-text class="pa-3">
            <div class="text-overline">
              <v-icon small color="error">mdi-alert-circle</v-icon>
              {{ $t('deviceUnhealthy') }}
            </div>
            <div class="text-h4">{{ stats.unhealthy }}</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="6" sm="3">
        <v-card outlined>
          <v-card-text class="pa-3">
            <div class="text-overline">
              <v-icon small color="grey">mdi-help-circle</v-icon>
              {{ $t('deviceUnknown') }}
            </div>
            <div class="text-h4">{{ stats.unknown }}</div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-data-table
      :headers="headers"
      :items="items"
      hide-default-footer
      class="mt-2"
      :items-per-page="Number.MAX_VALUE"
      style="max-width: calc(var(--breakpoint-xl) - var(--nav-drawer-width) - 200px); margin: auto;"
    >
      <template v-slot:item.device_status="{ item }">
        <v-chip x-small :color="statusColor(item.device_status)" dark>
          {{ item.device_status }}
        </v-chip>
      </template>
      <template v-slot:item.last_updated="{ item }">
        {{ formatTime(item.last_updated) }}
      </template>
      <template v-slot:item.actions="{ item }">
        <v-btn-toggle dense :value-comparator="() => false">
          <v-btn
            :title="$t('deviceProbe')"
            :loading="busyId === item.id"
            @click="probeDevice(item)"
          >
            <v-icon>mdi-radar</v-icon>
          </v-btn>
          <v-menu offset-y>
            <template v-slot:activator="{ on, attrs }">
              <v-btn :title="$t('deviceActions')" v-bind="attrs" v-on="on">
                <v-icon>mdi-lightning-bolt</v-icon>
              </v-btn>
            </template>
            <v-list dense>
              <v-list-item @click="runAction(item, 'start')">
                <v-list-item-icon><v-icon>mdi-play</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceStart') }}</v-list-item-title>
              </v-list-item>
              <v-list-item @click="runAction(item, 'stop')">
                <v-list-item-icon><v-icon>mdi-stop</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceStop') }}</v-list-item-title>
              </v-list-item>
              <v-list-item @click="runAction(item, 'restart')">
                <v-list-item-icon><v-icon>mdi-restart</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceRestart') }}</v-list-item-title>
              </v-list-item>
              <v-list-item @click="runAction(item, 'status')">
                <v-list-item-icon><v-icon>mdi-stethoscope</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceStatusCheck') }}</v-list-item-title>
              </v-list-item>
              <v-list-item @click="runAction(item, 'config')">
                <v-list-item-icon><v-icon>mdi-cloud-upload</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceConfigPush') }}</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-menu>
          <v-btn :title="$t('deviceConfig')" @click="openConfigDialog(item)">
            <v-icon>mdi-cog</v-icon>
          </v-btn>
          <v-btn :title="$t('edit')" @click="editItem(item.id)">
            <v-icon>mdi-pencil</v-icon>
          </v-btn>
          <v-btn :title="$t('delete')" @click="askDeleteItem(item.id)">
            <v-icon>mdi-delete</v-icon>
          </v-btn>
        </v-btn-toggle>
      </template>
    </v-data-table>
  </div>
</template>
<script>
import axios from 'axios';
import EventBus from '@/event-bus';
import ItemListPageBase from '@/components/ItemListPageBase';
import DeviceForm from '@/components/DeviceForm.vue';
import DeviceConfigDialog from '@/components/DeviceConfigDialog.vue';
import { getErrorMessage } from '@/lib/error';

export default {
  mixins: [ItemListPageBase],
  components: { DeviceForm, DeviceConfigDialog },

  data() {
    return {
      stats: {
        total: 0, healthy: 0, unhealthy: 0, checking: 0, unknown: 0,
      },
      discovering: false,
      patrolling: false,
      busyId: null,
      configDialog: false,
      configDeviceId: null,
      configDeviceName: '',
      discoveryDialog: false,
      discoveryJson: '',
      discoveryError: '',
      discoveredDevices: [],
      selectedDiscovered: [],
      importingDiscovery: false,
      discoveryHeaders: [
        { text: 'Hostname', value: 'hostname' },
        { text: 'IP', value: 'ip_address' },
        { text: 'Status', value: 'device_status' },
      ],
    };
  },

  // ItemListPageBase has askDeleteItem call /refs which we don't expose;
  // override to skip the refs check and go straight to the confirm dialog.
  methods: {
    async askDeleteItem(itemId) {
      this.itemId = itemId;
      this.deleteItemDialog = true;
    },

    statusColor(s) {
      if (s === 'healthy' || s === 'online') return 'success';
      if (s === 'unhealthy' || s === 'offline') return 'error';
      if (s === 'checking') return 'warning';
      return 'grey';
    },

    formatTime(t) {
      if (!t) return '\u2014';
      try {
        return new Date(t).toLocaleString();
      } catch (_) {
        return t;
      }
    },

    getHeaders() {
      return [
        { text: this.$i18n.t('deviceIpAddress'), value: 'ip_address', width: '18%' },
        { text: this.$i18n.t('deviceHostname'), value: 'hostname', width: '25%' },
        { text: this.$i18n.t('deviceStatus'), value: 'device_status', width: '12%' },
        { text: this.$i18n.t('deviceLastUpdated'), value: 'last_updated', width: '15%' },
        { value: 'actions', sortable: false, width: '0%' },
      ];
    },

    getItemsUrl() {
      return `/api/project/${this.projectId}/devices`;
    },
    getSingleItemUrl() {
      return `/api/project/${this.projectId}/devices/${this.itemId}`;
    },
    getEventName() {
      return 'i-device';
    },

    async loadItems() {
      const [devicesRes, statsRes] = await Promise.all([
        axios.get(this.getItemsUrl()),
        axios.get(`${this.getItemsUrl()}/stats`),
      ]);
      this.items = devicesRes.data || [];
      this.stats = statsRes.data || this.stats;
    },

    async probeDevice(device) {
      this.busyId = device.id;
      try {
        await axios.post(`${this.getItemsUrl()}/${device.id}/probe`);
        await this.loadItems();
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.busyId = null;
      }
    },

    async runAction(device, action) {
      this.busyId = device.id;
      try {
        const res = await axios.post(`${this.getItemsUrl()}/${device.id}/action`, { action });
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceTaskQueued', { id: res.data && res.data.id }),
        });
        if (res.data && res.data.id) {
          EventBus.$emit('i-show-task', { taskId: res.data.id });
        }
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.busyId = null;
      }
    },

    async discoverDevices() {
      this.discovering = true;
      try {
        const res = await axios.post(`${this.getItemsUrl()}/discover`);
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceTaskQueued', { id: res.data && res.data.id }),
        });
        if (res.data && res.data.id) {
          EventBus.$emit('i-show-task', { taskId: res.data.id });
        }
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.discovering = false;
      }
    },
    parseDiscoveryJson() {
      this.discoveryError = '';
      try {
        const parsed = JSON.parse(this.discoveryJson || '[]');
        if (!Array.isArray(parsed)) {
          throw new Error('json must be array');
        }
        this.discoveredDevices = parsed
          .map((x) => ({
            hostname: (x.hostname || '').trim(),
            ip_address: (x.ip_address || x.ip || '').trim(),
            device_status: x.device_status || x.status || 'unknown',
          }))
          .filter((x) => x.hostname);
        this.selectedDiscovered = [...this.discoveredDevices];
      } catch (e) {
        this.discoveryError = this.$i18n.t('deviceDiscoveryJsonInvalid');
      }
    },
    async runDiscoverTemplate() {
      await this.discoverDevices();
    },
    async importSelectedDiscovery() {
      this.importingDiscovery = true;
      this.discoveryError = '';
      try {
        const payload = {
          devices: this.discoveredDevices,
          selected_hostnames: this.selectedDiscovered.map((x) => x.hostname),
        };
        const res = await axios.post(`${this.getItemsUrl()}/discovery/import`, payload);
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceImportSaved', { count: res.data.saved_count || 0 }),
        });
        this.discoveryDialog = false;
        await this.loadItems();
      } catch (e) {
        this.discoveryError = getErrorMessage(e);
      } finally {
        this.importingDiscovery = false;
      }
    },
    async runPatrol() {
      this.patrolling = true;
      try {
        const res = await axios.post(`${this.getItemsUrl()}/patrol`);
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceTaskQueued', { id: res.data && res.data.id }),
        });
        if (res.data && res.data.id) {
          EventBus.$emit('i-show-task', { taskId: res.data.id });
        }
        await this.loadItems();
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.patrolling = false;
      }
    },

    openConfigDialog(device) {
      this.configDeviceId = device.id;
      this.configDeviceName = device.hostname;
      this.configDialog = true;
    },
  },
};
</script>
