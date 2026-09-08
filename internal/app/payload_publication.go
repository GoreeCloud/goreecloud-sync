package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/GoreeCloud/goreecloud-sync/internal/transfer"
)

var (
	ErrPayloadPublicationAuthorizationDenied = errors.New("payload publication authorization denied")
	ErrPayloadDestinationExists              = errors.New("payload destination already exists")
)

// PayloadPublicationRequest is the bounded local authorization request for
// publishing one already-received payload into a caller-selected destination.
// DestinationID is an opaque local identifier; the remote peer never supplies
// or learns the destination filesystem root through this contract.
type PayloadPublicationRequest struct {
	DestinationID string
	TransferID    string
	Kind          transfer.PayloadKind
	Filename      string
	Size          int64
	Hash          string
}

// PayloadDestinationAuthorizer independently authorizes the final local
// destination after transport verification. Transport trust, offer acceptance,
// and a verified payload receipt do not imply this authorization.
type PayloadDestinationAuthorizer func(PayloadPublicationRequest) error

// PublishedPayload is a bounded local publication result. It intentionally does
// not expose the physical filesystem root.
type PublishedPayload struct {
	DestinationID string
	TransferID    string
	Filename      string
	Size          int64
	Hash          string
}

// LocalPayloadPublisher confines final publication to one local directory that
// is selected by trusted local application composition rather than by the peer.
// The root is canonicalized at construction and revalidated before every
// publication so a later symlink substitution fails closed.
type LocalPayloadPublisher struct {
	destinationID string
	root          string
	authorize     PayloadDestinationAuthorizer
}

// NewLocalPayloadPublisher constructs a final-publication boundary for one
// existing local directory. The caller remains responsible for selecting a root
// whose ownership and permissions satisfy the surrounding product policy.
func NewLocalPayloadPublisher(destinationID, root string, authorize PayloadDestinationAuthorizer) (*LocalPayloadPublisher, error) {
	if err := validateLocalDestinationID(destinationID); err != nil {
		return nil, err
	}
	if authorize == nil {
		return nil, fmt.Errorf("payload destination authorizer must not be nil")
	}
	canonicalRoot, err := canonicalPublicationRoot(root)
	if err != nil {
		return nil, err
	}
	return &LocalPayloadPublisher{
		destinationID: destinationID,
		root:          canonicalRoot,
		authorize:     authorize,
	}, nil
}

// Publish independently validates the offer/receipt binding, obtains explicit
// local destination authorization, re-verifies the exact staged bytes while
// copying them into a private temporary file inside the confined root, and then
// atomically links that verified file into the final leaf name without
// overwriting an existing destination.
//
// A successful network receipt alone is never sufficient. The staged content is
// re-read against the manifest because callers may persist or transform staging
// between receipt generation and publication.
func (p *LocalPayloadPublisher) Publish(offer transfer.PayloadOffer, receipt transfer.PayloadReceipt, staged io.Reader) (PublishedPayload, error) {
	if p == nil || p.authorize == nil || p.root == "" {
		return PublishedPayload{}, fmt.Errorf("payload publisher is not initialized")
	}
	if staged == nil {
		return PublishedPayload{}, fmt.Errorf("staged payload reader must not be nil")
	}
	if err := offer.Validate(); err != nil {
		return PublishedPayload{}, fmt.Errorf("validate payload offer: %w", err)
	}
	if err := receipt.ValidateFor(offer); err != nil {
		return PublishedPayload{}, fmt.Errorf("validate payload receipt: %w", err)
	}
	if err := p.revalidateRoot(); err != nil {
		return PublishedPayload{}, err
	}

	request := PayloadPublicationRequest{
		DestinationID: p.destinationID,
		TransferID:    offer.TransferID,
		Kind:          offer.Kind,
		Filename:      offer.Manifest.Filename,
		Size:          offer.Manifest.Size,
		Hash:          offer.Manifest.Hash,
	}
	if err := p.authorize(request); err != nil {
		return PublishedPayload{}, fmt.Errorf("%w: %v", ErrPayloadPublicationAuthorizationDenied, err)
	}

	target, err := p.targetForFilename(offer.Manifest.Filename)
	if err != nil {
		return PublishedPayload{}, err
	}
	temp, err := os.CreateTemp(p.root, ".goreecloud-sync-staged-*")
	if err != nil {
		return PublishedPayload{}, fmt.Errorf("create publication temporary file: %w", err)
	}
	tempName := temp.Name()
	keepTemp := true
	defer func() {
		_ = temp.Close()
		if keepTemp {
			_ = os.Remove(tempName)
		}
	}()
	if err := temp.Chmod(0o600); err != nil {
		return PublishedPayload{}, fmt.Errorf("secure publication temporary file: %w", err)
	}

	if err := transfer.VerifyPayload(offer.Manifest, io.TeeReader(staged, temp)); err != nil {
		return PublishedPayload{}, fmt.Errorf("verify staged payload before publication: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return PublishedPayload{}, fmt.Errorf("sync verified publication temporary file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return PublishedPayload{}, fmt.Errorf("close verified publication temporary file: %w", err)
	}

	if err := os.Link(tempName, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			return PublishedPayload{}, ErrPayloadDestinationExists
		}
		return PublishedPayload{}, fmt.Errorf("publish verified payload: %w", err)
	}

	published := PublishedPayload{
		DestinationID: p.destinationID,
		TransferID:    offer.TransferID,
		Filename:      offer.Manifest.Filename,
		Size:          offer.Manifest.Size,
		Hash:          offer.Manifest.Hash,
	}
	if err := os.Remove(tempName); err != nil {
		// The final target has already been atomically published. Return the
		// publication result together with the cleanup failure so callers do not
		// mistake the error for proof that no side effect occurred.
		keepTemp = false
		return published, fmt.Errorf("payload published but temporary cleanup failed: %w", err)
	}
	keepTemp = false
	return published, nil
}

func (p *LocalPayloadPublisher) targetForFilename(filename string) (string, error) {
	target := filepath.Join(p.root, filename)
	relative, err := filepath.Rel(p.root, target)
	if err != nil {
		return "", fmt.Errorf("resolve publication target: %w", err)
	}
	if relative == "." || filepath.IsAbs(relative) || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative != filename {
		return "", fmt.Errorf("publication target escaped authorized destination root")
	}
	return target, nil
}

func (p *LocalPayloadPublisher) revalidateRoot() error {
	canonical, err := canonicalPublicationRoot(p.root)
	if err != nil {
		return err
	}
	if canonical != p.root {
		return fmt.Errorf("authorized publication root changed after initialization")
	}
	return nil
}

func canonicalPublicationRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("publication root must not be empty")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve publication root: %w", err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve publication root symlinks: %w", err)
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", fmt.Errorf("stat publication root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("publication root must be a directory")
	}
	return filepath.Clean(canonical), nil
}

func validateLocalDestinationID(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("destination ID must not be empty")
	}
	if len(value) > 256 {
		return fmt.Errorf("destination ID exceeds 256 bytes")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("destination ID must not contain control characters")
		}
	}
	return nil
}
