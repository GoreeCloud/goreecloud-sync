package transport

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/GoreeCloud/goreecloud-sync/internal/transfer"
)

var (
	ErrPayloadTransferPeerNotTrusted      = errors.New("payload transfer requires an authenticated trusted peer")
	ErrPayloadTransferRejected            = errors.New("payload transfer was rejected by the receiver")
	ErrPayloadTransferAuthorizationDenied = errors.New("payload transfer authorization was denied")
)

// PayloadAuthorizer is the application-authorization seam for an incoming
// transfer. Secure transport trust is necessary but does not by itself grant
// authority to receive a file or text payload.
type PayloadAuthorizer func(transfer.PayloadOffer) error

// SendTransferPayload sends one already-manifested file or text payload through
// a secure peer that has been composed with durable trusted-device state. The
// receiver must explicitly accept the offer before any payload bytes are sent.
// The returned receipt is accepted only when it matches the exact transfer ID,
// byte size, and whole-payload digest from the offer.
//
// This is a Development-stage one-to-one stream primitive. It does not persist
// resumable progress, select peers, discover addresses, authorize destination
// paths, or define production session/reconnect policy.
func (p *PeerConn) SendTransferPayload(offer transfer.PayloadOffer, source io.Reader) (transfer.PayloadReceipt, error) {
	if err := p.requireTrustedPayloadPeer(); err != nil {
		return transfer.PayloadReceipt{}, err
	}
	if err := offer.Validate(); err != nil {
		return transfer.PayloadReceipt{}, fmt.Errorf("validate payload offer: %w", err)
	}
	if source == nil {
		return transfer.PayloadReceipt{}, fmt.Errorf("payload source must not be nil")
	}

	if err := WriteJSONFrame(p.conn, offer); err != nil {
		return transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("send payload offer: %w", err))
	}
	var decision transfer.PayloadDecision
	if err := ReadJSONFrame(p.conn, &decision); err != nil {
		return transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("receive payload decision: %w", err))
	}
	if err := decision.ValidateFor(offer); err != nil {
		return transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("validate payload decision: %w", err))
	}
	if !decision.Accepted {
		return transfer.PayloadReceipt{}, ErrPayloadTransferRejected
	}

	wholeHash := sha256.New()
	buffer := make([]byte, offer.Manifest.ChunkSize)
	for _, expected := range offer.Manifest.Chunks {
		data := buffer[:expected.Size]
		if _, err := io.ReadFull(source, data); err != nil {
			return transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("read source chunk %d: %w", expected.Index, err))
		}
		if err := transfer.VerifyChunk(expected, data); err != nil {
			return transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("verify source chunk %d: %w", expected.Index, err))
		}
		if _, err := wholeHash.Write(data); err != nil {
			return transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("hash source chunk %d: %w", expected.Index, err))
		}
		if err := WriteBinaryFrame(p.conn, data); err != nil {
			return transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("send payload chunk %d: %w", expected.Index, err))
		}
	}

	if n, err := io.CopyN(io.Discard, source, 1); n > 0 {
		return transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("source contains undeclared trailing data"))
	} else if err != nil && err != io.EOF {
		return transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("check source payload boundary: %w", err))
	}
	if actual := hex.EncodeToString(wholeHash.Sum(nil)); actual != offer.Manifest.Hash {
		return transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("whole source payload integrity verification failed"))
	}

	completion := transfer.CompletedPayload(offer)
	if err := WriteJSONFrame(p.conn, completion); err != nil {
		return transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("send payload completion: %w", err))
	}

	var receipt transfer.PayloadReceipt
	if err := ReadJSONFrame(p.conn, &receipt); err != nil {
		return transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("receive payload receipt: %w", err))
	}
	if err := receipt.ValidateFor(offer); err != nil {
		return transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("validate payload receipt: %w", err))
	}
	return receipt, nil
}

// ReceiveTransferPayload receives one file or text payload into staging after an
// explicit application authorization callback approves the offer. destination
// must be treated as staging by callers: partial bytes may have been written if
// a later integrity or stream failure occurs, and the caller must not publish or
// commit that staged content until this function returns a verified receipt.
func (p *PeerConn) ReceiveTransferPayload(authorize PayloadAuthorizer, destination io.Writer) (transfer.PayloadOffer, transfer.PayloadReceipt, error) {
	if err := p.requireTrustedPayloadPeer(); err != nil {
		return transfer.PayloadOffer{}, transfer.PayloadReceipt{}, err
	}
	if authorize == nil {
		return transfer.PayloadOffer{}, transfer.PayloadReceipt{}, fmt.Errorf("payload authorizer must not be nil")
	}
	if destination == nil {
		return transfer.PayloadOffer{}, transfer.PayloadReceipt{}, fmt.Errorf("payload staging destination must not be nil")
	}

	var offer transfer.PayloadOffer
	if err := ReadJSONFrame(p.conn, &offer); err != nil {
		return transfer.PayloadOffer{}, transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("receive payload offer: %w", err))
	}
	if err := offer.Validate(); err != nil {
		return transfer.PayloadOffer{}, transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("validate payload offer: %w", err))
	}

	if err := authorize(offer); err != nil {
		// A trust-aware authorizer may deliberately close the peer while rejecting
		// stale authorization. Preserve that failure rather than attempting to send
		// a policy rejection over a connection that has already failed closed.
		if p.IsClosed() {
			return offer, transfer.PayloadReceipt{}, err
		}
		decision := transfer.PayloadDecision{
			Version:    transfer.PayloadProtocolVersion,
			TransferID: offer.TransferID,
			Accepted:   false,
		}
		if writeErr := WriteJSONFrame(p.conn, decision); writeErr != nil {
			return offer, transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("send payload rejection: %w", writeErr))
		}
		return offer, transfer.PayloadReceipt{}, fmt.Errorf("%w: %v", ErrPayloadTransferAuthorizationDenied, err)
	}

	decision := transfer.PayloadDecision{
		Version:    transfer.PayloadProtocolVersion,
		TransferID: offer.TransferID,
		Accepted:   true,
	}
	if err := WriteJSONFrame(p.conn, decision); err != nil {
		return offer, transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("send payload acceptance: %w", err))
	}

	wholeHash := sha256.New()
	for _, expected := range offer.Manifest.Chunks {
		data, err := ReadBinaryFrame(p.conn, expected.Size)
		if err != nil {
			return offer, transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("receive payload chunk %d: %w", expected.Index, err))
		}
		if err := transfer.VerifyChunk(expected, data); err != nil {
			return offer, transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("verify received chunk %d: %w", expected.Index, err))
		}
		if _, err := wholeHash.Write(data); err != nil {
			return offer, transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("hash received chunk %d: %w", expected.Index, err))
		}
		if err := writePayloadBytes(destination, data); err != nil {
			return offer, transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("stage received chunk %d: %w", expected.Index, err))
		}
	}

	var completion transfer.PayloadCompletion
	if err := ReadJSONFrame(p.conn, &completion); err != nil {
		return offer, transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("receive payload completion: %w", err))
	}
	if err := completion.ValidateFor(offer); err != nil {
		return offer, transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("validate payload completion: %w", err))
	}
	if actual := hex.EncodeToString(wholeHash.Sum(nil)); actual != offer.Manifest.Hash {
		return offer, transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("whole received payload integrity verification failed"))
	}

	receipt := transfer.VerifiedPayloadReceipt(offer)
	if err := WriteJSONFrame(p.conn, receipt); err != nil {
		return offer, transfer.PayloadReceipt{}, p.failPayloadTransfer(fmt.Errorf("send payload receipt: %w", err))
	}
	return offer, receipt, nil
}

func (p *PeerConn) requireTrustedPayloadPeer() error {
	if p == nil || p.conn == nil || p.IsClosed() || p.AuthenticatedDeviceID() == "" || p.AuthenticatedKeyFingerprint() == "" {
		if p != nil {
			_ = p.Close()
		}
		return ErrPayloadTransferPeerNotTrusted
	}
	return nil
}

func (p *PeerConn) failPayloadTransfer(err error) error {
	if p != nil {
		_ = p.Close()
	}
	return err
}

func writePayloadBytes(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if n > 0 {
			data = data[n:]
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}
