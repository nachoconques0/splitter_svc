# The Bill carries a version, and a stale replace is rejected

`PUT /bills/{id}/shares` replaces the whole Share Set, so two people editing one
Bill would otherwise clobber each other silently. The Bill carries a version that
clients read on `GET` and send back on `PUT`; the write is guarded by
`UPDATE bills SET version = version + 1 WHERE id = $1 AND version = $2`, and zero
rows affected rolls the transaction back and returns `409 bill_modified`.

## Consequences

The version sits on the Bill because the Bill is the aggregate root that owns its
Shares. `POST /people` never touches it — Person is a separate aggregate.

A stale `PUT` is rejected even when it would have written an identical Share Set.
Comparing payloads for semantic equality is a rabbit hole for a case that
essentially does not occur.

On conflict the UI shows a banner and leaves the user's edits on screen rather
than refetching over them. Silently replacing what someone typed turns a conflict
into data loss, which is worse than the problem being solved.

Note that authentication is out of scope, so the shipped UI cannot actually
produce two concurrent users. This is defensive: the endpoint's replace-the-whole-
set shape makes the failure mode bad enough to be worth closing off cheaply.
