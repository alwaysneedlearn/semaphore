<template>
  <div v-if="items != null">
    <v-dialog v-model="deviceSettingsDialog" :max-width="900">
      <v-card>
        <v-card-title>{{ $t('deviceSettingsTitle') }}</v-card-title>
        <v-card-text>
          <DeviceSettingsForm
            ref="deviceSettingsForm"
            :project-id="projectId"
            :hide-actions="true"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text @click="deviceSettingsDialog = false">{{ $t('close') }}</v-btn>
          <v-btn
            color="primary"
            depressed
            :loading="deviceSettingsSaving"
            @click="saveDeviceSettings"
          >
            {{ $t('save') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="discoveryDialog" :max-width="900">
      <v-card>
        <v-card-title>{{ $t('deviceDiscoveryTitle') }}</v-card-title>
        <v-card-text>
          <p class="text--secondary">
            {{ $t('deviceDiscoveryHelp') }}
          </p>
          <v-text-field
            v-model="discoverySubnet"
            :label="$t('deviceDiscoverySubnet')"
            :hint="$t('deviceDiscoverySubnetHint')"
            persistent-hint
            outlined
            dense
            class="mb-2"
            placeholder="192.168.1.0/24"
          />
          <v-btn
            color="primary"
            class="mb-3"
            :loading="discovering"
            @click="runDiscoverTemplate"
          >
            {{ $t('deviceDiscoverRunTemplate') }}
          </v-btn>
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
            <template v-slot:item.rdp_status="{ item }">
              <v-chip x-small :color="statusColor(item.rdp_status)" dark>
                {{ item.rdp_status }}
              </v-chip>
            </template>
            <template v-slot:item.winrm_status="{ item }">
              <v-chip x-small :color="statusColor(item.winrm_status)" dark>
                {{ item.winrm_status }}
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
    <v-dialog v-model="reasonDialog" max-width="700">
      <v-card>
        <v-card-title>
          {{ $t('deviceAbnormalReason') }} - {{ reasonDeviceHostname }}
        </v-card-title>
        <v-card-text>
          <v-alert dense type="warning" v-if="reasonError">{{ reasonError }}</v-alert>
          <div class="mb-2">
            <strong>{{ $t('deviceStatus') }}:</strong> {{ reasonData.device_status || '-' }}
          </div>
          <div class="mb-4">
            <strong>{{ $t('deviceAbnormalReason') }}:</strong>
            {{ reasonData.abnormal_reason || $t('deviceReasonEmpty') }}
          </div>
          <v-data-table
            :headers="reasonHeaders"
            :items="reasonData.logs || []"
            dense
            hide-default-footer
            :items-per-page="Number.MAX_VALUE"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text @click="reasonDialog = false">{{ $t('close') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-toolbar flat>
      <v-app-bar-nav-icon @click="showDrawer()"></v-app-bar-nav-icon>
      <v-toolbar-title>{{ $t('devices') }}</v-toolbar-title>
      <v-spacer></v-spacer>
      <v-btn
        v-if="can(USER_PERMISSIONS.manageProjectResources)"
        text
        class="mr-2"
        @click="deviceSettingsDialog = true"
      >
        <v-icon left>mdi-cog</v-icon>
        {{ $t('deviceSettingsTitle') }}
      </v-btn>
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
      <v-menu offset-y v-if="can(USER_PERMISSIONS.manageProjectResources)">
        <template v-slot:activator="{ on, attrs }">
          <v-btn
            class="ml-2"
            outlined
            v-bind="attrs"
            v-on="on"
            :disabled="selectedDeviceIds.length === 0 || bulkLoading"
          >
            <v-icon left>mdi-playlist-check</v-icon>
            {{ $t('deviceBulkOps') }} ({{ selectedDeviceIds.length }})
          </v-btn>
        </template>
        <v-list dense>
          <v-list-item @click="runBulkProbe()">
            <v-list-item-icon><v-icon>mdi-radar</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceProbe') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('start')">
            <v-list-item-icon><v-icon>mdi-play</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceStart') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('stop')">
            <v-list-item-icon><v-icon>mdi-stop</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceStop') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('restart')">
            <v-list-item-icon><v-icon>mdi-restart</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceRestart') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('status')">
            <v-list-item-icon><v-icon>mdi-stethoscope</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceStatusCheck') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('config')">
            <v-list-item-icon><v-icon>mdi-cloud-upload</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceConfigPush') }}</v-list-item-title>
          </v-list-item>
          <v-divider />
          <v-list-item @click="bulkDeleteDialog = true">
            <v-list-item-icon><v-icon color="error">mdi-delete</v-icon></v-list-item-icon>
            <v-list-item-title class="error--text">{{ $t('delete') }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
    </v-toolbar>

    <v-divider />

    <YesNoDialog
      :title="$t('deleteDevice')"
      :text="$t('deviceBulkDeleteConfirm', { count: selectedDeviceIds.length })"
      v-model="bulkDeleteDialog"
      @yes="bulkDelete()"
    />

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
              <v-icon small color="warning">mdi-progress-clock</v-icon>
              {{ $t('deviceChecking') }}
            </div>
            <div class="text-h4">{{ stats.checking }}</div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-data-table
      :headers="headers"
      :items="items || []"
      item-key="id"
      :server-items-length="totalDevices"
      :options="tableOptions"
      :loading="devicesLoading"
      :footer-props="{
        itemsPerPageOptions: [10, 15, 25, 50],
        showFirstLastPage: true,
      }"
      class="mt-2"
      style="max-width: calc(var(--breakpoint-xl) - var(--nav-drawer-width) - 200px); margin: auto;"
      @update:options="onDeviceTableOptions"
    >
      <template v-slot:header._select>
        <v-checkbox
          class="ma-0 pa-0"
          dense
          hide-details
          :ripple="false"
          :input-value="pageAllSelected"
          :indeterminate="pageSomeSelected"
          @click.stop.prevent="toggleSelectPage"
        />
      </template>
      <template v-slot:item._select="{ item }">
        <v-checkbox
          class="ma-0 pa-0"
          dense
          hide-details
          :ripple="false"
          :input-value="selectedDeviceIds.includes(item.id)"
          @click.stop.prevent="toggleDeviceRow(item)"
        />
      </template>
      <template v-slot:top>
        <v-row dense class="px-4 pt-3">
          <v-col cols="12" md="3">
            <v-text-field
              v-model.trim="filters.hostname"
              :label="$t('deviceFilterHostname')"
              clearable
              dense
              outlined
            />
          </v-col>
          <v-col cols="12" md="3">
            <v-text-field
              v-model.trim="filters.ip"
              :label="$t('deviceFilterIp')"
              clearable
              dense
              outlined
            />
          </v-col>
          <v-col cols="12" md="2">
            <v-select
              v-model="filters.deviceStatus"
              :items="statusFilterOptions"
              :label="$t('deviceStatus')"
              clearable
              dense
              outlined
            />
          </v-col>
          <v-col cols="12" md="2">
            <v-select
              v-model="filters.rdpStatus"
              :items="protocolFilterOptions"
              :label="$t('deviceRdpStatus')"
              clearable
              dense
              outlined
            />
          </v-col>
          <v-col cols="12" md="2">
            <v-select
              v-model="filters.winrmStatus"
              :items="protocolFilterOptions"
              :label="$t('deviceWinrmStatus')"
              clearable
              dense
              outlined
            />
          </v-col>
        </v-row>
        <div class="px-4 pb-1 text--secondary caption">
          {{ $t('devicePaginationSelectionHint') }}
        </div>
      </template>
      <template v-slot:item.device_status="{ item }">
        <v-chip x-small :color="statusColor(item.device_status)" dark>
          {{ item.device_status }}
        </v-chip>
      </template>
      <template v-slot:item.rdp_status="{ item }">
        <v-chip x-small :color="statusColor(item.rdp_status)" dark>
          {{ item.rdp_status }}
        </v-chip>
      </template>
      <template v-slot:item.winrm_status="{ item }">
        <v-chip x-small :color="statusColor(item.winrm_status)" dark>
          {{ item.winrm_status }}
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
          <v-btn
            :title="$t('deviceAbnormalReason')"
            :disabled="item.device_status === 'healthy' || item.device_status === 'unknown'"
            @click="openReasonDialog(item)"
          >
            <v-icon>mdi-alert-circle-outline</v-icon>
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
import DeviceSettingsForm from '@/components/DeviceSettingsForm.vue';
import { getErrorMessage } from '@/lib/error';

export default {
  mixins: [ItemListPageBase],
  components: { DeviceForm, DeviceConfigDialog, DeviceSettingsForm },

  data() {
    return {
      stats: {
        total: 0, healthy: 0, unhealthy: 0, checking: 0, unknown: 0,
      },
      discovering: false,
      patrolling: false,
      deviceSettingsDialog: false,
      deviceSettingsSaving: false,
      busyId: null,
      configDialog: false,
      configDeviceId: null,
      configDeviceName: '',
      reasonDialog: false,
      reasonError: '',
      reasonDeviceHostname: '',
      reasonData: {},
      reasonHeaders: [
        { text: this.$i18n.t('deviceLastUpdated'), value: 'created' },
        { text: this.$i18n.t('deviceStatus'), value: 'status' },
        { text: this.$i18n.t('deviceRdpStatus'), value: 'rdp_status' },
        { text: this.$i18n.t('deviceWinrmStatus'), value: 'winrm_status' },
        { text: this.$i18n.t('deviceAbnormalReason'), value: 'abnormal_reason' },
      ],
      discoveryDialog: false,
      discoverySubnet: '',
      discoveryError: '',
      discoveredDevices: [],
      selectedDiscovered: [],
      selectedDeviceIds: [],
      totalDevices: 0,
      devicesLoading: false,
      tableOptions: {
        page: 1,
        itemsPerPage: 15,
        sortBy: ['hostname'],
        sortDesc: [false],
        groupBy: [],
        groupDesc: [],
        multiSort: false,
        mustSort: false,
      },
      filterDebounceTimer: null,
      importingDiscovery: false,
      bulkLoading: false,
      bulkDeleteDialog: false,
      filters: {
        hostname: '',
        ip: '',
        deviceStatus: null,
        rdpStatus: null,
        winrmStatus: null,
      },
      discoveryHeaders: [
        { text: 'Hostname', value: 'hostname' },
        { text: 'IP', value: 'ip_address' },
        { text: 'Status', value: 'device_status' },
        { text: 'RDP', value: 'rdp_status' },
        { text: 'WinRM', value: 'winrm_status' },
      ],
    };
  },
  computed: {
    statusFilterOptions() {
      return ['healthy', 'unhealthy', 'checking', 'unknown'];
    },
    protocolFilterOptions() {
      return ['online', 'offline', 'unknown'];
    },
    pageDeviceIds() {
      return (this.items || []).map((x) => x.id);
    },
    pageAllSelected() {
      if (!this.pageDeviceIds.length) {
        return false;
      }
      return this.pageDeviceIds.every((id) => this.selectedDeviceIds.includes(id));
    },
    pageSomeSelected() {
      const some = this.pageDeviceIds.some((id) => this.selectedDeviceIds.includes(id));
      return some && !this.pageAllSelected;
    },
  },

  watch: {
    filters: {
      handler() {
        clearTimeout(this.filterDebounceTimer);
        this.filterDebounceTimer = setTimeout(() => {
          this.tableOptions = {
            ...this.tableOptions,
            page: 1,
          };
          this.loadItems();
        }, 350);
      },
      deep: true,
    },
  },

  // ItemListPageBase has askDeleteItem call /refs which we don't expose;
  // override to skip the refs check and go straight to the confirm dialog.
  methods: {
    async askDeleteItem(itemId) {
      this.itemId = itemId;
      this.deleteItemDialog = true;
    },

    async saveDeviceSettings() {
      if (!this.$refs.deviceSettingsForm || this.deviceSettingsSaving) {
        return;
      }

      this.deviceSettingsSaving = true;
      try {
        const ok = await this.$refs.deviceSettingsForm.save();
        if (ok) {
          this.deviceSettingsDialog = false;
        }
      } finally {
        this.deviceSettingsSaving = false;
      }
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
        {
          text: '',
          value: '_select',
          sortable: false,
          width: '52px',
          align: 'center',
        },
        { text: this.$i18n.t('deviceIpAddress'), value: 'ip_address', width: '18%' },
        { text: this.$i18n.t('deviceHostname'), value: 'hostname', width: '25%' },
        { text: this.$i18n.t('deviceStatus'), value: 'device_status', width: '12%' },
        { text: this.$i18n.t('deviceRdpStatus'), value: 'rdp_status', width: '9%' },
        { text: this.$i18n.t('deviceWinrmStatus'), value: 'winrm_status', width: '9%' },
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

    onDeviceTableOptions(opts) {
      this.tableOptions = opts;
      this.loadItems();
    },

    toggleSelectPage() {
      const checked = !this.pageAllSelected;
      const idsOnPage = new Set(this.pageDeviceIds);
      if (checked) {
        const merged = new Set(this.selectedDeviceIds);
        this.pageDeviceIds.forEach((id) => merged.add(id));
        this.selectedDeviceIds = [...merged];
      } else {
        this.selectedDeviceIds = this.selectedDeviceIds.filter((id) => !idsOnPage.has(id));
      }
    },

    toggleDeviceRow(item) {
      const i = this.selectedDeviceIds.indexOf(item.id);
      if (i >= 0) {
        this.selectedDeviceIds.splice(i, 1);
      } else {
        this.selectedDeviceIds.push(item.id);
      }
    },

    async loadItems() {
      this.devicesLoading = true;
      try {
        const opts = this.tableOptions || {};
        const page = opts.page || 1;
        const itemsPerPage = opts.itemsPerPage || 15;
        const offset = (page - 1) * itemsPerPage;
        const sortArr = opts.sortBy || ['hostname'];
        const sortDescArr = opts.sortDesc || [false];
        const sortBy = sortArr[0] || 'hostname';
        const sortDesc = !!sortDescArr[0];

        const params = {
          limit: itemsPerPage,
          offset,
          sort: sortBy,
          order: sortDesc ? 'desc' : 'asc',
        };
        if (this.filters.hostname) params.hostname = this.filters.hostname;
        if (this.filters.ip) params.ip = this.filters.ip;
        if (this.filters.deviceStatus) params.device_status = this.filters.deviceStatus;
        if (this.filters.rdpStatus) params.rdp_status = this.filters.rdpStatus;
        if (this.filters.winrmStatus) params.winrm_status = this.filters.winrmStatus;

        const [devicesRes, statsRes] = await Promise.all([
          axios.get(this.getItemsUrl(), { params }),
          axios.get(`${this.getItemsUrl()}/stats`),
        ]);
        const body = devicesRes.data || {};
        this.items = body.devices || [];
        this.totalDevices = typeof body.total === 'number' ? body.total : (this.items || []).length;
        this.stats = statsRes.data || this.stats;
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
        this.items = [];
        this.totalDevices = 0;
      } finally {
        this.devicesLoading = false;
      }
    },

    async deleteItem(itemId) {
      this.itemId = itemId;
      try {
        const item = (this.items || []).find(
          (x) => x[this.IDFieldName] === itemId,
        ) || { id: itemId };
        await axios({
          method: 'delete',
          url: this.getSingleItemUrl(),
          responseType: 'json',
        });
        EventBus.$emit(this.getEventName(), {
          action: 'delete',
          item,
        });
        this.selectedDeviceIds = this.selectedDeviceIds.filter((id) => id !== itemId);
        await this.loadItems();
      } catch (err) {
        EventBus.$emit('i-snackbar', {
          color: 'error',
          text: getErrorMessage(err),
        });
      }
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
        const res = await axios.post(`${this.getItemsUrl()}/actions/bulk`, {
          action,
          device_ids: [device.id],
        });
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
        const payload = {
          subnet: this.discoverySubnet || null,
        };
        const res = await axios.post(`${this.getItemsUrl()}/discover`, payload);
        const taskId = res.data && res.data.id;
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceTaskQueued', { id: taskId }),
        });
        if (taskId) {
          EventBus.$emit('i-show-task', { taskId });
          await this.waitAndLoadDiscoveryResult(taskId);
        }
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.discovering = false;
      }
    },
    async waitAndLoadDiscoveryResult(taskId, attemptsLeft = 60) {
      const taskRes = await axios.get(`/api/project/${this.projectId}/tasks/${taskId}`);
      const status = (taskRes.data && taskRes.data.status) || '';
      if (status === 'success') {
        await this.loadDiscoveryResultFromTask(taskId);
        return;
      }
      if (status === 'error' || status === 'stopped') {
        this.discoveryError = this.$i18n.t('deviceDiscoveryTaskFailed');
        return;
      }
      if (attemptsLeft <= 1) {
        this.discoveryError = this.$i18n.t('deviceDiscoveryTaskTimeout');
        return;
      }
      await new Promise((resolve) => {
        setTimeout(resolve, 2000);
      });
      await this.waitAndLoadDiscoveryResult(taskId, attemptsLeft - 1);
    },
    tryParseJsonArray(candidate) {
      if (!candidate) {
        return null;
      }
      try {
        const parsed = JSON.parse(candidate.trim());
        return Array.isArray(parsed) ? parsed : null;
      } catch (e) {
        return null;
      }
    },
    stripAnsiCodes(input) {
      if (!input) {
        return '';
      }
      const ESC = '\u001b';
      let i = 0;
      let out = '';
      while (i < input.length) {
        if (input[i] === ESC && input[i + 1] === '[') {
          i += 2;
          while (i < input.length) {
            const ch = input[i];
            if ((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')) {
              i += 1;
              break;
            }
            i += 1;
          }
        } else {
          out += input[i];
          i += 1;
        }
      }
      return out;
    },
    extractJsonArrayFromText(text) {
      if (!text) {
        return null;
      }

      const clean = this.stripAnsiCodes(text);

      const prefixed = this.extractPrefixedDiscoveryJson(clean);
      if (prefixed) {
        return prefixed;
      }

      const direct = this.tryParseJsonArray(clean);
      if (direct) {
        return direct;
      }

      const escapedFieldRegex = /"(stdout|msg)"\s*:\s*"((?:\\.|[^"\\])*)"/g;
      let escapedFieldMatch = escapedFieldRegex.exec(clean);
      while (escapedFieldMatch !== null) {
        try {
          const unescaped = JSON.parse(`"${escapedFieldMatch[2]}"`);
          const prefixedField = this.extractPrefixedDiscoveryJson(unescaped);
          if (prefixedField) {
            return prefixedField;
          }
          const fromField = this.tryParseJsonArray(unescaped);
          if (fromField) {
            return fromField;
          }
        } catch (e) {
          // continue
        }
        escapedFieldMatch = escapedFieldRegex.exec(clean);
      }

      const arrayLikeMatches = clean.match(/\[\s*\{[\s\S]*?\}\s*\]/g) || [];
      for (let i = arrayLikeMatches.length - 1; i >= 0; i -= 1) {
        const parsed = this.tryParseJsonArray(arrayLikeMatches[i]);
        if (parsed) {
          return parsed;
        }
      }

      const yamlLike = this.tryParseYamlLikeDiscoveryList(clean);
      if (yamlLike) {
        return yamlLike;
      }

      return null;
    },
    extractPrefixedDiscoveryJson(text) {
      if (!text) {
        return null;
      }
      const marker = 'SEMAPHORE_DISCOVERY_JSON=';
      const idx = text.lastIndexOf(marker);
      if (idx < 0) {
        return null;
      }
      const after = text.slice(idx + marker.length).trim();
      const endLine = after.indexOf('\n');
      const candidate = (endLine >= 0 ? after.slice(0, endLine) : after).trim();
      return this.tryParseJsonArray(candidate);
    },
    tryParseYamlLikeDiscoveryList(text) {
      if (!text || !text.includes('device_status:')) {
        return null;
      }

      const lines = text.split('\n');
      const items = [];
      let current = null;

      const parseValue = (v) => {
        const raw = (v || '').trim();
        if (!raw) {
          return '';
        }
        if ((raw.startsWith('"') && raw.endsWith('"')) || (raw.startsWith('\'') && raw.endsWith('\''))) {
          return raw.slice(1, -1);
        }
        return raw;
      };

      for (let i = 0; i < lines.length; i += 1) {
        const line = this.stripAnsiCodes(lines[i] || '');
        const itemStart = line.match(/^\s*-\s+([a-z_]+)\s*:\s*(.*)$/i);
        if (itemStart) {
          if (current) {
            items.push(current);
          }
          current = { [itemStart[1]]: parseValue(itemStart[2]) };
        } else {
          const kv = line.match(/^\s{2,}([a-z_]+)\s*:\s*(.*)$/i);
          if (kv && current) {
            current[kv[1]] = parseValue(kv[2]);
          }
        }
      }

      if (current) {
        items.push(current);
      }

      if (items.length === 0 || !items.some((x) => x.ip_address || x.hostname)) {
        return null;
      }

      return items;
    },
    async loadDiscoveryResultFromTask(taskId) {
      const outputRes = await axios.get(`/api/project/${this.projectId}/tasks/${taskId}/output`);
      const lines = outputRes.data || [];
      const merged = lines.map((line) => line.output || '').join('\n');
      let parsed = this.extractJsonArrayFromText(merged);

      if (!parsed) {
        try {
          const rawRes = await axios.get(
            `/api/project/${this.projectId}/tasks/${taskId}/raw_output`,
            { responseType: 'text' },
          );
          parsed = this.extractJsonArrayFromText(rawRes.data || '');
        } catch (e) {
          // ignore raw output fallback errors and keep original message below
        }
      }

      if (!parsed) {
        this.discoveryError = this.$i18n.t('deviceDiscoveryResultNotFound');
        return;
      }
      this.applyDiscoveredDevices(parsed);
      EventBus.$emit('i-snackbar', {
        color: 'success',
        text: this.$i18n.t('deviceDiscoveryLoaded', { count: this.discoveredDevices.length }),
      });
    },
    applyDiscoveredDevices(parsed) {
      this.discoveredDevices = (parsed || [])
        .map((x) => ({
          hostname: (x.hostname || '').trim(),
          ip_address: (x.ip_address || x.ip || '').trim(),
          device_status: x.device_status || x.status || 'unknown',
          rdp_status: x.rdp_status || 'unknown',
          winrm_status: x.winrm_status || 'unknown',
          abnormal_reason: x.abnormal_reason || null,
        }))
        .filter((x) => x.hostname);
      this.selectedDiscovered = [...this.discoveredDevices];
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
          selected_ips: this.selectedDiscovered
            .map((x) => String(x.ip_address || x.ip || '').trim())
            .filter((ip) => ip),
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
    async runBulkProbe() {
      this.bulkLoading = true;
      try {
        await Promise.all(this.selectedDeviceIds.map((id) => axios.post(`${this.getItemsUrl()}/${id}/probe`)));
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceBulkDone', { count: this.selectedDeviceIds.length }),
        });
        await this.loadItems();
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.bulkLoading = false;
      }
    },
    async runBulkAction(action) {
      this.bulkLoading = true;
      try {
        const res = await axios.post(`${this.getItemsUrl()}/actions/bulk`, {
          action,
          device_ids: this.selectedDeviceIds,
        });
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceTaskQueued', { id: res.data && res.data.id }),
        });
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.bulkLoading = false;
      }
    },
    async bulkDelete() {
      this.bulkLoading = true;
      try {
        await Promise.all(this.selectedDeviceIds.map((id) => axios.delete(`${this.getItemsUrl()}/${id}`)));
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceBulkDone', { count: this.selectedDeviceIds.length }),
        });
        this.selectedDeviceIds = [];
        await this.loadItems();
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.bulkLoading = false;
      }
    },

    openConfigDialog(device) {
      this.configDeviceId = device.id;
      this.configDeviceName = device.hostname;
      this.configDialog = true;
    },
    async openReasonDialog(device) {
      this.reasonDialog = true;
      this.reasonDeviceHostname = device.hostname;
      this.reasonError = '';
      this.reasonData = {};
      try {
        const res = await axios.get(`${this.getItemsUrl()}/${device.id}/status/reason`);
        this.reasonData = res.data || {};
      } catch (e) {
        this.reasonError = getErrorMessage(e);
      }
    },
  },
};
</script>
