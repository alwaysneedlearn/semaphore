<template>
  <div class="task-log-page" data-testid="taskLogDialog" role="main">
    <v-toolbar flat>
      <v-app-bar-nav-icon @click="showDrawer()"></v-app-bar-nav-icon>
      <div class="task-log-page__title text-truncate">
        <v-skeleton-loader
          v-if="template == null"
          type="button"
          style="display: inline-block; margin-right: 10px;"
        ></v-skeleton-loader>
        <router-link
          v-else
          class="breadcrumbs__item breadcrumbs__item--link"
          :to="`/project/${projectId}/templates/${template.id}`"
        >{{ template.name }}</router-link>
        <v-icon v-if="template || item" small class="mx-1">mdi-chevron-right</v-icon>
        <span class="breadcrumbs__item">{{ $t('task', { expr: item ? item.id : taskId }) }}</span>
      </div>
      <v-spacer />
      <v-btn
        icon
        data-testid="editDialog-close"
        :title="$t('close')"
        @click="close()"
      >
        <v-icon>mdi-close</v-icon>
      </v-btn>
    </v-toolbar>

    <div class="task-log-page__body">
      <TaskLogView
        v-if="item != null"
        :project-id="projectId"
        :item="item"
        :system-info="systemInfo"
        layout="page"
      />
      <v-skeleton-loader
        v-else
        class="mx-4"
        type="
            table-heading,
            image,
            list-item-two-line,
            list-item-two-line,
            list-item-two-line,
            list-item-two-line"
      ></v-skeleton-loader>
    </div>
  </div>
</template>
<style lang="scss">
.task-log-page {
  display: flex;
  flex-direction: column;
  min-height: 100dvh;
}

.task-log-page__title {
  min-width: 0;
  font-size: 1.05rem;
}

.task-log-page__body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>
<script>
import EventBus from '@/event-bus';
import TaskLogView from '@/components/TaskLogView.vue';
import ProjectMixin from '@/components/ProjectMixin';

export default {
  components: { TaskLogView },

  mixins: [ProjectMixin],

  props: {
    projectId: Number,
    systemInfo: Object,
  },

  data() {
    return {
      item: null,
      template: null,
    };
  },

  computed: {
    taskId() {
      return parseInt(this.$route.params.taskId, 10) || null;
    },
  },

  watch: {
    async taskId() {
      await this.loadData();
    },
    async projectId() {
      await this.loadData();
    },
  },

  async created() {
    await this.loadData();
  },

  methods: {
    showDrawer() {
      EventBus.$emit('i-show-drawer');
    },

    close() {
      if (window.history.length <= 1) {
        window.close();
        return;
      }
      this.$router.push({ path: `/project/${this.projectId}/history` });
    },

    async loadData() {
      if (this.projectId == null || this.taskId == null) {
        this.item = null;
        this.template = null;
        return;
      }
      this.item = await this.loadProjectResource('tasks', this.taskId);
      this.template = await this.loadProjectResource('templates', this.item.template_id);
    },
  },
};
</script>
