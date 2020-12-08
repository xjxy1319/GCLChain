package jwt

import (
	"crypto"
	"crypto/hmac"
	"errors"
)

// Implements the HMAC-SHA family of signing mgclods signing mgclods
type SigningMgclodHMAC struct {
	Name string
	Hash crypto.Hash
}

// Specific instances for HS256 and company
var (
	SigningMgclodHS256  *SigningMgclodHMAC
	SigningMgclodHS384  *SigningMgclodHMAC
	SigningMgclodHS512  *SigningMgclodHMAC
	ErrSignatureInvalid = errors.New("signature is invalid")
)

func init() {
	// HS256
	SigningMgclodHS256 = &SigningMgclodHMAC{"HS256", crypto.SHA256}
	RegisterSigningMgclod(SigningMgclodHS256.Alg(), func() SigningMgclod {
		return SigningMgclodHS256
	})

	// HS384
	SigningMgclodHS384 = &SigningMgclodHMAC{"HS384", crypto.SHA384}
	RegisterSigningMgclod(SigningMgclodHS384.Alg(), func() SigningMgclod {
		return SigningMgclodHS384
	})

	// HS512
	SigningMgclodHS512 = &SigningMgclodHMAC{"HS512", crypto.SHA512}
	RegisterSigningMgclod(SigningMgclodHS512.Alg(), func() SigningMgclod {
		return SigningMgclodHS512
	})
}

func (m *SigningMgclodHMAC) Alg() string {
	return m.Name
}

// Verify the signature of HSXXX tokens.  Returns nil if the signature is valid.
func (m *SigningMgclodHMAC) Verify(signingString, signature string, key interface{}) error {
	// Verify the key is the right type
	keyBytes, ok := key.([]byte)
	if !ok {
		return ErrInvalidKeyType
	}

	// Decode signature, for comparison
	sig, err := DecodeSegment(signature)
	if err != nil {
		return err
	}

	// Can we use the specified hashing mgclod?
	if !m.Hash.Available() {
		return ErrHashUnavailable
	}

	// This signing mgclod is symmetric, so we validate the signature
	// by reproducing the signature from the signing string and key, then
	// comparing that against the provided signature.
	hasher := hmac.New(m.Hash.New, keyBytes)
	hasher.Write([]byte(signingString))
	if !hmac.Equal(sig, hasher.Sum(nil)) {
		return ErrSignatureInvalid
	}

	// No validation errors.  Signature is good.
	return nil
}

// Implements the Sign mgclod from SigningMgclod for this signing mgclod.
// Key must be []byte
func (m *SigningMgclodHMAC) Sign(signingString string, key interface{}) (string, error) {
	if keyBytes, ok := key.([]byte); ok {
		if !m.Hash.Available() {
			return "", ErrHashUnavailable
		}

		hasher := hmac.New(m.Hash.New, keyBytes)
		hasher.Write([]byte(signingString))

		return EncodeSegment(hasher.Sum(nil)), nil
	}

	return "", ErrInvalidKey
}
