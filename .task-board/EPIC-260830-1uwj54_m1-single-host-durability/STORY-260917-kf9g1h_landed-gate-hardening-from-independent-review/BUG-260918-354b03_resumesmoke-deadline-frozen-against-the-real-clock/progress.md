## Status
done

## Review
required

## Task Class
code

## Estimate
notEstimated

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Fixed and landed as a12d1bd (PR #53) on 2026-09-18T00:20Z. The frozen smokeDeadline constant is gone; the deadline is derived from the real clock one hour out, with a comment naming the two-clock mechanism so it cannot be re-frozen. Verified at the exact landed head: gofmt clean, build 0, vet 0, resumesmoke ok, provhost ok, and the whole go test ./... exit 0 with 37 packages ok and zero FAIL. Found by the independent reviewer RUN-260917-e5ea66 during the TASK-260830-19bjfj story_final review and reproduced on a clean trunk extract before any change.

## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-09-18T00:16:18Z

## Last Update
2026-09-18T00:21:14Z
