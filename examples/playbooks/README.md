# Example Ansible playbooks

## `device_status_patrol_with_callback.yml`

Use as the **Status** template for device patrol / scheduled status checks. It ends with `PUT /api/project/{id}/devices/status/bulk` so devices leave the `checking` state after Patrol all.

Configure **Environment** variables on the template (or a Variable Group attached to it):

| Variable | Required | Description |
|----------|----------|-------------|
| `SEMAPHORE_PROJECT_ID` | Yes | Project id (same as `/project/<id>/devices`). |
| `SEMAPHORE_API_TOKEN` | Yes | User → API tokens → create token; passed as `Authorization: Bearer`. |
| `SEMAPHORE_URL` | No | Base URL Semaphore from the controller (default `http://127.0.0.1:3000`). |

Customize the `win_shell` process check or add tasks for API/log validation before building `patrol_update_row`.
