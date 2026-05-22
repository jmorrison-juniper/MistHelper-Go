// Package ssh implements the SSH server for MistHelper-Go.
// It provides host key management and a ForceCommand SSH listener that launches the interactive menu.
package ssh

import (
	"crypto/rand"    // for rand.Reader -- cryptographically secure random source for RSA generation
	"crypto/rsa"     // for rsa.GenerateKey -- RSA 2048-bit private key generation
	"crypto/x509"    // for x509.MarshalPKCS1PrivateKey -- PKCS#1 DER encoding of the RSA key
	"encoding/pem"   // for pem.Block and pem.EncodeToMemory -- PEM wrapper for the DER bytes
	"fmt"            // for fmt.Errorf with %w -- wraps errors with caller context
	"log/slog"       // for slog.Info and slog.Debug -- structured logging before and after each action
	"os"             // for os.Stat, os.ReadFile, os.WriteFile -- host key file I/O
	"path/filepath"  // for filepath.Join -- cross-platform path construction

	gossh "golang.org/x/crypto/ssh" // aliased to gossh to avoid conflict with this package name
)

// keyFileName is the base name of the RSA host key file stored in dataDir.
const keyFileName = "ssh_host_rsa_key"

// rsaKeyBits is the size in bits for the generated RSA key (NIST minimum for servers is 2048).
const rsaKeyBits = 2048

// LoadOrCreateHostKey loads the RSA host key from dataDir/ssh_host_rsa_key if it exists,
// or generates a new 2048-bit RSA key, saves it, and returns the signer.
// dataDir is the directory where the key file is written (e.g. "data/").
func LoadOrCreateHostKey(dataDir string) (gossh.Signer, error) {
	keyPath := filepath.Join(dataDir, keyFileName)   // Build the full path to the key file
	slog.Info("loading SSH host key", "path", keyPath) // Log before checking the file system

	exists, err := keyFileExists(keyPath) // Check whether the key already exists on disk
	if err != nil {                       // Stat failure means we cannot determine file state
		return nil, fmt.Errorf("check host key %s: %w", keyPath, err) // Wrap with path context
	}

	if !exists { // Key is missing -- generate a fresh one so the server can start
		if genErr := generateAndSaveKey(keyPath); genErr != nil { // Create and persist the new key
			return nil, fmt.Errorf("generate host key: %w", genErr) // Wrap generation error
		}
	}

	return loadKeyFromFile(keyPath) // Load (and parse) the key whether brand-new or pre-existing
}

// keyFileExists returns true when keyPath refers to a regular file that exists.
func keyFileExists(keyPath string) (bool, error) {
	_, err := os.Stat(keyPath)    // Stat the file to check for existence
	if os.IsNotExist(err) {       // os.IsNotExist distinguishes "missing" from other I/O errors
		return false, nil         // File not found is a normal first-run condition, not an error
	}
	if err != nil {               // Any other Stat error (permission denied, I/O failure) is fatal
		return false, err         // Return the raw error so the caller can wrap it with context
	}
	return true, nil // File exists and is accessible
}

// generateAndSaveKey creates a 2048-bit RSA key, PEM-encodes it, and writes it to keyPath with mode 0600.
func generateAndSaveKey(keyPath string) error {
	slog.Info("generating new RSA host key", "path", keyPath, "bits", rsaKeyBits) // Log before generation
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)                   // Generate the RSA key pair
	if err != nil {                                                                // Generation can fail if the system PRNG is broken
		return fmt.Errorf("generate RSA key: %w", err)                            // Wrap so callers know what step failed
	}

	pemData := encodePEM(privateKey) // PEM-encode the private key so OpenSSH clients can verify the signature

	if writeErr := os.WriteFile(keyPath, pemData, 0600); writeErr != nil { // Write with mode 0600 -- owner-read-only
		return fmt.Errorf("write host key to %s: %w", keyPath, writeErr)  // Wrap with path for diagnosis
	}
	slog.Debug("generated and saved RSA host key", "path", keyPath) // Log success with the key path
	return nil                                                       // Caller can now load the key from disk
}

// encodePEM converts an RSA private key to PKCS#1 PEM bytes.
func encodePEM(key *rsa.PrivateKey) []byte {
	derBytes := x509.MarshalPKCS1PrivateKey(key) // Encode the key to DER format (binary ASN.1)
	block := &pem.Block{                         // Wrap DER bytes in a PEM block with the standard RSA header
		Type:  "RSA PRIVATE KEY", // Standard OpenSSH/OpenSSL PEM type for PKCS#1 RSA keys
		Bytes: derBytes,          // DER-encoded key material
	}
	return pem.EncodeToMemory(block) // Convert the PEM block to bytes ready for WriteFile
}

// loadKeyFromFile reads keyPath, parses the PEM-encoded RSA key, and returns an ssh.Signer.
func loadKeyFromFile(keyPath string) (gossh.Signer, error) {
	slog.Info("reading SSH host key from disk", "path", keyPath) // Log before the file read
	pemData, err := os.ReadFile(keyPath)                         // Read the entire key file into memory
	if err != nil {                                              // File may have been deleted between stat and read
		return nil, fmt.Errorf("read host key %s: %w", keyPath, err) // Wrap with path for diagnosis
	}

	signer, err := gossh.ParsePrivateKey(pemData) // Parse PEM bytes into an ssh.Signer
	if err != nil {                               // Key file may be corrupt or in an unexpected format
		return nil, fmt.Errorf("parse host key %s: %w", keyPath, err) // Wrap with path so caller knows which file failed
	}
	slog.Debug("loaded SSH host key", "path", keyPath, "type", signer.PublicKey().Type()) // Log key type after successful load
	return signer, nil                                                                     // Return the ready-to-use signer
}
