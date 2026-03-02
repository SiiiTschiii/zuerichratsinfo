package bluesky

import (
	"fmt"
	"log"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/bskyapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// BlueskyContent implements platforms.Content for Bluesky
type BlueskyContent struct {
	thread []*BlueskyPost // [0] = root, [1:] = replies
}

// String returns the text representation for logging/preview
func (c *BlueskyContent) String() string {
	var sb strings.Builder
	for i, post := range c.thread {
		if i == 0 {
			sb.WriteString(post.Text)
		} else {
			sb.WriteString(fmt.Sprintf("\n  ↳ Reply %d:\n%s", i, post.Text))
		}
		// Show detected mentions in preview
		for _, m := range post.Mentions {
			name := post.Text[m.ByteStart:m.ByteEnd]
			sb.WriteString(fmt.Sprintf("\n  📎 @%s → %s", m.Handle, name))
		}
	}
	return sb.String()
}

// BlueskyPlatform implements the platforms.Platform interface for Bluesky
type BlueskyPlatform struct {
	handle         string
	password       string
	session        *bskyapi.Session
	contactMapper  *contacts.Mapper
	postsThisRun   int
	maxPostsPerRun int
	didCache       map[string]string // cache handle -> DID resolutions
}

// NewBlueskyPlatform creates a new Bluesky platform poster
func NewBlueskyPlatform(handle, password string, maxPostsPerRun int, contactMapper *contacts.Mapper) *BlueskyPlatform {
	return &BlueskyPlatform{
		handle:         handle,
		password:       password,
		maxPostsPerRun: maxPostsPerRun,
		contactMapper:  contactMapper,
		postsThisRun:   0,
		didCache:       make(map[string]string),
	}
}

// ensureSession creates or reuses a Bluesky session
func (p *BlueskyPlatform) ensureSession() error {
	if p.session != nil {
		return nil
	}

	session, err := bskyapi.CreateSession(p.handle, p.password)
	if err != nil {
		return fmt.Errorf("failed to authenticate with Bluesky: %w", err)
	}

	p.session = session
	fmt.Printf("🔑 Authenticated as %s (DID: %s)\n", session.Handle, session.DID)
	return nil
}

// Format formats a group of votes into a Bluesky thread
func (p *BlueskyPlatform) Format(votes []zurichapi.Abstimmung) (platforms.Content, error) {
	thread := FormatVoteThread(votes, p.contactMapper)
	return &BlueskyContent{thread: thread}, nil
}

// Post posts a thread to Bluesky (root post + reply chain)
// Returns shouldContinue=false when the post limit is reached
func (p *BlueskyPlatform) Post(content platforms.Content) (bool, error) {
	bskyContent, ok := content.(*BlueskyContent)
	if !ok {
		return false, fmt.Errorf("unexpected content type for Bluesky")
	}

	if len(bskyContent.thread) == 0 {
		return false, fmt.Errorf("empty thread")
	}

	// Authenticate lazily on first post
	if err := p.ensureSession(); err != nil {
		return false, err
	}

	// Post the root
	root := bskyContent.thread[0]
	p.resolveMentionFacets(root)
	rootRef, err := bskyapi.CreateRecord(p.session, root.Text, root.Facets, nil)
	if err != nil {
		return false, fmt.Errorf("failed to post root: %w", err)
	}
	fmt.Printf("✅ Root post created (uri: %s)\n", rootRef.URI)

	// Post replies as a chain
	parentRef := rootRef
	for i, reply := range bskyContent.thread[1:] {
		p.resolveMentionFacets(reply)
		replyRef := &bskyapi.ReplyRef{
			Root:   *rootRef,
			Parent: *parentRef,
		}

		ref, err := bskyapi.CreateRecord(p.session, reply.Text, reply.Facets, replyRef)
		if err != nil {
			return false, fmt.Errorf("failed to post reply %d: %w", i+1, err)
		}
		fmt.Printf("  ↳ Reply %d created (uri: %s)\n", i+1, ref.URI)
		parentRef = ref
	}

	p.postsThisRun++
	shouldContinue := p.postsThisRun < p.maxPostsPerRun

	return shouldContinue, nil
}

// Name returns the platform name
func (p *BlueskyPlatform) Name() string {
	return "Bluesky"
}

// resolveMentionFacets resolves Bluesky mentions to DIDs and adds mention facets.
// Uses a cache to avoid redundant API calls.
func (p *BlueskyPlatform) resolveMentionFacets(post *BlueskyPost) {
	for _, mention := range post.Mentions {
		did, err := p.resolveHandleCached(mention.Handle)
		if err != nil {
			log.Printf("Warning: could not resolve Bluesky handle %q: %v", mention.Handle, err)
			continue
		}
		post.Facets = append(post.Facets, bskyapi.MentionFacet(mention.ByteStart, mention.ByteEnd, did))
	}
}

// resolveHandleCached resolves a handle to DID, caching results
func (p *BlueskyPlatform) resolveHandleCached(handle string) (string, error) {
	if did, ok := p.didCache[handle]; ok {
		return did, nil
	}

	did, err := bskyapi.ResolveHandle(handle)
	if err != nil {
		return "", err
	}

	p.didCache[handle] = did
	return did, nil
}
