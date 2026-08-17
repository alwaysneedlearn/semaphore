export default function taskLogPath(projectId, taskId) {
  return `/project/${projectId}/history/${taskId}`;
}
