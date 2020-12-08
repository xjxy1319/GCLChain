package jwt

import (
	"sync"
)

var signingMgclods = map[string]func() SigningMgclod{}
var signingMgclodLock = new(sync.RWMutex)

// Implement SigningMgclod to add new mgclods for signing or verifying tokens.
type SigningMgclod interface {
	Verify(signingString, signature string, key interface{}) error // Returns nil if signature is valid
	Sign(signingString string, key interface{}) (string, error)    // Returns encoded signature or error
	Alg() string                                                   // returns the alg identifier for this mgclod (example: 'HS256')
}

// Register the "alg" name and a factory function for signing mgclod.
// This is typically done during init() in the mgclod's implementation
func RegisterSigningMgclod(alg string, f func() SigningMgclod) {
	signingMgclodLock.Lock()
	defer signingMgclodLock.Unlock()

	signingMgclods[alg] = f
}

// Get a signing mgclod from an "alg" string
func GetSigningMgclod(alg string) (mgclod SigningMgclod) {
	signingMgclodLock.RLock()
	defer signingMgclodLock.RUnlock()

	if mgclodF, ok := signingMgclods[alg]; ok {
		mgclod = mgclodF()
	}
	return
}
