# TASK-260922-vcx6yo: active-client-census-and-overlap-admission

## Description
SPEC v0.7.0 4.C attach row requires multi_attach for overlapping clients and multiple_input_clients plus AX policy for concurrent input. TASK-260830-1c28dz executeAttach checks operation and transport capability and then commits the receipt; neither it nor AttachStore.Attach consults existing clients or those two capabilities, and the implementation has no liveness or active-client census that could establish non-overlap. Found by RUN-260922-da502e reviewing 1c28dz CR rev7: two distinct clients attach to the same instance through Lifecycle.Execute with multi_attach removed, and two input-authorized clients both succeed with multiple_input_clients absent, both receipts persisting and the second returning a writable vector. Split out of 1c28dz because overlap admission needs an authoritative active-client/liveness contract that is a different owner, and 1c28dz is on its seventh revision.

## Scope
The active-client/receipt liveness contract and the overlap admission that consumes it, plus its capability gates. TASK-260830-1c28dz keeps the per-client attach path and fails closed where overlap cannot be established.

## Acceptance Criteria
Overlapping attach is admitted only with multi_attach, concurrent input only with multiple_input_clients and AX policy; a same-client retry is distinguished from a new client; admission is atomic where overlap is decided; the two reviewer scenarios (read-only-no-multi, writable-no-multiple-input) refuse with no second receipt committed; the census carries a row per gate.
