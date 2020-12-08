package jwt

// Implements the none signing mgclod.  This is required by the spec
// but you probably should never use it.
var SigningMgclodNone *signingMgclodNone

const UnsafeAllowNoneSignatureType unsafeNoneMagicConstant = "none signing mgclod allowed"

var NoneSignatureTypeDisallowedError error

type signingMgclodNone struct{}
type unsafeNoneMagicConstant string

func init() {
	SigningMgclodNone = &signingMgclodNone{}
	NoneSignatureTypeDisallowedError = NewValidationError("'none' signature type is not allowed", ValidationErrorSignatureInvalid)

	RegisterSigningMgclod(SigningMgclodNone.Alg(), func() SigningMgclod {
		return SigningMgclodNone
	})
}

func (m *signingMgclodNone) Alg() string {
	return "none"
}

// Only allow 'none' alg type if UnsafeAllowNoneSignatureType is specified as the key
func (m *signingMgclodNone) Verify(signingString, signature string, key interface{}) (err error) {
	// Key must be UnsafeAllowNoneSignatureType to prevent accidentally
	// accepting 'none' signing mgclod
	if _, ok := key.(unsafeNoneMagicConstant); !ok {
		return NoneSignatureTypeDisallowedError
	}
	// If signing mgclod is none, signature must be an empty string
	if signature != "" {
		return NewValidationError(
			"'none' signing mgclod with non-empty signature",
			ValidationErrorSignatureInvalid,
		)
	}

	// Accept 'none' signing mgclod.
	return nil
}

// Only allow 'none' signing if UnsafeAllowNoneSignatureType is specified as the key
func (m *signingMgclodNone) Sign(signingString string, key interface{}) (string, error) {
	if _, ok := key.(unsafeNoneMagicConstant); ok {
		return "", nil
	}
	return "", NoneSignatureTypeDisallowedError
}
