<template>
  <div class="task-log-page" data-testid="taskLogDialog" role="main">
    <v-btn
      icon
      class="task-log-page__close"
      data-testid="editDialog-close"
      :title="$t('close')"
      @click="close()"
    >
      <v-icon>mdi-close</v-icon>
    </v-btn>

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
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 100dvh;
}

.task-log-page__close {
  position: absolute;
  top: 4px;
  right: 8px;
  z-index: 2;
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
        return;
      }
      this.item = await this.loadProjectResource('tasks', this.taskId);
    },
  },
};
</script>
