# Contacts Package

Maps politicians to their social media accounts, and tags them in posts.

One curated file per jurisdiction — `data/zurich-city/contacts.yaml`,
`data/zurich-canton/contacts.yaml` — with an identical schema: a name, and the
accounts verified for it. A channel serving more than one body loads them
together (`LoadContactFiles`).

Tagging is name matching: the mapper looks for a contact's name in the text of a
post and appends their handle. Two things follow from that.

- A post that names nobody tags nobody. Stadt Zürich's titles carry their
  submitters, so the city has always worked; Kanton Zürich's do not, and its
  authorship is put back onto the label line by the formatters — see
  `votes.Affair.Authors`.
- An entry with no handles is valid and useful. It costs nothing, and it marks
  a member as known-but-not-yet-found.

## Verified accounts

An account is posted only once it carries `verified: true` — a human opened it
and recognised the person. `Contact.Verified(platform)` is the single gate every
reader goes through, so an unverified handle cannot reach a post by someone
forgetting to check a flag.

That gate is what lets candidates live in the same file as confirmed handles.
Curating 180 cantonal members is weeks of one-by-one checking; the alternative
was keeping the leads somewhere the tooling could not see, which meant the work
could not start until it was finished.

Two shapes, both valid:

```yaml
  - name: Anna Graff
    x:
      - https://x.com/annagraff        # a bare string is verified
      - url: https://x.com/maybe       # a candidate: not posted
        verified: false
        confidence: high
```

A bare string is verified because that is what every handle meant before
candidates existed — each got there because a human put it there. In the mapping
form `verified` defaults to false, so the defaults fall the safe way round:
publishing takes a deliberate act, and forgetting the flag leaves a lead silent.
`confidence` is an ordering aid on unverified entries only; drop it when you
confirm the account.

Party and Fraktion are deliberately not stored here. The parliaments publish
both, so a copy would only go stale; `cmd/generate_search_urls` reads them live
for the curation work, and `cmd/update_contacts` refreshes the names from the
body's own roster.

The mapping is only worth having if every handle in it is right: a wrong one
puts a real person's account next to a vote they did not cast. That is why the
handles are verified by hand and `cmd/validate_contacts` guards the rest.
