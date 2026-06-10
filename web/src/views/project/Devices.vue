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
          <v-list-item @click="runBulkAction('redeploy')">
            <v-list-item-icon><v-icon>mdi-package-down</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceRedeploy') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="runBulkAction('status')">
            <v-list-item-icon><v-icon>mdi-stethoscope</v-icon></v-list-item-icon>
            <v-list-item-title>{{ $t('deviceStatusCheck') }}</v-list-item-title>
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

    <YesNoDialog
      title="Confirm bulk action"
      :text="bulkActionConfirmText"
      v-model="bulkActionConfirmDialog"
      @yes="executePendingBulkAction()"
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
              <v-list-item @click="runAction(item, 'redeploy')">
                <v-list-item-icon><v-icon>mdi-package-down</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceRedeploy') }}</v-list-item-title>
              </v-list-item>
              <v-list-item @click="runAction(item, 'status')">
                <v-list-item-icon><v-icon>mdi-stethoscope</v-icon></v-list-item-icon>
                <v-list-item-title>{{ $t('deviceStatusCheck') }}</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-menu>
          <v-btn :title="$t('deviceConfig')" @click="openConfigDialog(item)">
            <v-icon>mdi-cog</v-icon>
          </v-btn>
          <v-btn
            :title="$t('deviceAbnormalReason')"
            :disabled="item.device_status === 'healthy'"
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
import DeviceWinrmConsoleDialog from '@/components/DeviceWinrmConsoleDialog.vue';
import DeviceProfilesForm from '@/components/DeviceProfilesForm.vue';
import { getErrorMessage } from '@/lib/error';

export default {
  mixins: [ItemListPageBase],
  components: {
    DeviceForm,
    DeviceConfigDialog,
    DeviceWinrmConsoleDialog,
    DeviceProfilesForm,
  },

  data() {
    return {
      stats: {
        total: 0, healthy: 0, unhealthy: 0, checking: 0, unknown: 0,
      },
      patrolling: false,
      deviceProfilesDialog: false,
      busyId: null,
      configDialog: false,
      configDeviceId: null,
      configDeviceName: '',
      winrmDialog: false,
      winrmDevice: null,
      reasonDialog: false,
      reasonError: '',
      reasonDeviceHostname: '',
      reasonData: {},
      reasonHeaders: [
        { text: this.$i18n.t('deviceLastUpdated'), value: 'created' },
        { text: this.$i18n.t('deviceStatus'), value: 'status' },
        { text: this.$i18n.t('deviceRdpStatus'), value: 'rdp_status' },
        { text: this.$i18n.t('deviceWinrmStatus'), value: 'winrm_status' },
        { text: this.$i18n.t('deviceApiStatus'), value: 'api_status' },
        { text: this.$i18n.t('deviceAbnormalReason'), value: 'abnormal_reason' },
      ],
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
      bulkLoading: false,
      selectAllFilteredLoading: false,
      bulkDeleteDialog: false,
      bulkActionConfirmDialog: false,
      pendingBulkAction: null,
      pendingBulkTaskCount: 0,
      profilesList: [],
      profileById: {},
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
      const selected = new Set(this.selectedDeviceIds);
      const profileIds = new Set();
      (this.items || []).forEach((d) => {
        if (!selected.has(d.id)) {
          return;
        }
        profileIds.add(d.device_profile_id > 0 ? d.device_profile_id : 0);
      });
      if (profileIds.has(0) && profileIds.size > 1) {
        return profileIds.size;
      }
      return profileIds.size || 0;
    },

    runBulkAction(action) {
      const n = this.countProfileGroupsForSelection();
      if (n > 1) {
        this.pendingBulkAction = action;
        this.pendingBulkTaskCount = n;
        this.bulkActionConfirmDialog = true;
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

    clearSelectedDevices() {
      const count = this.selectedDeviceIds.length;
      if (!count) {
        return;
      }
      this.selectedDeviceIds = [];
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
      } catch (_) {
        this.profilesList = [];
        this.profileById = {};
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

    async fetchDeviceIdsMatchingFilters() {
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
      return (body.devices || []).map((d) => d.id).filter((id) => id != null);
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
        const ids = await this.fetchDeviceIdsMatchingFilters();
        if (!ids.length) {
          EventBus.$emit('i-snackbar', {
            color: 'info',
            text: this.$i18n.t('deviceSelectAllFilteredNone'),
          });
          return;
        }
        const merged = new Set(this.selectedDeviceIds);
        ids.forEach((id) => merged.add(id));
        this.selectedDeviceIds = [...merged];
        EventBus.$emit('i-snackbar', {
          color: 'success',
          text: this.$i18n.t('deviceSelectAllFilteredDone', { count: ids.length }),
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

    async runAction(device, action) {
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
