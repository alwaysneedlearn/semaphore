# Ansible errors ↔ fixes (Semaphore runner)

Quick map from common ERROR lines to fixes in `cursor-playbooks/`.

| Error message | Cause | Fix |
|---------------|--------|-----|
| `'delegate_to' is not a valid attribute for a TaskInclude` | `delegate_to` / `run_once` on `include_tasks` | Parent **block** with those keys; include inside block |
| `'register' is not a valid attribute for a Block` | `register` on `block:` | Register the **module** task inside the block |
| `'until' is not a valid attribute for a TaskInclude` | Poll loop on `include_tasks` | Inline **win_shell** (or other module) with `until`/`retries` |
| Play ends at `WinRM ping` **fatal UNREACHABLE**, no bulk | Host dropped from play; callback skipped | `ignore_unreachable: true` on ping; `always` + `clear_host_errors`; failed callback + **post_tasks** bulk |
| `Failed to run task: exit status 4` | Unreachable hosts in play | Expected; ensure **post_tasks** bulk still runs; fallback rows; `max_fail_percentage: 100` for patrol |
| UI stuck **checking** | No `semaphore_callback_row` or no `SEMAPHORE_API_TOKEN` | Register row on all paths; **localhost second play** bulk; verify `[DEBUG-API] bulk PUT` |
| Batch restart success but UI unhealthy | `run_once` bulk in play1 post_tasks ran too early | Move bulk to **second play** only |

See **SKILL.md** in this directory for full patterns.
