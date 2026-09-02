# STORY-260902-227fzs: immutable-object-store

## Description
Reapply of the two accepted leaves of STORY-260830-2dy8oj onto current trunk. That Story branch forked from 48db30b and trunk has since advanced well past it, and the package it introduces does not exist on main - which blocks every later change to internal/localstore, including the quarantine classification fix already produced and verified against its checkpoint.

## Scope
Normative scope: §3.2, §3.3, §10.1, §10.2, §18.4.

## Acceptance Criteria
Both accepted leaves are reapplied faithfully onto current trunk as one signed commit, with byte-identity to the accepted trees asserted everywhere no conflict occurred and every deviation reported rather than absorbed.
