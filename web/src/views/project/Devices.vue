<template>
  <div v-if="items != null">
    <v-dialog v-model="deviceProfilesDialog" :max-width="1140" scrollable>
      <v-card>
        <v-card-title>Device types (profiles)</v-card-title>
        <v-card-text>
          <DeviceProfilesForm :project-id="projectId" />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text @click="deviceProfilesDialog = false">{{ $t('close') }}</v-btn>
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
      @yes="deleteItemFinalDialog = true"
    />

    <YesNoDialog
      :title="$t('deviceDeleteConfirmAgainTitle')"
      :text="deviceDeleteFinalText"
      :warning-text="$t('deviceDeleteRiskWarning')"
      v-model="deleteItemFinalDialog"
      :yes-button-title="$t('delete')"
      @yes="deleteItem(itemId)"
    />

    <DeviceConfigDialog
      v-model="configDialog"
      :project-id="projectId"
      :device-id="configDeviceId"
      :device-name="configDeviceName"
      @saved="loadItems"
    />

    <DeviceWinrmConsoleDialog
      v-model="winrmDialog"
      :project-id="projectId"
      :device="winrmDevice"
      @device-updated="onWinrmDeviceUpdated"
    />

    <DeviceRemoteDesktopDialog
      v-model="rdpDialog"
      :project-id="projectId"
      :device="rdpDevice"
    />

    <DeviceImportExportDialog
      v-model="importExportDialog"
      :project-id="projectId"
      :selected-device-ids="selectedDeviceIds"
      @imported="loadItems"
    />

    <DeviceOperationHistoryDialog
      v-model="operationHistoryDialog"
      :project-id="projectId"
      :device="operationHistoryDevice"
    />

    <v-toolbar flat>
      <v-app-bar-nav-icon @click="showDrawer()"></v-app-bar-nav-icon>
      <v-toolbar-title>{{ $t('devices') }}</v-toolbar-title>
      <v-spacer></v-spacer>
      <v-btn
        v-if="can(USER_PERMISSIONS.manageProjectResources)"
        text
        class="mr-2"
        @click="importExportDialog = true"
      >
        <v-icon left>mdi-swap-vertical</v-icon>
        {{ $t('deviceImportExport') }}
      </v-btn>
      <v-btn
        v-if="can(USER_PERMISSIONS.manageProjectResources)"
        text
        class="mr-2"
        @click="deviceProfilesDialog = true"
      >
        <v-icon left>mdi-shape-outline</v-icon>
        Device types
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
      <v-btn
        class="ml-2"
        text
        :disabled="selectedDeviceIds.length === 0 || bulkLoading"
        @click="clearSelectedDevices"
      >
        <v-icon left>mdi-checkbox-blank-off-outline</v-icon>
        {{ $t('deviceClearSelection') }}
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
          <v-list-item :title="$t('deviceProbeTooltip')" @click="runBulkProbe()">
            <v-list-item-icon><v-icon>mdi-radar</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceProbe') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('status')">
            <v-list-item-icon><v-icon>mdi-stethoscope</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceStatusCheck') }}</v-list-item-title>
          </v-list-item>
          <v-list-item
            v-if="selectionHasResendData"
            @click="runBulkAction('resend_data')"
          >
            <v-list-item-icon><v-icon>mdi-database-sync</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceResendData') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('restart')">
            <v-list-item-icon><v-icon>mdi-restart</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceRestart') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('redeploy')">
            <v-list-item-icon><v-icon>mdi-package-down</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceRedeploy') }}</v-list-item-title>
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
      @yes="bulkDeleteFinalDialog = true"
    />

    <YesNoDialog
      :title="$t('deviceDeleteConfirmAgainTitle')"
      :text="$t('deviceBulkDeleteConfirmAgain', { count: selectedDeviceIds.length })"
      :warning-text="$t('deviceDeleteRiskWarning')"
      v-model="bulkDeleteFinalDialog"
      :yes-button-title="$t('delete')"
      @yes="bulkDelete()"
    />

    <YesNoDialog
      title="Confirm bulk action"
      :text="bulkActionConfirmText"
      v-model="bulkActionConfirmDialog"
      @yes="executePendingBulkAction()"
    />

    <DeviceResendDataDialog
      v-model="resendDataDialog"
      :devices="pendingResendDevices"
      :profile-by-id="profileById"
      :loading="bulkLoading || busyId != null"
      @submit="executeResendSubmit"
    />

    <YesNoDialog
      :title="$t('deviceActionConfirmTitle')"
      :text="deviceActionConfirmText"
      :warning-text="$t('deviceActionRiskWarning')"
      v-model="deviceActionConfirmDialog"
      :yes-button-title="$t('confirmTask')"
      @yes="executePendingDeviceAction()"
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

    <v-card outlined flat class="mx-4 mb-2 device-list-filters">
      <v-card-text class="py-3 px-4">
        <div class="d-flex align-center flex-wrap mb-2">
          <v-icon small color="primary" class="mr-2">mdi-filter-variant</v-icon>
          <span class="subtitle-2">{{ $t('deviceFiltersTitle') }}</span>
          <v-spacer />
          <v-btn
            text
            small
            class="px-2"
            :disabled="!hasActiveFilters"
            @click="clearDeviceFilters"
          >
            <v-icon left small>mdi-filter-off</v-icon>
            {{ $t('deviceFiltersClear') }}
          </v-btn>
        </div>
        <v-row dense align="center">
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              v-model.trim="filters.ip"
              :label="$t('deviceFilterIp')"
              clearable
              dense
              outlined
              hide-details
              prepend-inner-icon="mdi-ip-network"
            />
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              v-model.trim="filters.hostname"
              :label="$t('deviceFilterHostname')"
              clearable
              dense
              outlined
              hide-details
              prepend-inner-icon="mdi-server"
            />
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-select
              v-model="filters.deviceProfileId"
              :items="profileFilterOptions"
              item-value="id"
              item-text="label"
              :label="$t('deviceFilterType')"
              clearable
              dense
              outlined
              hide-details
              prepend-inner-icon="mdi-shape-outline"
            />
          </v-col>
        </v-row>
        <v-row dense align="center" class="mt-1">
          <v-col cols="6" sm="3">
            <v-select
              v-model="filters.deviceStatus"
              :items="statusFilterOptions"
              :label="$t('deviceStatus')"
              clearable
              dense
              outlined
              hide-details
            />
          </v-col>
          <v-col cols="6" sm="3">
            <v-select
              v-model="filters.rdpStatus"
              :items="protocolFilterOptions"
              :label="$t('deviceRdpStatus')"
              clearable
              dense
              outlined
              hide-details
            />
          </v-col>
          <v-col cols="6" sm="3">
            <v-select
              v-model="filters.winrmStatus"
              :items="protocolFilterOptions"
              :label="$t('deviceWinrmStatus')"
              clearable
              dense
              outlined
              hide-details
            />
          </v-col>
          <v-col cols="6" sm="3">
            <v-select
              v-model="filters.apiStatus"
              :items="protocolFilterOptions"
              :label="$t('deviceFilterApiStatus')"
              clearable
              dense
              outlined
              hide-details
            />
          </v-col>
        </v-row>
        <div class="d-flex align-center flex-wrap mt-2">
          <p class="text--secondary caption mb-0 mr-3">
            {{ $t('devicePaginationSelectionHint') }}
          </p>
          <v-chip
            v-if="selectedDeviceIds.length > 0"
            small
            color="primary"
            text-color="white"
            class="mr-2"
          >
            {{ $t('deviceSelectedCount', { count: selectedDeviceIds.length }) }}
          </v-chip>
          <v-btn
            text
            small
            class="px-2"
            :loading="selectAllFilteredLoading"
            :disabled="selectAllFilteredDisabled"
            @click="selectAllMatchingFilters"
          >
            <v-icon left small>mdi-checkbox-multiple-marked</v-icon>
            {{ $t('deviceSelectAllFiltered', { count: totalDevices }) }}
          </v-btn>
          <v-btn
            text
            small
            class="px-2"
            :disabled="selectedDeviceIds.length === 0 || bulkLoading"
            @click="clearSelectedDevices"
          >
            <v-icon left small>mdi-checkbox-blank-off-outline</v-icon>
            {{ $t('deviceClearSelection') }}
          </v-btn>
        </div>
      </v-card-text>
    </v-card>

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
      <template v-slot:item.device_profile_id="{ item }">
        {{ profileLabel(item.device_profile_id) }}
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
      <template v-slot:item.api_status="{ item }">
        <v-chip x-small :color="statusColor(item.api_status)" dark>
          {{ item.api_status }}
        </v-chip>
      </template>
      <template v-slot:item.last_updated="{ item }">
        {{ formatTime(item.last_updated) }}
      </template>
      <template v-slot:item.actions="{ item }">
        <v-btn-toggle dense :value-comparator="() => false">
          <v-btn
            :title="`${$t('deviceProbe')}: ${$t('deviceProbeHelp')}`"
            :loading="busyId === item.id"
            @click="probeDevice(item)"
          >
            <v-icon>mdi-radar</v-icon>
          </v-btn>
          <v-btn
            :title="$t('deviceRemoteDesktop')"
            @click.stop="openRemoteDesktop(item)"
          >
            <v-icon>mdi-remote-desktop</v-icon>
          </v-btn>
          <v-btn
            v-if="showWinrmConsole(item)"
            :title="$t('deviceWinrmConsole')"
            @click="openWinrmConsole(item)"
          >
            <v-icon>mdi-console</v-icon>
          </v-btn>
          <v-menu offset-y>
            <template v-slot:activator="{ on, attrs }">
              <v-btn :title="$t('deviceActions')" v-bind="attrs" v-on="on">
                <v-icon>mdi-lightning-bolt</v-icon>
              </v-btn>
            </template>
            <v-list dense>
              <v-list-item @click="runAction(item, 'status')">
                <v-list-item-icon><v-icon>mdi-stethoscope</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceStatusCheck') }}</v-list-item-title>
              </v-list-item>
              <v-list-item
                v-if="deviceSupportsResendData(item)"
                @click="runAction(item, 'resend_data')"
              >
                <v-list-item-icon><v-icon>mdi-database-sync</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceResendData') }}</v-list-item-title>
              </v-list-item>
              <v-list-item @click="runAction(item, 'restart')">
                <v-list-item-icon><v-icon>mdi-restart</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceRestart') }}</v-list-item-title>
              </v-list-item>
              <v-list-item @click="runAction(item, 'redeploy')">
                <v-list-item-icon><v-icon>mdi-package-down</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceRedeploy') }}</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-menu>
          <v-btn :title="$t('deviceDetailTitle')" @click="openOperationHistory(item)">
            <v-icon>mdi-history</v-icon>
          </v-btn>
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
import DeviceWinrmConsoleDialog from '@/components/DeviceWinrmConsoleDialog.vue';
import DeviceRemoteDesktopDialog from '@/components/DeviceRemoteDesktopDialog.vue';
import DeviceProfilesForm from '@/components/DeviceProfilesForm.vue';
import DeviceImportExportDialog from '@/components/DeviceImportExportDialog.vue';
import DeviceOperationHistoryDialog from '@/components/DeviceOperationHistoryDialog.vue';
import DeviceResendDataDialog from '@/components/DeviceResendDataDialog.vue';
import { getErrorMessage } from '@/lib/error';

const RESEND_DATA_PROFILE_KEYS = new Set(['JHAI', 'LAND', 'SINEXCEL', 'NBT', 'NEWARE']);

export default {
  mixins: [ItemListPageBase],
  components: {
    DeviceForm,
    DeviceConfigDialog,
    DeviceWinrmConsoleDialog,
    DeviceRemoteDesktopDialog,
    DeviceProfilesForm,
    DeviceImportExportDialog,
    DeviceOperationHistoryDialog,
    DeviceResendDataDialog,
  },

  data() {
    return {
      stats: {
        total: 0, healthy: 0, unhealthy: 0, checking: 0, unknown: 0,
      },
      patrolling: false,
      deviceProfilesDialog: false,
      importExportDialog: false,
      busyId: null,
      rdpDialog: false,
      rdpDevice: null,
      configDialog: false,
      configDeviceId: null,
      configDeviceName: '',
      winrmDialog: false,
      winrmDevice: null,
      operationHistoryDialog: false,
      operationHistoryDevice: null,
      resendDataDialog: false,
      pendingResendDevices: [],
      selectedDeviceIds: [],
      selectedDevicesMap: {},
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
      bulkLoading: false,
      selectAllFilteredLoading: false,
      bulkDeleteDialog: false,
      bulkDeleteFinalDialog: false,
      deleteItemFinalDialog: false,
      bulkActionConfirmDialog: false,
      pendingBulkAction: null,
      pendingBulkTaskCount: 0,
      deviceActionConfirmDialog: false,
      pendingDeviceAction: null,
      pendingDeviceActionTarget: null,
      profilesList: [],
      profileById: {},
      profileSettingsById: {},
      filters: {
        hostname: '',
        ip: '',
        deviceProfileId: null,
        deviceStatus: null,
        rdpStatus: null,
        winrmStatus: null,
        apiStatus: null,
      },
    };
  },
  computed: {
    statusFilterOptions() {
      return ['healthy', 'unhealthy', 'checking'];
    },
    protocolFilterOptions() {
      return ['online', 'offline'];
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
    profileFilterOptions() {
      return (this.profilesList || []).map((p) => ({
        id: p.id,
        label: `${p.name} (${p.profile_key})`,
      }));
    },
    bulkActionConfirmText() {
      const n = this.pendingBulkTaskCount;
      const action = this.pendingBulkAction || '';
      return `Selected devices span ${n} device type(s). This will create ${n} separate `
        + `task(s) (one per type). Continue with "${action}"?`;
    },
    deviceActionConfirmText() {
      const action = this.pendingDeviceAction;
      if (!action) {
        return '';
      }
      const actionLabel = this.deviceActionLabel(action);
      const device = this.pendingDeviceActionTarget;
      if (device) {
        const host = device.hostname || device.ip_address || `#${device.id}`;
        const ip = device.ip_address || '—';
        return this.$t('deviceActionConfirmSingle', { action: actionLabel, host, ip });
      }
      const count = this.selectedDeviceIds.length;
      let text = this.$t('deviceActionConfirmBulk', { action: actionLabel, count });
      const n = this.pendingBulkTaskCount;
      if (n > 1) {
        text += ` ${this.$t('deviceActionConfirmMultiType', { n })}`;
      }
      return text;
    },
    deviceDeleteFinalText() {
      const id = this.itemId;
      const item = (this.items || []).find((d) => d.id === id);
      if (!item) {
        return this.$t('deviceDeleteConfirmAgainText');
      }
      return this.$t('deviceDeleteConfirmAgainSingle', {
        host: item.hostname || '—',
        ip: item.ip_address || '—',
      });
    },
    hasActiveFilters() {
      const f = this.filters || {};
      return Boolean(
        (f.ip && String(f.ip).trim())
        || (f.hostname && String(f.hostname).trim())
        || f.deviceProfileId
        || f.deviceStatus
        || f.rdpStatus
        || f.winrmStatus
        || f.apiStatus,
      );
    },
    selectAllFilteredDisabled() {
      return this.totalDevices === 0
        || this.devicesLoading
        || this.bulkLoading
        || this.selectAllFilteredLoading;
    },
    selectionHasResendData() {
      return this.selectedDeviceIds.some((id) => {
        const d = this.selectedDevicesMap[id];
        return d && this.deviceSupportsResendData(d);
      });
    },
  },

  watch: {
    deviceProfilesDialog(val, oldVal) {
      if (oldVal && !val) {
        this.loadDeviceProfiles();
      }
    },
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
    clearDeviceFilters() {
      this.filters = {
        hostname: '',
        ip: '',
        deviceProfileId: null,
        deviceStatus: null,
        rdpStatus: null,
        winrmStatus: null,
        apiStatus: null,
      };
    },

    /** Snackbar with clickable task id + open task log dialog (same as single-device actions). */
    notifyDeviceTaskQueued(taskId) {
      const id = taskId != null ? Number(taskId) : null;
      if (!id) {
        return;
      }
      EventBus.$emit('i-snackbar', {
        color: 'success',
        textPrefix: this.$t('deviceTaskQueuedPrefix'),
        textSuffix: this.$t('deviceTaskQueuedSuffix'),
        taskId: id,
      });
      EventBus.$emit('i-show-task', { taskId: id });
    },

    notifyDeviceTasksFromResponse(data) {
      if (!data) {
        return;
      }
      if (data.id != null) {
        this.notifyDeviceTaskQueued(data.id);
        return;
      }
      const tasks = data.tasks;
      if (!Array.isArray(tasks) || tasks.length === 0) {
        return;
      }
      const ids = tasks.map((t) => t && t.id).filter((id) => id != null);
      if (ids.length === 0) {
        return;
      }
      if (ids.length === 1) {
        this.notifyDeviceTaskQueued(ids[0]);
        return;
      }
      EventBus.$emit('i-snackbar', {
        color: 'success',
        textPrefix: this.$t('deviceTasksQueuedPrefix'),
        textSuffix: this.$t('deviceTasksQueuedSuffix'),
        taskIds: ids,
      });
    },

    countProfileGroupsForSelection() {
      const profileIds = new Set();
      this.selectedDeviceIds.forEach((id) => {
        const d = this.selectedDevicesMap[id];
        if (d) {
          profileIds.add(d.device_profile_id > 0 ? d.device_profile_id : 0);
        }
      });
      if (profileIds.has(0) && profileIds.size > 1) {
        return profileIds.size;
      }
      return profileIds.size || 0;
    },

    deviceActionNeedsConfirm(action) {
      return action === 'restart' || action === 'redeploy';
    },

    deviceActionLabel(action) {
      const key = {
        restart: 'deviceRestart',
        redeploy: 'deviceRedeploy',
        resend_data: 'deviceResendData',
      }[action];
      return key ? this.$t(key) : action;
    },

    deviceActionTemplateField(action) {
      if (action === 'restart') {
        return 'restart_template_id';
      }
      if (action === 'redeploy') {
        return 'redeploy_template_id';
      }
      if (action === 'resend_data') {
        return 'resend_data_template_id';
      }
      return null;
    },

    deviceActionTemplateConfigured(device, action) {
      const profileId = device && device.device_profile_id;
      const field = this.deviceActionTemplateField(action);
      if (!profileId || !field) {
        return false;
      }
      const settings = this.profileSettingsById[profileId];
      const tplId = settings && settings[field];
      return tplId != null && Number(tplId) > 0;
    },

    notifyDeviceActionTemplateMissing(action) {
      const key = {
        restart: 'deviceRestartTemplateMissing',
        redeploy: 'deviceRedeployTemplateMissing',
        resend_data: 'deviceResendTemplateMissing',
      }[action];
      if (!key) {
        return;
      }
      EventBus.$emit('i-snackbar', {
        color: 'warning',
        text: this.$t(key),
      });
    },

    bulkActionProfilesHaveTemplate(action, devices) {
      const list = devices || [];
      const profileIds = [...new Set(list.map((d) => d.device_profile_id || 0))];
      if (!profileIds.length) {
        return false;
      }
      return profileIds.every((pid) => {
        const device = list.find((d) => (d.device_profile_id || 0) === pid);
        return device && this.deviceActionTemplateConfigured(device, action);
      });
    },

    deviceProfileKey(device) {
      const profileId = device && device.device_profile_id;
      if (!profileId) {
        return '';
      }
      return String(this.profileById[profileId] || '').toUpperCase();
    },

    deviceSupportsResendData(device) {
      const key = this.deviceProfileKey(device);
      return key !== '' && RESEND_DATA_PROFILE_KEYS.has(key);
    },

    deviceResendTemplateConfigured(device) {
      return this.deviceActionTemplateConfigured(device, 'resend_data');
    },

    filterDevicesReadyForResend(devices) {
      const supported = (devices || []).filter((d) => this.deviceSupportsResendData(d));
      if (!supported.length) {
        return { ready: [], skippedUnsupported: (devices || []).length };
      }
      const ready = supported.filter((d) => this.deviceResendTemplateConfigured(d));
      return {
        ready,
        skippedUnsupported: (devices || []).length - supported.length,
        skippedNoTemplate: supported.length - ready.length,
      };
    },

    notifyResendSelectionIssues({ skippedUnsupported, skippedNoTemplate }) {
      if (skippedUnsupported > 0) {
        EventBus.$emit('i-snackbar', {
          color: 'warning',
          text: this.$t('deviceResendUnsupportedSkipped', { count: skippedUnsupported }),
        });
      }
      if (skippedNoTemplate > 0) {
        this.notifyDeviceActionTemplateMissing('resend_data');
      }
    },

    openResendDialog(devices) {
      const filtered = this.filterDevicesReadyForResend(devices);
      const { ready, skippedUnsupported, skippedNoTemplate } = filtered;
      if (!ready.length) {
        if (skippedNoTemplate > 0) {
          this.notifyDeviceActionTemplateMissing('resend_data');
        } else {
          EventBus.$emit('i-snackbar', {
            color: 'warning',
            text: this.$t('deviceResendNoEligible'),
          });
        }
        return;
      }
      if (skippedUnsupported > 0 || skippedNoTemplate > 0) {
        this.notifyResendSelectionIssues({ skippedUnsupported, skippedNoTemplate });
      }
      this.pendingResendDevices = ready;
      this.resendDataDialog = true;
    },

    selectedDevicesFull() {
      return this.selectedDeviceIds
        .map((id) => this.selectedDevicesMap[id])
        .filter(Boolean);
    },

    runBulkAction(action) {
      if (action === 'resend_data') {
        const devices = this.selectedDevicesFull();
        this.openResendDialog(devices);
        return;
      }
      if (this.deviceActionNeedsConfirm(action)) {
        const devices = this.selectedDevicesFull();
        if (!this.bulkActionProfilesHaveTemplate(action, devices)) {
          this.notifyDeviceActionTemplateMissing(action);
          return;
        }
        this.pendingDeviceAction = action;
        this.pendingDeviceActionTarget = null;
        this.pendingBulkTaskCount = this.countProfileGroupsForSelection();
        this.deviceActionConfirmDialog = true;
        return;
      }
      const n = this.countProfileGroupsForSelection();
      if (n > 1) {
        this.pendingBulkAction = action;
        this.pendingBulkTaskCount = n;
        this.bulkActionConfirmDialog = true;
        return;
      }
      this.executeBulkAction(action);
    },

    executePendingDeviceAction() {
      const action = this.pendingDeviceAction;
      const device = this.pendingDeviceActionTarget;
      this.pendingDeviceAction = null;
      this.pendingDeviceActionTarget = null;
      this.pendingBulkTaskCount = 0;
      if (!action) {
        return;
      }
      if (device) {
        this.executeDeviceAction(device, action);
        return;
      }
      this.executeBulkAction(action);
    },

    executePendingBulkAction() {
      const action = this.pendingBulkAction;
      this.pendingBulkAction = null;
      this.pendingBulkTaskCount = 0;
      if (action) {
        this.executeBulkAction(action);
      }
    },

    async executeBulkAction(action) {
      this.bulkLoading = true;
      try {
        const res = await axios.post(`${this.getItemsUrl()}/actions/bulk`, {
          action,
          device_ids: this.selectedDeviceIds,
        });
        this.notifyDeviceTasksFromResponse(res.data);
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.bulkLoading = false;
      }
    },
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
        {
          text: '',
          value: '_select',
          sortable: false,
          width: '52px',
          align: 'center',
        },
        { text: this.$i18n.t('deviceIpAddress'), value: 'ip_address', width: '18%' },
        { text: this.$i18n.t('deviceHostname'), value: 'hostname', width: '20%' },
        { text: 'Type', value: 'device_profile_id', width: '10%' },
        { text: this.$i18n.t('deviceStatus'), value: 'device_status', width: '12%' },
        { text: this.$i18n.t('deviceRdpStatus'), value: 'rdp_status', width: '9%' },
        { text: this.$i18n.t('deviceWinrmStatus'), value: 'winrm_status', width: '9%' },
        { text: this.$i18n.t('deviceApiStatus'), value: 'api_status', width: '9%' },
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

    syncSelectedDevicesMap(addedItems, removedIds) {
      const map = { ...this.selectedDevicesMap };
      if (addedItems) {
        addedItems.forEach((d) => { map[d.id] = d; });
      }
      if (removedIds) {
        removedIds.forEach((id) => { delete map[id]; });
      }
      this.selectedDevicesMap = map;
    },

    toggleSelectPage() {
      const checked = !this.pageAllSelected;
      const idsOnPage = new Set(this.pageDeviceIds);
      if (checked) {
        const merged = new Set(this.selectedDeviceIds);
        this.pageDeviceIds.forEach((id) => merged.add(id));
        this.selectedDeviceIds = [...merged];
        this.syncSelectedDevicesMap(this.items || [], null);
      } else {
        this.selectedDeviceIds = this.selectedDeviceIds.filter((id) => !idsOnPage.has(id));
        this.syncSelectedDevicesMap(null, [...idsOnPage]);
      }
    },

    toggleDeviceRow(item) {
      const i = this.selectedDeviceIds.indexOf(item.id);
      if (i >= 0) {
        this.selectedDeviceIds.splice(i, 1);
        this.syncSelectedDevicesMap(null, [item.id]);
      } else {
        this.selectedDeviceIds.push(item.id);
        this.syncSelectedDevicesMap([item], null);
      }
    },

    clearSelectedDevices() {
      const count = this.selectedDeviceIds.length;
      if (!count) {
        return;
      }
      this.selectedDeviceIds = [];
      this.selectedDevicesMap = {};
      EventBus.$emit('i-snackbar', {
        color: 'info',
        text: this.$i18n.t('deviceSelectionCleared', { count }),
      });
    },

    async loadDeviceProfiles() {
      try {
        const { data } = await axios.get(`/api/project/${this.projectId}/devices/profiles`);
        const map = {};
        (data || []).forEach((p) => {
          map[p.id] = p.profile_key;
        });
        this.profilesList = data || [];
        this.profileById = map;
        const settingsMap = {};
        await Promise.all((data || []).map(async (p) => {
          try {
            const { data: settings } = await axios.get(
              `/api/project/${this.projectId}/devices/profiles/${p.id}/settings`,
            );
            settingsMap[p.id] = settings || {};
          } catch (_) {
            settingsMap[p.id] = {};
          }
        }));
        this.profileSettingsById = settingsMap;
      } catch (_) {
        this.profilesList = [];
        this.profileById = {};
        this.profileSettingsById = {};
      }
    },

    profileLabel(profileId) {
      return this.profileById[profileId] || '—';
    },

    buildDeviceListQueryParams({
      limit, offset, sortBy, sortDesc,
    }) {
      const params = {
        limit,
        offset,
        sort: sortBy,
        order: sortDesc ? 'desc' : 'asc',
      };
      if (this.filters.hostname) params.hostname = this.filters.hostname;
      if (this.filters.ip) params.ip = this.filters.ip;
      if (this.filters.deviceProfileId) {
        params.device_profile_id = this.filters.deviceProfileId;
      }
      if (this.filters.deviceStatus) params.device_status = this.filters.deviceStatus;
      if (this.filters.rdpStatus) params.rdp_status = this.filters.rdpStatus;
      if (this.filters.winrmStatus) params.winrm_status = this.filters.winrmStatus;
      if (this.filters.apiStatus) params.api_status = this.filters.apiStatus;
      return params;
    },

    async fetchDevicesMatchingFilters() {
      const total = this.totalDevices || 0;
      if (total <= 0) {
        return [];
      }
      const opts = this.tableOptions || {};
      const sortArr = opts.sortBy || ['hostname'];
      const sortDescArr = opts.sortDesc || [false];
      const sortBy = sortArr[0] || 'hostname';
      const sortDesc = !!sortDescArr[0];
      const cap = 10000;
      const limit = Math.min(total, cap);
      const params = this.buildDeviceListQueryParams({
        limit,
        offset: 0,
        sortBy,
        sortDesc,
      });
      const { data } = await axios.get(this.getItemsUrl(), { params });
      const body = data || {};
      return (body.devices || []).filter((d) => d.id != null);
    },

    async selectAllMatchingFilters() {
      if (!this.totalDevices) {
        EventBus.$emit('i-snackbar', {
          color: 'info',
          text: this.$i18n.t('deviceSelectAllFilteredNone'),
        });
        return;
      }
      this.selectAllFilteredLoading = true;
      try {
        const devices = await this.fetchDevicesMatchingFilters();
        if (!devices.length) {
          EventBus.$emit('i-snackbar', {
            color: 'info',
            text: this.$i18n.t('deviceSelectAllFilteredNone'),
          });
          return;
        }
        const merged = new Set(this.selectedDeviceIds);
        devices.forEach((d) => merged.add(d.id));
        this.selectedDeviceIds = [...merged];
        this.syncSelectedDevicesMap(devices, null);
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceSelectAllFilteredDone', { count: devices.length }),
        });
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.selectAllFilteredLoading = false;
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

        const params = this.buildDeviceListQueryParams({
          limit: itemsPerPage,
          offset,
          sortBy,
          sortDesc,
        });

        const [devicesRes, statsRes] = await Promise.all([
          axios.get(this.getItemsUrl(), { params }),
          axios.get(`${this.getItemsUrl()}/stats`),
          this.loadDeviceProfiles(),
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

    openRemoteDesktop(device) {
      // Only open the dialog — never launch semaphore-rdp:// here.
      // Protocol open happens when the user clicks Connect inside the dialog.
      this.rdpDevice = device ? { ...device } : null;
      this.rdpDialog = false;
      this.$nextTick(() => {
        if (this.rdpDevice) {
          this.rdpDialog = true;
        }
      });
    },

    runAction(device, action) {
      if (action === 'resend_data') {
        this.openResendDialog([device]);
        return;
      }
      if (action === 'restart' || action === 'redeploy') {
        if (!this.deviceActionTemplateConfigured(device, action)) {
          this.notifyDeviceActionTemplateMissing(action);
          return;
        }
      }
      if (this.deviceActionNeedsConfirm(action)) {
        this.pendingDeviceAction = action;
        this.pendingDeviceActionTarget = device;
        this.pendingBulkTaskCount = 0;
        this.deviceActionConfirmDialog = true;
        return;
      }
      this.executeDeviceAction(device, action);
    },

    async executeResendSubmit(resend) {
      const devices = this.pendingResendDevices || [];
      if (!devices.length) {
        return;
      }
      const deviceIds = devices.map((d) => d.id);
      const single = devices.length === 1 ? devices[0] : null;
      if (single) {
        this.busyId = single.id;
      } else {
        this.bulkLoading = true;
      }
      try {
        const res = await axios.post(`${this.getItemsUrl()}/actions/bulk`, {
          action: 'resend_data',
          device_ids: deviceIds,
          resend,
        });
        this.resendDataDialog = false;
        this.pendingResendDevices = [];
        this.notifyDeviceTasksFromResponse(res.data);
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.busyId = null;
        this.bulkLoading = false;
      }
    },

    async executeDeviceAction(device, action) {
      this.busyId = device.id;
      try {
        const res = await axios.post(`${this.getItemsUrl()}/actions/bulk`, {
          action,
          device_ids: [device.id],
        });
        this.notifyDeviceTasksFromResponse(res.data);
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: getErrorMessage(e) });
      } finally {
        this.busyId = null;
      }
    },

    async runPatrol() {
      this.patrolling = true;
      try {
        const res = await axios.post(`${this.getItemsUrl()}/patrol`);
        this.notifyDeviceTasksFromResponse(res.data);
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

    openOperationHistory(device) {
      this.operationHistoryDevice = device;
      this.operationHistoryDialog = true;
    },
    showWinrmConsole(device) {
      if (!device || !device.ip_address) {
        return false;
      }
      const conn = (device.ansible_connection || 'winrm').toLowerCase();
      return conn === 'winrm';
    },
    openWinrmConsole(device) {
      this.winrmDevice = { ...device };
      this.winrmDialog = true;
    },
    onWinrmDeviceUpdated(device) {
      if (!device || !this.items) {
        return;
      }
      const idx = this.items.findIndex((d) => d.id === device.id);
      if (idx >= 0) {
        this.$set(this.items, idx, { ...this.items[idx], ...device });
      }
      if (this.winrmDevice && this.winrmDevice.id === device.id) {
        this.winrmDevice = { ...this.winrmDevice, ...device };
      }
    },
  },
};
</script>
