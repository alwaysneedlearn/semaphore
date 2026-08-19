<template>
  <EditDialog
    v-model="dialog"
    :max-width="1000"
    :hide-buttons="true"
    :expandable="true"
    no-body-paddings
    @close="onClose()"
    test-id="taskLogDialog"
  >
    <template v-slot:title>
      <div class="text-truncate" style="max-width: calc(100% - 240px);">
        <v-skeleton-loader
          v-if="template == null"
          type="button"
          style="display: inline-block; margin-right: 10px;"
        ></v-skeleton-loader>
        <span v-else class="breadcrumbs__item">{{ template.name }}</span>
        <v-icon v-if="template || item" small class="mx-1">mdi-chevron-right</v-icon>
        <span class="breadcrumbs__item">{{ $t('task', { expr: item ? item.id : null }) }}</span>
      </div>
    </template>
    <template v-slot:title-extra>
      <v-btn
        text
        small
        class="mr-1"
        data-testid="taskLog-openNewTab"
        :title="$t('taskLogOpenNewTab')"
        :disabled="!itemId"
        @click="openInNewTab"
      >
        <v-icon left small>mdi-open-in-new</v-icon>
        {{ $t('taskLogOpenNewTab') }}
      </v-btn>
    </template>
    <template v-slot:form>
      <TaskLogView
        v-if="item != null"
        :project-id="projectId"
        :item="item"
        :system-info="systemInfo"
      />
      <v-skeleton-loader
        class="task-log-view__placeholder"
        v-else
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
    </template>
  </EditDialog>
</template>
<style lang="scss">
.task-log-view__placeholder {
  margin-left: 24px;
  margin-right: 24px;
  height: calc(100dvh - 208px);
}
</style>
<script>
import TaskLogView from '@/components/TaskLogView.vue';
import EditDialog from '@/components/EditDialog.vue';
import ProjectMixin from '@/components/ProjectMixin';
import taskLogPath from '@/lib/taskLog';

export default {
  components: { EditDialog, TaskLogView },

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
    };
  },

  methods: {
    openInNewTab() {
      if (!this.projectId || !this.itemId) {
        return;
      }
      const href = this.$router.resolve({
        path: taskLogPath(this.projectId, this.itemId),
      }).href;
      const tab = window.open(href, '_blank', 'noopener,noreferrer');
      if (!tab) {
        return;
      }
      this.dialog = false;
      this.item = null;
      this.template = null;
      this.onClose();
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
