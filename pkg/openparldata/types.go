package openparldata

// The DTOs below cover only the fields this adapter reads. OpenParlData returns
// considerably more; adding a field here is the cheap part, deciding what it
// means across 20-odd cantons is not.
//
// Every localised field is read in its `_de` form, which requires
// lang_format=flat on the request (see Client.get).

type votingsResponse struct {
	Data []votingDTO `json:"data"`
	Meta meta        `json:"meta"`
}

type meta struct {
	Offset       int  `json:"offset"`
	Limit        int  `json:"limit"`
	TotalRecords int  `json:"total_records"`
	HasMore      bool `json:"has_more"`
}

type votingDTO struct {
	ID         int64  `json:"id"`
	ExternalID string `json:"external_id"`
	BodyKey    string `json:"body_key"`
	Date       string `json:"date"`
	AffairID   *int64 `json:"affair_id"`

	ResultsYes        *int `json:"results_yes"`
	ResultsNo         *int `json:"results_no"`
	ResultsAbstention *int `json:"results_abstention"`
	ResultsAbsent     *int `json:"results_absent"`

	// Decision is frequently null — Kanton Zürich never populates it — and is
	// then derived from the counts.
	Decision *string `json:"decision"`

	TitleDe       *string `json:"title_de"`
	AffairTitleDe *string `json:"affair_title_de"`
	TypeDe        *string `json:"type_de"`
	URLExternalDe *string `json:"url_external_de"`
}

type votesResponse struct {
	Data []voteDTO `json:"data"`
	Meta meta      `json:"meta"`
}

type voteDTO struct {
	ID int64 `json:"id"`

	// Vote is the harmonised value ("yes", "no", "abstention", "absent") and is
	// what this adapter maps from. VoteDisplayDe is the source's own label and
	// varies between bodies — Kanton Zürich says "Enthalten" where Stadt Zürich
	// says "Enthaltung" — so it is only a fallback for values we do not know.
	Vote          string  `json:"vote"`
	VoteDisplayDe *string `json:"vote_display_de"`

	PersonFullname                 *string `json:"person_fullname"`
	PersonPartyDe                  *string `json:"person_party_de"`
	PersonParliamentaryGroupNameDe *string `json:"person_parliamentary_group_name_de"`
}

type affairsResponse struct {
	Data []affairDTO `json:"data"`
}

// affairDTO is fetched separately: affair_number is null inline on a voting,
// so the human-readable business number requires a second call.
type affairDTO struct {
	ID            int64   `json:"id"`
	Number        *string `json:"number"`
	TitleDe       *string `json:"title_de"`
	URLExternalDe *string `json:"url_external_de"`
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
