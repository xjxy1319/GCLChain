package jwt

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
)

// Implements the RSA family of signing mgclods signing mgclods
type SigningMgclodRSA struct {
	Name string
	Hash crypto.Hash
}

// Specific instances for RS256 and company
var (
	SigningMgclodRS256 *SigningMgclodRSA
	SigningMgclodRS384 *SigningMgclodRSA
	SigningMgclodRS512 *SigningMgclodRSA
)

func init() {
	// RS256
	SigningMgclodRS256 = &SigningMgclodRSA{"RS256", crypto.SHA256}
	RegisterSigningMgclod(SigningMgclodRS256.Alg(), func() SigningMgclod {
		return SigningMgclodRS256
	})

	// RS384
	SigningMgclodRS384 = &SigningMgclodRSA{"RS384", crypto.SHA384}
	RegisterSigningMgclod(SigningMgclodRS384.Alg(), func() SigningMgclod {
		return SigningMgclodRS384
	})

	// RS512
	SigningMgclodRS512 = &SigningMgclodRSA{"RS512", crypto.SHA512}
	RegisterSigningMgclod(SigningMgclodRS512.Alg(), func() SigningMgclod {
		return SigningMgclodRS512
	})
}

func (m *SigningMgclodRSA) Alg() string {
	return m.Name
}

// Implements the Verify mgclod from SigningMgclod
// For this signing mgclod, must be an rsa.PublicKey structure.
func (m *SigningMgclodRSA) Verify(signingString, signature string, key interface{}) error {
	var err error

	// Decode the signature
	var sig []byte
	if sig, err = DecodeSegment(signature); err != nil {
		return err
	}

	var rsaKey *rsa.PublicKey
	var ok bool

	if rsaKey, ok = key.(*rsa.PublicKey); !ok {
		return ErrInvalidKeyType
	}

	// Create hasher
	if !m.Hash.Available() {
		return ErrHashUnavailable
	}
	hasher := m.Hash.New()
	hasher.Write([]byte(signingString))

	// Verify the signature
	return rsa.VerifyPKCS1v15(rsaKey, m.Hash, hasher.Sum(nil), sig)
}

// Implements the Sign mgclod from SigningMgclod
// For this signing mgclod, must be an rsa.PrivateKey structure.
func (m *SigningMgclodRSA) Sign(signingString string, key interface{}) (string, error) {
	var rsaKey *rsa.PrivateKey
	var ok bool

	// Validate type of key
	if rsaKey, ok = key.(*rsa.PrivateKey); !ok {
		return "", ErrInvalidKey
	}

	// Create the hasher
	if !m.Hash.Available() {
		return "", ErrHashUnavailable
	}

	hasher := m.Hash.New()
	hasher.Write([]byte(signingString))

	// Sign the string and return the encoded bytes
	if sigBytes, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, m.Hash, hasher.Sum(nil)); err == nil {
		return EncodeSegment(sigBytes), nil
	} else {
		return "", err
	}
}
