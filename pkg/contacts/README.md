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
  a member as known-but-not-yet-found. All 180 Kantonsrat members start that
  way.

Party and Fraktion are deliberately not stored here. The parliaments publish
both, so a copy would only go stale; `cmd/generate_search_urls` reads them live
for the curation work, and `cmd/update_contacts` refreshes the names from the
body's own roster.

The mapping is only worth having if every handle in it is right: a wrong one
puts a real person's account next to a vote they did not cast. That is why the
handles are verified by hand and `cmd/validate_contacts` guards the rest.
