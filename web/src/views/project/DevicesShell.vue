<template>
  <div>
    <v-tabs class="pl-4" show-arrows>
      <v-tab :to="`/project/${projectId}/devices/list`">
        {{ $t('deviceTabList') }}
      </v-tab>
      <v-tab
        v-if="can(USER_PERMISSIONS.manageProjectResources)"
        :to="`/project/${projectId}/devices/discovery`"
      >
        {{ $t('deviceTabDiscovery') }}
      </v-tab>
    </v-tabs>
    <v-divider style="margin-top: -1px;" />
    <router-view />
  </div>
</template>

<script>
import PermissionsCheck from '@/components/PermissionsCheck';
import ProjectMixin from '@/components/ProjectMixin';

export default {
  mixins: [PermissionsCheck, ProjectMixin],
};
</script>
