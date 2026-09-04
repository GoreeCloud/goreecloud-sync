package app

import (
	"fmt"
	"io"

	"github.com/GoreeCloud/goreecloud-sync/internal/transfer"
	"github.com/GoreeCloud/goreecloud-sync/internal/transport"
)

// SendTransferPayload revalidates the exact durable account/device/key trust
// before starting one payload transfer and before each source read performed by
// the transport primitive. A final revalidation prevents the caller from
// accepting local success if trust is no longer current when the operation
// returns.
//
// These are explicit checkpoints, not asynchronous revocation. A trust change
// cannot retroactively stop bytes already written after a successful checkpoint.
func (f SecurePeerFactory) SendTransferPayload(peer *transport.PeerConn, offer transfer.PayloadOffer, source io.Reader) (transfer.PayloadReceipt, error) {
	if source == nil {
		return transfer.PayloadReceipt{}, fmt.Errorf("payload source must not be nil")
	}
	if err := f.RevalidatePeer(peer); err != nil {
		return transfer.PayloadReceipt{}, err
	}
	receipt, err := peer.SendTransferPayload(offer, currentTrustReader{factory: f, peer: peer, source: source})
	if err != nil {
		return transfer.PayloadReceipt{}, err
	}
	if err := f.RevalidatePeer(peer); err != nil {
		return transfer.PayloadReceipt{}, err
	}
	return receipt, nil
}

// ReceiveTransferPayload revalidates durable trust before entering the receive
// operation, again immediately before the application authorizes the offered
// transfer, and before each verified chunk is written to the caller's staging
// destination. The caller-supplied authorizer remains mandatory and independent
// of transport/device trust.
func (f SecurePeerFactory) ReceiveTransferPayload(peer *transport.PeerConn, authorize transport.PayloadAuthorizer, destination io.Writer) (transfer.PayloadOffer, transfer.PayloadReceipt, error) {
	if authorize == nil {
		return transfer.PayloadOffer{}, transfer.PayloadReceipt{}, fmt.Errorf("payload authorizer must not be nil")
	}
	if destination == nil {
		return transfer.PayloadOffer{}, transfer.PayloadReceipt{}, fmt.Errorf("payload staging destination must not be nil")
	}
	if err := f.RevalidatePeer(peer); err != nil {
		return transfer.PayloadOffer{}, transfer.PayloadReceipt{}, err
	}
	guardedAuthorize := func(offer transfer.PayloadOffer) error {
		if err := f.RevalidatePeer(peer); err != nil {
			return err
		}
		return authorize(offer)
	}
	offer, receipt, err := peer.ReceiveTransferPayload(guardedAuthorize, currentTrustWriter{factory: f, peer: peer, destination: destination})
	if err != nil {
		return offer, transfer.PayloadReceipt{}, err
	}
	if err := f.RevalidatePeer(peer); err != nil {
		return offer, transfer.PayloadReceipt{}, err
	}
	return offer, receipt, nil
}

type currentTrustReader struct {
	factory SecurePeerFactory
	peer    *transport.PeerConn
	source  io.Reader
}

func (r currentTrustReader) Read(p []byte) (int, error) {
	if err := r.factory.RevalidatePeer(r.peer); err != nil {
		return 0, err
	}
	return r.source.Read(p)
}

type currentTrustWriter struct {
	factory     SecurePeerFactory
	peer        *transport.PeerConn
	destination io.Writer
}

func (w currentTrustWriter) Write(p []byte) (int, error) {
	if err := w.factory.RevalidatePeer(w.peer); err != nil {
		return 0, err
	}
	return w.destination.Write(p)
}
