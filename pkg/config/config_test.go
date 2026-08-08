package config

import "testing"

func TestValidate(t *testing.T) {
	if err := Validate(); err != nil {
		t.Errorf("the shipped configuration must be valid: %v", err)
	}
}

func TestEnvKey(t *testing.T) {
	tests := map[string]string{
		"zurich":        "ZURICH",
		"zurich-city":   "ZURICH_CITY",
		"zurich-canton": "ZURICH_CANTON",
	}
	for in, want := range tests {
		if got := envKey(in); got != want {
			t.Errorf("envKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestChannelEnv(t *testing.T) {
	zurich := Channel{Key: "zurich"}

	t.Setenv("ZURICH_X_API_KEY", "prefixed")
	if got := zurich.Env("X_API_KEY"); got != "prefixed" {
		t.Errorf("Env = %q, want the channel-scoped value", got)
	}

	// An unprefixed variable belongs to no channel. Honouring it would let a
	// channel with a missing secret post to another channel's account.
	t.Setenv("ZURICH_X_API_KEY", "")
	t.Setenv("X_API_KEY", "unscoped")
	if got := zurich.Env("X_API_KEY"); got != "" {
		t.Errorf("Env = %q, want empty — unprefixed names are not channel credentials", got)
	}
}

func TestChannelEnvInt(t *testing.T) {
	c := Channel{Key: "zurich"}

	if got := c.EnvInt("X_MAX_POSTS_PER_RUN", 10); got != 10 {
		t.Errorf("unset should give the fallback, got %d", got)
	}

	t.Setenv("ZURICH_X_MAX_POSTS_PER_RUN", "2")
	if got := c.EnvInt("X_MAX_POSTS_PER_RUN", 10); got != 2 {
		t.Errorf("channel-scoped name should be used, got %d", got)
	}

	t.Setenv("ZURICH_X_MAX_POSTS_PER_RUN", "not a number")
	if got := c.EnvInt("X_MAX_POSTS_PER_RUN", 10); got != 10 {
		t.Errorf("an unparseable value should fall back, got %d", got)
	}
}

func TestMaxAgeDaysOverrides(t *testing.T) {
	j, err := LookupJurisdiction("zurich-city")
	if err != nil {
		t.Fatal(err)
	}
	if j.MaxAgeDays != 90 {
		t.Errorf("default city age guard = %d, want 90", j.MaxAgeDays)
	}

	t.Setenv("MAX_VOTE_AGE_DAYS_ZURICH_CITY", "7")
	if j, _ := LookupJurisdiction("zurich-city"); j.MaxAgeDays != 7 {
		t.Errorf("per-jurisdiction override ignored, got %d", j.MaxAgeDays)
	}
}

func TestLookupJurisdiction_Unknown(t *testing.T) {
	if _, err := LookupJurisdiction("no-such-body"); err == nil {
		t.Error("expected an error for an unregistered jurisdiction")
	}
}

func TestEveryJurisdictionHasASource(t *testing.T) {
	for _, key := range JurisdictionKeys() {
		j, err := LookupJurisdiction(key)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		if j.NewSource == nil {
			t.Errorf("%s has no source constructor", key)
		}
		if j.Seats <= 0 {
			t.Errorf("%s has no seat count; the completeness gate needs one", key)
		}
		// An unbounded age guard on a jurisdiction whose log may be empty is
		// how a first run posts years of history at once.
		if j.MaxAgeDays <= 0 {
			t.Errorf("%s has no age guard", key)
		}
	}
}
