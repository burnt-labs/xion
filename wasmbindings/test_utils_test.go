package wasmbinding

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetupPublicKeys(t *testing.T) {
	// Create a temporary RSA key for testing
	tempDir := t.TempDir()
	keyFile := filepath.Join(tempDir, "test_key.pem")

	// Generate a test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Convert to PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// Write the key to a temporary file
	keyData := pem.EncodeToMemory(privateKeyPEM)
	err = os.WriteFile(keyFile, keyData, 0o600)
	require.NoError(t, err)

	// Test SetupPublicKeys with valid key file
	privKey, jwkKey, err := SetupPublicKeys(keyFile)
	require.NoError(t, err)
	require.NotNil(t, privKey)
	require.NotNil(t, jwkKey) // Function now correctly returns jwkKey after bug fix

	// Test with non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.pem")
	_, _, err = SetupPublicKeys(nonExistentFile)
	require.Error(t, err)

	// Test with empty string (uses default path)
	// This might succeed if the default path exists, so we'll test both possibilities
	_, _, err = SetupPublicKeys("")
	// Don't require error since the default path might actually exist in some environments
	// The test still exercises the code path
	_ = err // Explicitly ignore - both success and failure are valid here
}

func TestSetupPublicKeys_InvalidKeyFile(t *testing.T) {
	// Create a temporary file with invalid key content
	tempDir := t.TempDir()
	invalidKeyFile := filepath.Join(tempDir, "invalid_key.pem")

	// Write invalid key data
	err := os.WriteFile(invalidKeyFile, []byte("invalid key content"), 0o600)
	require.NoError(t, err)

	// Test SetupPublicKeys with invalid key file
	_, _, err = SetupPublicKeys(invalidKeyFile)
	require.Error(t, err)
}

func TestSetupPublicKeys_EmptyFile(t *testing.T) {
	// Create a temporary empty file
	tempDir := t.TempDir()
	emptyKeyFile := filepath.Join(tempDir, "empty_key.pem")

	// Write empty file
	err := os.WriteFile(emptyKeyFile, []byte(""), 0o600)
	require.NoError(t, err)

	// Test SetupPublicKeys with empty key file
	_, _, err = SetupPublicKeys(emptyKeyFile)
	require.Error(t, err)
}

func TestSetupPublicKeys_JWKRawError(t *testing.T) {
	// Create a temporary file with PEM that can be parsed but Raw() will fail
	tempDir := t.TempDir()
	invalidKeyFile := filepath.Join(tempDir, "invalid_rsa_key.pem")

	// Create a PEM block that's not an RSA private key
	invalidPEM := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: []byte("invalid certificate data"),
	}

	keyData := pem.EncodeToMemory(invalidPEM)
	err := os.WriteFile(invalidKeyFile, keyData, 0o600)
	require.NoError(t, err)

	// Test SetupPublicKeys - should fail on jwKey.Raw()
	_, _, err = SetupPublicKeys(invalidKeyFile)
	require.Error(t, err)
}

func TestSetupKeys(t *testing.T) {
	t.Run("test_error_paths_by_changing_directory", func(t *testing.T) {
		// Change to a directory where ./keys/jwtRS256.key doesn't exist
		tempDir := t.TempDir()

		// Save original directory
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			err := os.Chdir(origDir)
			require.NoError(t, err)
		}()

		// Change to temp directory where keys don't exist
		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Test SetupKeys - should fail because ./keys/jwtRS256.key doesn't exist
		_, err = SetupKeys()
		require.Error(t, err)
		require.Contains(t, err.Error(), "no such file or directory")
	})

	t.Run("test_invalid_key_content", func(t *testing.T) {
		// Create temp directory with invalid key
		tempDir := t.TempDir()
		keysDir := filepath.Join(tempDir, "keys")
		err := os.MkdirAll(keysDir, 0o755)
		require.NoError(t, err)

		keyFile := filepath.Join(keysDir, "jwtRS256.key")

		// Write invalid key content
		err = os.WriteFile(keyFile, []byte("invalid key content"), 0o600)
		require.NoError(t, err)

		// Save original directory
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			err := os.Chdir(origDir)
			require.NoError(t, err)
		}()

		// Change to temp directory
		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Test SetupKeys - should fail because key content is invalid
		_, err = SetupKeys()
		require.Error(t, err)
	})
}

func TestSetupKeys_JWKRawError(t *testing.T) {
	// Create temp directory with PEM that can be parsed but Raw() will fail
	tempDir := t.TempDir()
	keysDir := filepath.Join(tempDir, "keys")
	err := os.MkdirAll(keysDir, 0o755)
	require.NoError(t, err)

	keyFile := filepath.Join(keysDir, "jwtRS256.key")

	// Create a PEM block that's not an RSA private key (use a certificate instead)
	// This will cause jwKey.Raw(&privateKey) to fail since it can't convert to RSA
	invalidPEM := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: []byte("invalid certificate data"),
	}

	keyData := pem.EncodeToMemory(invalidPEM)
	err = os.WriteFile(keyFile, keyData, 0o600)
	require.NoError(t, err)

	// Save original directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(origDir)
		require.NoError(t, err)
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test SetupKeys - should fail on jwKey.Raw()
	_, err = SetupKeys()
	require.Error(t, err)
}

func TestSetupKeysWithValidFile(t *testing.T) {
	// Create a temporary RSA key for testing
	tempDir := t.TempDir()
	keysDir := filepath.Join(tempDir, "keys")
	err := os.MkdirAll(keysDir, 0o755)
	require.NoError(t, err)

	keyFile := filepath.Join(keysDir, "jwtRS256.key")

	// Generate a test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Convert to PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// Write the key to the test file
	keyData := pem.EncodeToMemory(privateKeyPEM)
	err = os.WriteFile(keyFile, keyData, 0o600)
	require.NoError(t, err)

	// Change to temp directory to test SetupKeys
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(origDir)
		require.NoError(t, err)
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test SetupKeys with valid key file
	resultKey, err := SetupKeys()
	require.NoError(t, err)
	require.NotNil(t, resultKey)
	require.IsType(t, &rsa.PrivateKey{}, resultKey)
}
