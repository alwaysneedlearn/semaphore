<template>
  <div v-if="items != null">

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
        :loading="discovering"
        @click="discoverDevices"
      >
        <v-icon left>mdi-radar</v-icon>
        {{ $t('deviceDiscover') }}
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
              <v-icon small color="success">mdi-checkbox-marked-circle</v-icon>
              {{ $t('deviceRdpOnline') }}
            </div>
            <div class="text-h4">{{ stats.rdp_online }}</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="6" sm="3">
        <v-card outlined>
          <v-card-text class="pa-3">
            <div class="text-overline">
              <v-icon small color="success">mdi-checkbox-marked-circle</v-icon>
              {{ $t('deviceWinrmOnline') }}
            </div>
            <div class="text-h4">{{ stats.winrm_online }}</div>
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
        total: 0, rdp_online: 0, winrm_online: 0, unknown: 0,
      },
      discovering: false,
      busyId: null,
      configDialog: false,
      configDeviceId: null,
      configDeviceName: '',
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
      if (s === 'online') return 'success';
      if (s === 'offline') return 'error';
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
        { text: this.$i18n.t('name'), value: 'name', width: '15%' },
        { text: this.$i18n.t('deviceIpAddress'), value: 'ip_address', width: '12%' },
        { text: this.$i18n.t('deviceHostname'), value: 'hostname', width: '20%' },
        { text: this.$i18n.t('deviceRdpStatus'), value: 'rdp_status', width: '10%' },
        { text: this.$i18n.t('deviceWinrmStatus'), value: 'winrm_status', width: '10%' },
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

    openConfigDialog(device) {
      this.configDeviceId = device.id;
      this.configDeviceName = device.name;
      this.configDialog = true;
    },
  },
};
</script>
