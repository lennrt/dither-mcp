## Decisions

Cell recipes copy resolved reusable settings, then substitute the candidate
algorithm or palette and actual rendered dimensions. They never include image
bytes or source, mask, or output paths. Image masks retain their external-mask
requirement. The shared artifact recipe remains compatible with existing callers.

CLI conflict detection uses JSON engine keys and explicit flag presence. Explicit
zero and false values count. Matching follows Go's case-insensitive field matching.
Distinct settings still combine. Rejection happens before workspace access.

An admitted batch retains successful artifacts and appends cancellation errors for
all unstarted indices. Cancellation before service admission still fails the call.
A transport that stops waiting cannot receive a response.

## Model scope

The publication model checks that cancellation of an unstarted item preserves an
earlier commit and prevents later admission. It abstracts receipts, paths, and
encoded bytes. Go tests check receipt count, ordering, and service delivery.
Comparison recipe metadata and CLI parsing do not alter modeled selection authority.
