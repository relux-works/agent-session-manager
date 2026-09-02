# TASK-260830-2iint0 configuration gate mutation results

Each production clause was disabled individually and tested with `go test ./internal/config -count=1`.

| Mutant | go test exit | Result |
| --- | ---: | --- |
| flag_precedence | 1 | reddened as required |
| empty_environment_is_unset | 1 | reddened as required |
| unknown_override_refusal | 1 | reddened as required |
| invalid_platform_refusal | 1 | reddened as required |
| linux_runtime_default_required | 1 | reddened as required |
| windows_appdata_required | 1 | reddened as required |
| windows_localappdata_required | 1 | reddened as required |
| config_absence_distinction | 1 | reddened as required |
| config_stat_failure_distinction | 1 | reddened as required |
| config_regular_file_gate | 1 | reddened as required |
| config_read_failure_distinction | 1 | reddened as required |
| root_inspection_failure_distinction | 1 | reddened as required |
| root_directory_kind_gate | 1 | reddened as required |
| secret_environment_not_registered | 1 | reddened as required |

Source restoration cmp exit: 0.

