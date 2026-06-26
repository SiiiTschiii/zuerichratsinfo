// Package testfixtures provides synthetic Geschaeft test fixtures for Motion and Postulat.
// Each fixture is derived from the structure of real API responses to keep the
// implementation honest about what the API actually delivers.
package testfixtures

import (
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func intPtr(i int) *int {
	return &i
}

// MotionSimple returns a standard (non-dringliche) Motion with one Erstunterzeichner
// and four Mitunterzeichnende. Mirrors real API data where AnzahlMitunterzeichnende
// is populated but the Mitunterzeichner list itself may be absent in the search response.
func MotionSimple() zurichapi.Geschaeft {
	return zurichapi.Geschaeft{
		OBJGUID:          "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		GRNr:             "2026/142",
		Titel:            "Machbarkeitsstudie für eine Überdeckung des Autobahnabschnitts in Zürich-Brunau mittels einer Einhausung",
		Geschaeftsart:    "Motion",
		Geschaeftsstatus: "Eingereicht",
		Dringlich:        false,
		Beginn: struct {
			Start string `xml:"Start"`
			Text  string `xml:"Text"`
		}{
			Start: "2026-06-24 00:00:00",
		},
		Erstunterzeichner: struct {
			KontaktGremium zurichapi.KontaktGremium `xml:"KontaktGremium"`
		}{
			KontaktGremium: zurichapi.KontaktGremium{
				OBJGUID: "kontakt-0001",
				Name:    "Martin Busekros",
				Partei:  "Grüne",
			},
		},
		AnzahlMitunterzeichnende: intPtr(4),
	}
}

// MotionDringlich returns a Dringliche Motion. The Dringlich flag is set to true,
// which changes the emoji to 🚨 and the type label to "Dringliche Motion".
func MotionDringlich() zurichapi.Geschaeft {
	return zurichapi.Geschaeft{
		OBJGUID:          "b2c3d4e5-f6a7-8901-bcde-f12345678901",
		GRNr:             "2026/201",
		Titel:            "Sofortmassnahmen zur Verbesserung der Luftqualität in Zürich-Nord",
		Geschaeftsart:    "Motion",
		Geschaeftsstatus: "Eingereicht",
		Dringlich:        true,
		Beginn: struct {
			Start string `xml:"Start"`
			Text  string `xml:"Text"`
		}{
			Start: "2026-06-10 00:00:00",
		},
		Erstunterzeichner: struct {
			KontaktGremium zurichapi.KontaktGremium `xml:"KontaktGremium"`
		}{
			KontaktGremium: zurichapi.KontaktGremium{
				OBJGUID: "kontakt-0002",
				Name:    "Jonas Keller",
				Partei:  "SP",
			},
		},
		AnzahlMitunterzeichnende: intPtr(7),
	}
}

// MotionNoMitunterzeichnende returns a Motion where AnzahlMitunterzeichnende is nil,
// which tests the branch where the co-signer count is omitted from the submitter line.
func MotionNoMitunterzeichnende() zurichapi.Geschaeft {
	return zurichapi.Geschaeft{
		OBJGUID:          "c3d4e5f6-a7b8-9012-cdef-123456789012",
		GRNr:             "2026/078",
		Titel:            "Förderung von Solarenergie auf städtischen Gebäuden",
		Geschaeftsart:    "Motion",
		Geschaeftsstatus: "Eingereicht",
		Dringlich:        false,
		Beginn: struct {
			Start string `xml:"Start"`
			Text  string `xml:"Text"`
		}{
			Start: "2026-03-15 00:00:00",
		},
		Erstunterzeichner: struct {
			KontaktGremium zurichapi.KontaktGremium `xml:"KontaktGremium"`
		}{
			KontaktGremium: zurichapi.KontaktGremium{
				OBJGUID: "kontakt-0003",
				Name:    "Lisa Weber",
				Partei:  "GLP",
			},
		},
		AnzahlMitunterzeichnende: nil,
	}
}

// PostulatSimple returns a standard Postulat with Mitunterzeichnende.
func PostulatSimple() zurichapi.Geschaeft {
	return zurichapi.Geschaeft{
		OBJGUID:          "d4e5f6a7-b8c9-0123-defa-234567890123",
		GRNr:             "2026/155",
		Titel:            "Erstellung eines Berichts zur Wohnraumversorgung für einkommensschwache Haushalte in der Stadt Zürich",
		Geschaeftsart:    "Postulat",
		Geschaeftsstatus: "Eingereicht",
		Dringlich:        false,
		Beginn: struct {
			Start string `xml:"Start"`
			Text  string `xml:"Text"`
		}{
			Start: "2026-05-20 00:00:00",
		},
		Erstunterzeichner: struct {
			KontaktGremium zurichapi.KontaktGremium `xml:"KontaktGremium"`
		}{
			KontaktGremium: zurichapi.KontaktGremium{
				OBJGUID: "kontakt-0004",
				Name:    "Anna Graff",
				Partei:  "SP",
			},
		},
		AnzahlMitunterzeichnende: intPtr(2),
	}
}

// PostulatDringlich returns a Dringliches Postulat.
func PostulatDringlich() zurichapi.Geschaeft {
	return zurichapi.Geschaeft{
		OBJGUID:          "e5f6a7b8-c9d0-1234-efab-345678901234",
		GRNr:             "2026/189",
		Titel:            "Sofortbericht über die Situation der Obdachlosen in Zürich im Winter 2026",
		Geschaeftsart:    "Postulat",
		Geschaeftsstatus: "Eingereicht",
		Dringlich:        true,
		Beginn: struct {
			Start string `xml:"Start"`
			Text  string `xml:"Text"`
		}{
			Start: "2026-11-05 00:00:00",
		},
		Erstunterzeichner: struct {
			KontaktGremium zurichapi.KontaktGremium `xml:"KontaktGremium"`
		}{
			KontaktGremium: zurichapi.KontaktGremium{
				OBJGUID: "kontakt-0005",
				Name:    "Reto Brüesch",
				Partei:  "SVP",
			},
		},
		AnzahlMitunterzeichnende: intPtr(3),
	}
}

// MotionLongTitle returns a Motion with a title exceeding 70 graphemes to test
// root-post truncation on character-limited platforms.
func MotionLongTitle() zurichapi.Geschaeft {
	return zurichapi.Geschaeft{
		OBJGUID:          "f6a7b8c9-d0e1-2345-fabc-456789012345",
		GRNr:             "2026/099",
		Titel:            "Prüfung der Machbarkeit einer vollständigen Elektrifizierung der städtischen Fahrzeugflotte bis 2030 sowie Erarbeitung eines detaillierten Umsetzungskonzepts inklusive Ladeinfrastruktur",
		Geschaeftsart:    "Motion",
		Geschaeftsstatus: "Eingereicht",
		Dringlich:        false,
		Beginn: struct {
			Start string `xml:"Start"`
			Text  string `xml:"Text"`
		}{
			Start: "2026-04-08 00:00:00",
		},
		Erstunterzeichner: struct {
			KontaktGremium zurichapi.KontaktGremium `xml:"KontaktGremium"`
		}{
			KontaktGremium: zurichapi.KontaktGremium{
				OBJGUID: "kontakt-0006",
				Name:    "Thomas Müller",
				Partei:  "AL",
			},
		},
		AnzahlMitunterzeichnende: intPtr(1),
	}
}

// AllFixtures returns all Geschaeft test fixtures keyed by name.
func AllFixtures() map[string]zurichapi.Geschaeft {
	return map[string]zurichapi.Geschaeft{
		"motion-simple":              MotionSimple(),
		"motion-dringlich":           MotionDringlich(),
		"motion-no-mitunterzeichner": MotionNoMitunterzeichnende(),
		"postulat-simple":            PostulatSimple(),
		"postulat-dringlich":         PostulatDringlich(),
		"motion-long-title":          MotionLongTitle(),
	}
}

// FixtureNames returns fixture names in a stable order.
var FixtureNames = []string{
	"motion-simple",
	"motion-dringlich",
	"motion-no-mitunterzeichner",
	"postulat-simple",
	"postulat-dringlich",
	"motion-long-title",
}
