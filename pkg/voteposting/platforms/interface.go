package platforms

import "github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"

// Content represents platform-specific formatted content
type Content interface {
	// String returns a text representation for logging/preview
	String() string
}

// Platform represents a social media platform that can post vote content
type Platform interface {
	// Format converts vote groups into platform-specific content
	Format(votes []zurichapi.Abstimmung) (Content, error)

	// Post publishes content to the platform
	// Returns shouldContinue=false if posting limit is reached
	Post(content Content) (shouldContinue bool, err error)

	// MaxPostsPerRun returns the maximum number of root posts per run.
	// Used by dry-run mode to simulate the same limit as real posting.
	MaxPostsPerRun() int

	// Name returns the platform identifier (e.g., "X", "Instagram")
	Name() string
}
