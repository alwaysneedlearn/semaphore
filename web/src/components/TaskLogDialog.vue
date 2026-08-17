<template>
  <!--
    Right-side task log panel (not a modal). No overlay, so the page behind
    (e.g. Devices) stays usable while watching task output.
  -->
  <v-navigation-drawer
    v-model="dialog"
    app
    right
    fixed
    :width="panelWidth"
    hide-overlay
    disable-resize-watcher
    class="task-log-drawer"
    :class="{ 'task-log-drawer--wide': wide }"
    data-testid="taskLogDialog"
    role="dialog"
    aria-modal="false"
  >
    <div class="task-log-drawer__header">
      <div class="task-log-drawer__title text-truncate">
        <v-skeleton-loader
          v-if="template == null && dialog"
          type="button"
          style="display: inline-block; margin-right: 10px;"
        ></v-skeleton-loader>
        <router-link
          v-else-if="template"
          class="breadcrumbs__item breadcrumbs__item--link"
          :to="`/project/${projectId}/templates/${template.id}`"
          @click="close()"
        >{{ template.name }}</router-link>
        <v-icon v-if="template || item" small class="mx-1">mdi-chevron-right</v-icon>
        <span class="breadcrumbs__item">{{ $t('task', { expr: item ? item.id : null }) }}</span>
      </div>

      <div class="task-log-drawer__actions">
        <v-btn
          icon
          small
          class="mr-1"
          :title="wide ? $t('taskLogPanelNarrow') : $t('taskLogPanelWiden')"
          @click="wide = !wide"
        >
          <v-icon small>{{ wide ? 'mdi-arrow-collapse-horizontal' : 'mdi-arrow-expand-horizontal' }}</v-icon>
        </v-btn>
        <v-btn
          icon
          small
          data-testid="editDialog-close"
          :title="$t('close')"
          @click="close()"
        >
          <v-icon small>mdi-close</v-icon>
        </v-btn>
      </div>
    </div>

    <div class="task-log-drawer__body">
      <TaskLogView
        v-if="item != null"
        :project-id="projectId"
        :item="item"
        :system-info="systemInfo"
        layout="drawer"
      />

      <v-skeleton-loader
        v-else
        class="task-log-view__placeholder"
        type="
            table-heading,
            image,
            list-item-two-line,
            list-item-two-line,
            list-item-two-line,
            list-item-two-line,
            list-item-two-line,
            list-item-two-line,
            list-item-two-line,
            list-item-two-line"
      ></v-skeleton-loader>
    </div>
  </v-navigation-drawer>
</template>
<style lang="scss">
.task-log-drawer {
  z-index: 6 !important;
  display: flex;
  flex-direction: column;
  border-left: 1px solid rgba(0, 0, 0, 0.12) !important;
}

.theme--dark .task-log-drawer {
  border-left-color: rgba(255, 255, 255, 0.12) !important;
}

.task-log-drawer__header {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  min-height: 56px;
  padding: 8px 8px 8px 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.08);
  gap: 8px;
}

.theme--dark .task-log-drawer__header {
  border-bottom-color: rgba(255, 255, 255, 0.08);
}

.task-log-drawer__title {
  flex: 1;
  min-width: 0;
  font-size: 0.95rem;
}

.task-log-drawer__actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.task-log-drawer__body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.task-log-view__placeholder {
  margin-left: 16px;
  margin-right: 16px;
  height: calc(100dvh - 72px);
}
</style>
<script>
import TaskLogView from '@/components/TaskLogView.vue';
import ProjectMixin from '@/components/ProjectMixin';

const PANEL_WIDTH_NARROW = 560;
const PANEL_WIDTH_WIDE = 900;

export default {
  components: { TaskLogView },

  mixins: [ProjectMixin],

  props: {
    value: Boolean,
    projectId: Number,
    itemId: Number,
    systemInfo: Object,
  },

  watch: {
    async dialog(val) {
      this.$emit('input', val);
      if (!val) {
        this.onClose();
      }
    },

    async value(val) {
      this.item = null;
      this.template = null;
      this.dialog = val;
      if (val) {
        await this.loadData();
      }
    },

    async itemId() {
      if (this.dialog) {
        await this.loadData();
      }
    },
  },

  data() {
    return {
      item: null,
      dialog: false,
      template: null,
      wide: false,
    };
  },

  computed: {
    panelWidth() {
      return this.wide ? PANEL_WIDTH_WIDE : PANEL_WIDTH_NARROW;
    },
  },

  methods: {
    close() {
      this.dialog = false;
      this.item = null;
      this.template = null;
      this.wide = false;
    },

    async loadData() {
      if (this.itemId == null) {
        return;
      }
      this.item = await this.loadProjectResource('tasks', this.itemId);
      this.template = await this.loadProjectResource('templates', this.item.template_id);
    },

    onClose() {
      this.$emit('close');
    },
  },
};
</script>
