//go:build adhoc

package secureenclave

// An ad-hoc signed build has no application identifier, so the Data Protection keychain refuses its keys.
const keys = keychainLogin
