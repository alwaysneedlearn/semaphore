<template>
  <span>
    <v-icon
        v-if="status != null"
        small
        class="mr-1"
        :color="statusColor"
    >
      mdi-{{ statusIcon }}
    </v-icon>

    <span v-if="disabled">{{ label }}</span>

    <v-tooltip
        v-else
        right
        max-width="350"
        transition="fade-transition"
        :disabled="!tooltip"
    >
      <template v-slot:activator="{ on, attrs }">
        <a
            v-bind="attrs"
            v-on="on"
            :href="href"
            target="_blank"
            rel="noopener noreferrer"
            :class="{'task-link-with-tooltip': tooltip}"
        >
          {{ label }}
        </a>
      </template>

      <span>{{ tooltip }}</span>

    </v-tooltip>
  </span>
</template>
<style lang="scss">

@import '~vuetify/src/styles/settings/_colors.scss';

.task-link-with-tooltip {
  text-decoration: underline !important;
  text-decoration-style: dashed !important;
  text-decoration-color: gray !important;
}

a.task-link-with-tooltip {
  &:hover {
    text-decoration-style: solid !important;
    text-decoration-color: map-deep-get($blue, 'darken-2') !important;
  }
}

</style>
<script>
import taskLogPath from '@/lib/taskLog';

export default {
  props: {
    label: String,
    tooltip: String,
    taskId: Number,
    projectId: Number,
    disabled: Boolean,
    status: String,
  },
  computed: {
    resolvedProjectId() {
      return this.projectId || parseInt(this.$route.params.projectId, 10) || null;
    },
    href() {
      if (!this.resolvedProjectId || !this.taskId) {
        return '#';
      }
      return taskLogPath(this.resolvedProjectId, this.taskId);
    },
    statusColor() {
      switch (this.status) {
        case 'success':
          return 'success';
        case 'error':
          return 'red';
        default:
          return 'gray';
      }
    },
    statusIcon() {
      switch (this.status) {
        case 'success':
          return 'check';
        case 'error':
          return 'close';
        default:
          return 'clock-time-three-outline';
      }
    },
  },
};
</script>
