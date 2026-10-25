package main

import (
	"slices"
	"testing"

	"github.com/siiitschiii/parlinfo/pkg/config"
	"github.com/siiitschiii/parlinfo/pkg/contacts"
	"github.com/siiitschiii/parlinfo/pkg/votes"
)

func TestPartition_AMoverIsNotALeaver(t *testing.T) {
	elsewhere := map[string]string{contacts.NameKey("Nuñez David Garcia"): "zurich-canton"}

	leavers, movers := partition([]string{"David Garcia Nuñez", "Weg Gegangen"}, elsewhere)

	if !slices.Equal(leavers, []string{"Weg Gegangen"}) {
		t.Errorf("leavers = %v, want only the person who sits nowhere", leavers)
	}
	if movers["David Garcia Nuñez"] != "zurich-canton" || len(movers) != 1 {
		t.Errorf("movers = %v, want David Garcia Nuñez in zurich-canton", movers)
	}
}

func TestPrune_RemovesOnlyTheNamedPeople(t *testing.T) {
	existing := map[string]*Contact{
		contacts.NameKey("Anna Aktiv"):      {Name: "Anna Aktiv"},
		contacts.NameKey("Weg Gegangen"):    {Name: "Weg Gegangen"},
		contacts.NameKey("SP Stadt Zürich"): {Name: "SP Stadt Zürich", Kind: contacts.KindOrganization},
	}

	prune(existing, []string{"Gegangen Weg"})

	if _, ok := existing[contacts.NameKey("Weg Gegangen")]; ok {
		t.Errorf("the named person was not removed")
	}
	if len(existing) != 2 {
		t.Errorf("others were touched: %v", existing)
	}
}

func TestCheckPruneLimit_RefusesWhatLooksLikeABrokenSource(t *testing.T) {
	if err := checkPruneLimit(25, 25); err != nil {
		t.Errorf("at the limit should pass: %v", err)
	}
	if err := checkPruneLimit(26, 25); err == nil {
		t.Errorf("over the limit should be refused")
	}
}

type rosterOnly struct{}

func (rosterOnly) FetchMembers() ([]votes.Member, error) { return nil, nil }

func TestSittingNames_AJurisdictionThatCannotSayIsAnError(t *testing.T) {
	for name, j := range map[string]config.Jurisdiction{
		"no roster":                 {Jurisdiction: votes.Jurisdiction{Key: "a"}},
		"a roster without who sits": {Jurisdiction: votes.Jurisdiction{Key: "b"}, NewMemberSource: func() votes.MemberSource { return rosterOnly{} }},
	} {
		t.Run(name, func(t *testing.T) {
			if names, err := sittingNames(j); err == nil {
				t.Errorf("sittingNames = %v, nil; want an error, not an empty chamber", names)
			}
		})
	}
}
