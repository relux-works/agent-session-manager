# TASK-260901-eqnoeh: admit-opus-reviewer-runtime

## Description
The codex reviewer was blocked four consecutive times by the provider content filter on its own adversarial probes, which is inherent to the reviewer role in this project: it attacks validation gates with deliberately malformed input. The project config admits only codex, so there was no viable fallback and the block had to be worked around with an uncommitted config edit.

## Scope
spawn.preferred_agentic_system and spawn.ceilings in task-board.config.json only.

## Acceptance Criteria
The reviewer role can run on claude-opus-5 at high effort; codex remains rank 1 in the review workload class so the change is a fallback rather than a replacement; spawn preflight admits both providers.
