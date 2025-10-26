package zurichapi

import "encoding/xml"

// Geschaeft represents a council business/matter from the Zurich city council
type Geschaeft struct {
	GRNr             string `xml:"GRNr"`
	Titel            string `xml:"Titel"`
	Geschaeftsart    string `xml:"Geschaeftsart"`
	Geschaeftsstatus string `xml:"Geschaeftsstatus"`
	Beginn           struct {
		Start string `xml:"Start"`
		Text  string `xml:"Text"`
	} `xml:"Beginn"`
	Erstunterzeichner struct {
		KontaktGremium struct {
			Name   string `xml:"Name"`
			Partei string `xml:"Partei"`
		} `xml:"KontaktGremium"`
	} `xml:"Erstunterzeichner"`
}

// GeschaeftSearchResponse represents the XML response from the geschaeft API
type GeschaeftSearchResponse struct {
	XMLName xml.Name      `xml:"SearchDetailResponse"`
	NumHits int           `xml:"numHits,attr"`
	Hits    []GeschaeftHit `xml:"Hit"`
}

// GeschaeftHit represents a single result in the geschaeft search response
type GeschaeftHit struct {
	Geschaeft Geschaeft `xml:"Geschaeft"`
}

// Kontakt represents a contact from the Zurich city council
type Kontakt struct {
	OBJGUID      string `xml:"OBJ_GUID,attr"`
	SEQ          string `xml:"SEQ,attr"`
	AltsystemID  string `xml:"AltsystemID"`
	NameVorname  string `xml:"NameVorname"`
	Name         string `xml:"Name"`
	Vorname      string `xml:"Vorname"`
	Partei       string `xml:"Partei"`
	ParteiGuid   string `xml:"ParteiGuid"`
	Fraktion     string `xml:"Fraktion"`
	Wahlkreis    string `xml:"Wahlkreis"`
	Wohnkreis    string `xml:"Wohnkreis"`
	Beruf        string `xml:"Beruf"`
	Jahrgang     *int   `xml:"Jahrgang"`
	Todesjahr    *int   `xml:"Todesjahr"`
	Geschlecht   string `xml:"Geschlecht"`
	EmailGeschaeft string `xml:"EmailGeschaeft"`
	EmailPrivat    string `xml:"EmailPrivat"`
	TelefonGeschaeft string `xml:"TelefonGeschaeft"`
	TelefonPrivat    string `xml:"TelefonPrivat"`
	TelefonMobileGeschaeft string `xml:"TelefonMobileGeschaeft"`
	TelefonMobilePrivat    string `xml:"TelefonMobilePrivat"`
	HomepageGeschaeft string `xml:"HomepageGeschaeft"`
	HomepagePrivat    string `xml:"HomepagePrivat"`
	SozialeMedien SozialeMedienList `xml:"SozialeMedien"`
}

// SozialeMedienList represents a list of social media communications
type SozialeMedienList struct {
	Kommunikation []SozialesMedium `xml:"Kommunikation"`
}

// SozialesMedium represents a single social media entry
type SozialesMedium struct {
	OBJGUID string `xml:"OBJ_GUID,attr"`
	Adresse string `xml:"Adresse"`
	Typ     string `xml:"Typ"`
}

// KontaktSearchResponse represents the XML response from the kontakt API
type KontaktSearchResponse struct {
	XMLName xml.Name     `xml:"SearchDetailResponse"`
	NumHits int          `xml:"numHits,attr"`
	Hits    []KontaktHit `xml:"Hit"`
}

// KontaktHit represents a single result in the kontakt search response
type KontaktHit struct {
	Kontakt Kontakt `xml:"Kontakt"`
}