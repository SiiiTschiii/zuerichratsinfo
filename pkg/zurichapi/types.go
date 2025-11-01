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

// Abstimmung represents a vote from the Zurich city council
type Abstimmung struct {
	OBJGUID                 string  `xml:"OBJ_GUID,attr"`
	SEQ                     string  `xml:"SEQ,attr"`
	SitzungGuid             string  `xml:"SitzungGuid"`
	SitzungTitel            string  `xml:"SitzungTitel"`
	SitzungDatum            string  `xml:"SitzungDatum"`
	TraktandumGuid          string  `xml:"TraktandumGuid"`
	TraktandumNr            string  `xml:"TraktandumNr"`
	TraktandumTitel         string  `xml:"TraktandumTitel"`
	GeschaeftGuid           string  `xml:"GeschaeftGuid"`
	GeschaeftTitel          string  `xml:"GeschaeftTitel"`
	GeschaeftGrNr           string  `xml:"GeschaeftGrNr"`
	GeschaeftRatsgeschaeftsart string `xml:"GeschaeftRatsgeschaeftsart"`
	Abstimmungstitel        string  `xml:"Abstimmungstitel"`
	Nummer                  string  `xml:"Nummer"`
	Abstimmungstyp          string  `xml:"Abstimmungstyp"`
	AnzahlJa                *int    `xml:"Anzahl_Ja"`
	AnzahlNein              *int    `xml:"Anzahl_Nein"`
	AnzahlEnthaltung        *int    `xml:"Anzahl_Enthaltung"`
	AnzahlAbwesend          *int    `xml:"Anzahl_Abwesend"`
	Schlussresultat         string  `xml:"Schlussresultat"`
	Stimmabgaben            struct {
		Stimmabgabe []Stimmabgabe `xml:"Stimmabgabe"`
	} `xml:"Stimmabgaben"`
}

// Stimmabgabe represents an individual vote by a council member
type Stimmabgabe struct {
	OBJGUID               string `xml:"OBJ_GUID,attr"`
	KontaktGuid           string `xml:"KontaktGuid"`
	Name                  string `xml:"Name"`
	Vorname               string `xml:"Vorname"`
	Partei                string `xml:"Partei"`
	Fraktion              string `xml:"Fraktion"`
	Spezialfunktion       string `xml:"Spezialfunktion"`
	Geschlecht            string `xml:"Geschlecht"`
	Alter                 *int   `xml:"Alter"`
	Abstimmungsverhalten  string `xml:"Abstimmungsverhalten"`
}

// AbstimmungSearchResponse represents the XML response from the abstimmung API
type AbstimmungSearchResponse struct {
	XMLName xml.Name        `xml:"SearchDetailResponse"`
	NumHits int             `xml:"numHits,attr"`
	Hits    []AbstimmungHit `xml:"Hit"`
}

// AbstimmungHit represents a single result in the abstimmung search response
type AbstimmungHit struct {
	Abstimmung Abstimmung `xml:"Abstimmung"`
}