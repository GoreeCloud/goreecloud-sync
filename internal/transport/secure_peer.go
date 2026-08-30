package transport

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"net"
	"time"
)

const (
	securePeerALPN                = "goreecloud-sync/1"
	securePeerCertificateLifetime = 24 * time.Hour
	securePeerClockSkew            = 5 * time.Minute
)

var ErrSecurePeerAuthentication = errors.New("secure peer authentication failed")

// SecurePeerIdentity is the local device identity used only in memory while a
// TLS session is established. The caller remains responsible for loading the
// private key through the platform key-protection boundary; transport never
// persists private-key material.
type SecurePeerIdentity struct {
	DeviceID   string
	PrivateKey ed25519.PrivateKey
}

// TrustedPeerIdentity is an already-authorized remote device identity. The raw
// Ed25519 key is pinned directly; no public CA or network location grants trust.
type TrustedPeerIdentity struct {
	DeviceID  string
	PublicKey ed25519.PublicKey
}

// DialSecurePeer establishes TCP and then completes a mutually authenticated
// TLS 1.3 handshake pinned to the expected peer device/key identity.
func DialSecurePeer(ctx context.Context, address string, timeout time.Duration, local SecurePeerIdentity, expected TrustedPeerIdentity) (*PeerConn, error) {
	if ctx == nil || address == "" || timeout <= 0 {
		return nil, fmt.Errorf("secure peer context, address, and timeout are required")
	}
	config, err := securePeerTLSConfig(local, expected, true)
	if err != nil {
		return nil, err
	}

	handshakeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(handshakeCtx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("dial secure peer: %w", err)
	}
	return upgradeSecurePeer(handshakeCtx, conn, config, local.DeviceID, expected.DeviceID, true)
}

// AcceptSecurePeer takes ownership of an accepted stream and requires a pinned
// client identity before returning a PeerConn. Failure closes the stream.
func AcceptSecurePeer(ctx context.Context, conn net.Conn, timeout time.Duration, local SecurePeerIdentity, expected TrustedPeerIdentity) (*PeerConn, error) {
	if ctx == nil || conn == nil || timeout <= 0 {
		if conn != nil {
			_ = conn.Close()
		}
		return nil, fmt.Errorf("secure peer context, connection, and timeout are required")
	}
	config, err := securePeerTLSConfig(local, expected, false)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	handshakeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return upgradeSecurePeer(handshakeCtx, conn, config, local.DeviceID, expected.DeviceID, false)
}

func upgradeSecurePeer(ctx context.Context, conn net.Conn, config *tls.Config, localDeviceID, expectedDeviceID string, client bool) (*PeerConn, error) {
	var tlsConn *tls.Conn
	if client {
		tlsConn = tls.Client(conn, config)
	} else {
		tlsConn = tls.Server(conn, config)
	}
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("%w: %v", ErrSecurePeerAuthentication, err)
	}
	return &PeerConn{
		conn:                  tlsConn,
		localDeviceID:         localDeviceID,
		authenticatedDeviceID: expectedDeviceID,
	}, nil
}

func securePeerTLSConfig(local SecurePeerIdentity, expected TrustedPeerIdentity, client bool) (*tls.Config, error) {
	if !validDeviceID(local.DeviceID) || len(local.PrivateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid local secure peer identity")
	}
	if !validDeviceID(expected.DeviceID) || len(expected.PublicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid trusted peer identity")
	}
	certificate, err := newSecurePeerCertificate(local)
	if err != nil {
		return nil, err
	}
	expectedKey := append(ed25519.PublicKey(nil), expected.PublicKey...)
	config := &tls.Config{
		MinVersion:             tls.VersionTLS13,
		MaxVersion:             tls.VersionTLS13,
		Certificates:           []tls.Certificate{certificate},
		NextProtos:              []string{securePeerALPN},
		SessionTicketsDisabled: true,
		VerifyConnection:        verifySecurePeer(expected.DeviceID, expectedKey, client),
	}
	if client {
		// Public Web PKI hostname verification is intentionally replaced by the
		// exact pinned GoreeCloud device identity in VerifyConnection. Go still
		// performs the TLS 1.3 handshake and CertificateVerify proof of key possession.
		config.InsecureSkipVerify = true
	} else {
		// Require a client certificate, then apply exact device/key pinning in
		// VerifyConnection rather than treating an arbitrary CA as device trust.
		config.ClientAuth = tls.RequireAnyClientCert
	}
	return config, nil
}

func newSecurePeerCertificate(local SecurePeerIdentity) (tls.Certificate, error) {
	publicKey := local.PrivateKey.Public().(ed25519.PublicKey)
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate secure peer certificate serial: %w", err)
	}
	if serial.Sign() == 0 {
		serial.SetInt64(1)
	}
	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: securePeerCommonName(local.DeviceID)},
		NotBefore:    now.Add(-securePeerClockSkew),
		NotAfter:     now.Add(securePeerCertificateLifetime),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, local.PrivateKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create secure peer certificate: %w", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("parse secure peer certificate: %w", err)
	}
	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  local.PrivateKey,
		Leaf:        leaf,
	}, nil
}

func verifySecurePeer(expectedDeviceID string, expectedPublicKey ed25519.PublicKey, client bool) func(tls.ConnectionState) error {
	requiredUsage := x509.ExtKeyUsageClientAuth
	if client {
		requiredUsage = x509.ExtKeyUsageServerAuth
	}
	return func(state tls.ConnectionState) error {
		if state.Version != tls.VersionTLS13 {
			return fmt.Errorf("%w: TLS 1.3 was not negotiated", ErrSecurePeerAuthentication)
		}
		if state.NegotiatedProtocol != securePeerALPN {
			return fmt.Errorf("%w: GoreeCloud Sync ALPN was not negotiated", ErrSecurePeerAuthentication)
		}
		if len(state.PeerCertificates) != 1 {
			return fmt.Errorf("%w: expected one peer certificate", ErrSecurePeerAuthentication)
		}
		certificate := state.PeerCertificates[0]
		if certificate.Subject.CommonName != securePeerCommonName(expectedDeviceID) {
			return fmt.Errorf("%w: peer device identity mismatch", ErrSecurePeerAuthentication)
		}
		if certificate.IsCA || certificate.KeyUsage&x509.KeyUsageDigitalSignature == 0 || !hasExtKeyUsage(certificate.ExtKeyUsage, requiredUsage) {
			return fmt.Errorf("%w: invalid peer certificate usage", ErrSecurePeerAuthentication)
		}
		now := time.Now().UTC()
		if now.Before(certificate.NotBefore) || now.After(certificate.NotAfter) {
			return fmt.Errorf("%w: peer certificate is outside its validity interval", ErrSecurePeerAuthentication)
		}
		publicKey, ok := certificate.PublicKey.(ed25519.PublicKey)
		if !ok || !bytes.Equal(publicKey, expectedPublicKey) {
			return fmt.Errorf("%w: peer public key mismatch", ErrSecurePeerAuthentication)
		}
		if err := certificate.CheckSignature(certificate.SignatureAlgorithm, certificate.RawTBSCertificate, certificate.Signature); err != nil {
			return fmt.Errorf("%w: invalid self-signed peer certificate: %v", ErrSecurePeerAuthentication, err)
		}
		return nil
	}
}

func securePeerCommonName(deviceID string) string {
	return "goreecloud-sync-device:" + deviceID
}

func hasExtKeyUsage(usages []x509.ExtKeyUsage, required x509.ExtKeyUsage) bool {
	for _, usage := range usages {
		if usage == required {
			return true
		}
	}
	return false
}
