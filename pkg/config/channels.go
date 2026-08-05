package config

import (
	"fmt"
	"os"
)

// Channel is one set of platform credentials — in practice one account per
// platform — serving one or more jurisdictions.
//
// The distinction between a channel and a jurisdiction is what makes the post
// budget correct. Rate limits and audience fatigue are properties of an
// account, so the per-run allowance belongs to the channel; dedup and contacts
// are properties of a body, so they belong to the jurisdiction. Conflating the
// two is how two jurisdictions end up each spending a full allowance on the
// same account.
type Channel struct {
	// Key names the channel and prefixes its environment variables.
	Key string

	// Jurisdictions are posted to this channel's accounts, in this order.
	// Order breaks ties when several are equally old.
	Jurisdictions []string

	// UsesLegacyEnv lets this channel fall back to the original unprefixed
	// variable names (X_API_KEY rather than ZURICH_X_API_KEY), so the existing
	// deployment keeps working untouched. Exactly one channel may set it:
	// without that restriction, a new channel missing a secret would silently
	// post to the first channel's account.
	UsesLegacyEnv bool
}

// channels is the configured set. Only one exists today; the model is N-channel
// from the start so that giving a future jurisdiction its own account is a
// config change rather than a rework of credential resolution.
var channels = []Channel{
	{
		Key: "zurich",
		// "Zürich Ratsinfo" is geographically accurate for both bodies and the
		// audiences overlap almost entirely, so the canton posts to the
		// existing accounts rather than needing a new credential set and a
		// second X Premium subscription.
		Jurisdictions: []string{"zurich-city", ZurichCantonKey},
		UsesLegacyEnv: true,
	},
}

// Channels returns the configured channels in order.
func Channels() []Channel {
	out := make([]Channel, len(channels))
	copy(out, channels)
	return out
}

// Env reads a channel-scoped environment variable, e.g. Env("X_API_KEY") reads
// ZURICH_X_API_KEY. Channels marked UsesLegacyEnv fall back to the unprefixed
// name; others deliberately do not, so a missing secret is an empty value
// rather than another channel's credential.
func (c Channel) Env(name string) string {
	if v := os.Getenv(envKey(c.Key) + "_" + name); v != "" {
		return v
	}
	if c.UsesLegacyEnv {
		return os.Getenv(name)
	}
	return ""
}

// EnvInt is Env for integer settings, returning fallback when unset or unparseable.
func (c Channel) EnvInt(name string, fallback int) int {
	if v, ok := envInt(envKey(c.Key) + "_" + name); ok {
		return v
	}
	if c.UsesLegacyEnv {
		if v, ok := envInt(name); ok {
			return v
		}
	}
	return fallback
}

// ResolveJurisdictions looks up every jurisdiction on this channel, in order.
func (c Channel) ResolveJurisdictions() ([]Jurisdiction, error) {
	out := make([]Jurisdiction, 0, len(c.Jurisdictions))
	for _, key := range c.Jurisdictions {
		j, err := LookupJurisdiction(key)
		if err != nil {
			return nil, fmt.Errorf("channel %q: %w", c.Key, err)
		}
		out = append(out, j)
	}
	return out, nil
}

// EnabledJurisdictions is ResolveJurisdictions filtered to those cleared to
// post. The scheduled run uses this; dry-run tools use ResolveJurisdictions,
// because previewing a body is what you do before enabling it.
func (c Channel) EnabledJurisdictions() ([]Jurisdiction, error) {
	all, err := c.ResolveJurisdictions()
	if err != nil {
		return nil, err
	}
	out := make([]Jurisdiction, 0, len(all))
	for _, j := range all {
		if j.Enabled {
			out = append(out, j)
		}
	}
	return out, nil
}

// Validate checks the configuration is internally consistent. It runs at
// startup so a misconfiguration fails loudly rather than posting somewhere
// unexpected.
func Validate() error {
	seenChannel := make(map[string]bool)
	seenJurisdiction := make(map[string]string)
	legacy := ""

	for _, c := range channels {
		if seenChannel[c.Key] {
			return fmt.Errorf("duplicate channel %q", c.Key)
		}
		seenChannel[c.Key] = true

		if c.UsesLegacyEnv {
			if legacy != "" {
				return fmt.Errorf("channels %q and %q both claim the unprefixed environment variables", legacy, c.Key)
			}
			legacy = c.Key
		}

		if len(c.Jurisdictions) == 0 {
			return fmt.Errorf("channel %q serves no jurisdictions", c.Key)
		}

		for _, key := range c.Jurisdictions {
			if other, dup := seenJurisdiction[key]; dup {
				// Two channels posting the same jurisdiction would share one
				// vote log, so whichever ran first would suppress the other.
				return fmt.Errorf("jurisdiction %q is served by both channel %q and channel %q", key, other, c.Key)
			}
			seenJurisdiction[key] = c.Key
			if _, err := LookupJurisdiction(key); err != nil {
				return fmt.Errorf("channel %q: %w", c.Key, err)
			}
		}
	}
	return nil
}
