# EPIC-260830-27bix9: m3-daily-driver-tmux

## Description
Harden the tmux-first macOS/Linux product slice into the first daily-driver release gate.

## Scope
AX v0.5.0 M3; SPEC sections 4, 13-16, 18-19. macOS arm64 and Linux amd64 are mandatory; no claim for cloning, Directory, or native Windows yet.

## Acceptance Criteria
The required tmux lanes pass credential, reboot, takeover, continuation UX, security, and sustained-use gates with no duplicate owner or silent fresh session.
