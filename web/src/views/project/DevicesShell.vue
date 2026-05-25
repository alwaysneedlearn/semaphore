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
    <router-view
      :project-id="projectId"
      :project-type="projectType"
      :user-permissions="userPermissions"
      :user-role="userRole"
      :user-id="userId"
      :is-admin="isAdmin"
      :user="user"
      :premium-features="premiumFeatures"
      :auth-methods="authMethods"
      :system-info="systemInfo"
    />
  </div>
</template>

<script>
import PermissionsCheck from '@/components/PermissionsCheck';
import ProjectMixin from '@/components/ProjectMixin';
import { USER_PERMISSIONS } from '@/lib/constants';

export default {
  mixins: [PermissionsCheck, ProjectMixin],

  props: {
    projectId: Number,
    projectType: String,
    userPermissions: Number,
    userRole: String,
    userId: Number,
    isAdmin: Boolean,
    user: Object,
    premiumFeatures: Object,
    authMethods: Object,
    systemInfo: Object,
  },

  data() {
    return {
      USER_PERMISSIONS,
    };
  },
};
</script>
