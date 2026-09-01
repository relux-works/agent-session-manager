# Contract check — TASK-260830-1pbx0c rework cycle 3

## Pinned AX source

- Local source checkout: `/Users/iv/Developer/ReluxWorks/agent-session-manager-spec`.
- Verified exact commit: `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.
- Section 1.6 requires timestamps to be real UTC RFC3339 calendar instants with at least millisecond precision and rejects lexically valid impossible dates.
- Assigned implementation bindings for this task are Section 1.6 and Sections 10.1-10.4. Section 17.3 is a durable migration contract; this scalar package is read-only and does not claim its migration behavior.

## Leap-second authority

- IERS Bulletin C is the primary announcement channel for UTC leap seconds: https://datacenter.iers.org/productMetadata.php?id=16
- IERS Bulletin C 72 states that no leap second will be introduced at the end of December 2026 and that UTC-TAI remains unchanged since 2017-01-01: https://datacenter.iers.org/data/html/bulletinc-072.html
- The implementation embeds the 27 published positive UTC leap-second dates from 1972-06-30 through 2016-12-31. It accepts `1990-12-31T23:59:60.000Z` and refuses `:60` outside 23:59, on an unpublished date, or on 2026-12-31.

## Representation decision

Go `time.Time` does not encode leap seconds. `Timestamp` preserves the exact accepted wire text. `Timestamp.Time()` parses the preceding representable second and adds one second, mapping the leap second onto the following UTC second without allowing a fabricated value through validation.
