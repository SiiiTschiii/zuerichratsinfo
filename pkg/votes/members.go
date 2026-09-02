package votes

// Member is one sitting member of a body, as that body's own source publishes
// them. It is the roster counterpart to Vote: adapters map their API onto it,
// and everything upstream of the sources stays unaware of which API served it.
//
// Party and Fraktion are carried but deliberately never persisted anywhere.
// They change between elections, mid-term on a defection, and on a Fraktion
// merger, and a curated file that copies them acquires a second version of a
// fact the parliament already publishes. They exist to help a human verify that
// the "Anna Müller" they found on Instagram is the one who sits in this chamber
// — a question best answered against today's roster, not last year's copy.
type Member struct {
	// Name is the display name in the form the body's vote records use, which
	// is what the tagger matches against post text.
	Name string
	// Party is the member's party, e.g. "SVP". Empty when the source omits it.
	Party string
	// Fraktion is the parliamentary group they sit with, e.g. "SVP". It is not
	// always the party: EDU members sit with the SVP, EVP members with their
	// own. Empty when the source omits it or the member sits with none.
	Fraktion string
	// ProfileURL is the member's page on the parliament's own site, the first
	// place to look for a published account. Empty when the source omits it.
	ProfileURL string
	// Accounts are the social media accounts the parliament itself publishes.
	// Most bodies publish none — Stadt Zürich is the exception — in which case
	// every account in the curated mapping came from manual verification.
	Accounts []Account
}

// Account is one published social media account.
//
// Platform uses the vocabulary of the contacts mapping ("x", "facebook",
// "instagram", "linkedin", "bluesky", "tiktok"); adapters normalise into it so
// a consumer never has to know what a given API calls Twitter this year.
type Account struct {
	Platform string
	URL      string
}

// MemberSource enumerates the members currently sitting in a body.
//
// It is separate from Source because the two are read on completely different
// clocks: votes are fetched by the bot every few hours, while a roster is
// fetched by a human running a tool after an election or a Nachrücken. A source
// may implement either or both.
type MemberSource interface {
	FetchMembers() ([]Member, error)
}
