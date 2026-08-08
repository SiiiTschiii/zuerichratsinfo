# Recorded OpenParlData responses

Captured from `api.openparldata.ch` for `body_key=ZH` (Kantonsrat Zürich),
covering the twelve most recent votings as of 2026-08-05 and every affair they
belong to.

Each object is pruned to the fields the adapter actually reads. That keeps the
fixtures reviewable — the full member-vote payloads are ~110 KB each — and makes
it obvious which parts of the API this code depends on. Adding a field to the
DTOs in `types.go` means re-recording.

The tests exist so CI never calls the live API: a third-party outage must not
fail the build, and a re-run must not depend on what the Kantonsrat did
yesterday.

To re-record, fetch each URL with `lang_format=flat` and prune to the fields in
`types.go`:

    /v1/votings/?body_key=ZH&sort_by=-date&limit=12&offset=0  -> zh_votings.json
    /v1/votings/<id>/votes?limit=500                          -> zh_votes_<id>.json
    /v1/votings/<id>/affairs                                  -> zh_affairs_<id>.json
    /v1/votings/?body_key=ZH&affair_id=<id>&limit=100         -> zh_votings_affair_<id>.json

Records are written one per line. These are captured API responses rather than
hand-maintained files, so per-field line breaks cost a reviewable diff and buy
nothing.
