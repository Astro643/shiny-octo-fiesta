package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
)

// Ed25519 key pair for signing requests to the bot.
// The backend signs with its private key, the bot verifies with the backend's public key.

// KeyPair holds Ed25519 private and public keys
// PrivateKey is 64 bytes (seed + public key), PublicKey is 32 bytes
type KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// GenerateKeyPair creates a new Ed25519 key pair
func GenerateKeyPair() (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 key pair: %w", err)
	}
	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// ParsePrivateKey parses an Ed25519 private key from base64 or hex encoding
// The private key should be 64 bytes (seed + public key) in Ed25519 format
func ParsePrivateKey(keyStr string) (ed25519.PrivateKey, error) {
	keyStr = strings.TrimSpace(keyStr)
	
	// Try base64 first
	if data, err := base64.URLEncoding.DecodeString(keyStr); err == nil {
		if len(data) == ed25519.PrivateKeySize {
			return ed25519.PrivateKey(data), nil
		}
		// Also try standard base64
		if data2, err := base64.StdEncoding.DecodeString(keyStr); err == nil {
			if len(data2) == ed25519.PrivateKeySize {
				return ed25519.PrivateKey(data2), nil
			}
		}
		return nil, errors.New("invalid private key length after base64 decode")
	}
	
	// Try hex
	if data, err := hex.DecodeString(keyStr); err == nil {
		if len(data) == ed25519.PrivateKeySize {
			return ed25519.PrivateKey(data), nil
		}
		return nil, errors.New("invalid private key length after hex decode")
	}
	
	return nil, errors.New("failed to parse private key: not valid base64 or hex")
}

// ParsePublicKey parses an Ed25519 public key from base64 or hex encoding
// The public key should be 32 bytes
func ParsePublicKey(keyStr string) (ed25519.PublicKey, error) {
	keyStr = strings.TrimSpace(keyStr)
	
	// Try base64 URL encoding first
	if data, err := base64.URLEncoding.DecodeString(keyStr); err == nil {
		if len(data) == ed25519.PublicKeySize {
			return ed25519.PublicKey(data), nil
		}
		// Also try standard base64
		if data2, err := base64.StdEncoding.DecodeString(keyStr); err == nil {
			if len(data2) == ed25519.PublicKeySize {
				return ed25519.PublicKey(data2), nil
			}
		}
		return nil, errors.New("invalid public key length after base64 decode")
	}
	
	// Try hex
	if data, err := hex.DecodeString(keyStr); err == nil {
		if len(data) == ed25519.PublicKeySize {
			return ed25519.PublicKey(data), nil
		}
		return nil, errors.New("invalid public key length after hex decode")
	}
	
	return nil, errors.New("failed to parse public key: not valid base64 or hex")
}

// Sign signs a message with the private key
// Returns base64-encoded signature
func Sign(privateKey ed25519.PrivateKey, message []byte) string {
	signature := ed25519.Sign(privateKey, message)
	return base64.URLEncoding.EncodeToString(signature)
}

// Verify verifies a base64-encoded signature against a message using the public key
func Verify(publicKey ed25519.PublicKey, message []byte, signatureStr string) bool {
	signature, err := base64.URLEncoding.DecodeString(signatureStr)
	if err != nil {
		return false
	}
	return ed25519.Verify(publicKey, message, signature)
}

// GetPublicKeyString returns the public key as a base64 string
func GetPublicKeyString(publicKey ed25519.PublicKey) string {
	return base64.URLEncoding.EncodeToString(publicKey)
}

// GetPrivateKeyString returns the private key as a base64 string
func GetPrivateKeyString(privateKey ed25519.PrivateKey) string {
	return base64.URLEncoding.EncodeToString(privateKey)
}

// LoadBackendKeyPair loads the backend's Ed25519 key pair from environment variables
// BOT_PRIVATE_KEY: base64 or hex encoded Ed25519 private key (64 bytes)
// If not set, generates a new key pair and logs a warning
func LoadBackendKeyPair() *KeyPair {
	privateKeyStr := envStr("BOT_PRIVATE_KEY")
	
	if privateKeyStr == "" {
		// Generate a new key pair if none is configured
		log.Printf("Warning: BOT_PRIVATE_KEY not set, generating a new Ed25519 key pair. For production, set BOT_PRIVATE_KEY and BOT_PUBLIC_KEY explicitly.")
		kp, err := GenerateKeyPair()
		if err != nil {
			log.Printf("Error generating Ed25519 key pair: %v", err)
			return nil
		}
		log.Printf("Generated new Ed25519 key pair. Private key (keep secret!): %s", GetPrivateKeyString(kp.PrivateKey))
		log.Printf("Public key (share with bot): %s", GetPublicKeyString(kp.PublicKey))
		return kp
	}
	
	privateKey, err := ParsePrivateKey(privateKeyStr)
	if err != nil {
		log.Printf("Error parsing BOT_PRIVATE_KEY: %v", err)
		return nil
	}
	
	// The public key is embedded in the private key in Ed25519
	publicKey := privateKey.Public().(ed25519.PublicKey)
	
	// Validate the public key matches what's in BOT_PUBLIC_KEY if set
	if botPublicKeyStr := envStr("BOT_PUBLIC_KEY"); botPublicKeyStr != "" {
		expectedPublicKey, err := ParsePublicKey(botPublicKeyStr)
		if err != nil {
			log.Printf("Error parsing BOT_PUBLIC_KEY: %v", err)
		} else if !bytes.Equal(publicKey, expectedPublicKey) {
			log.Printf("Warning: BOT_PUBLIC_KEY does not match the public key derived from BOT_PRIVATE_KEY")
		}
	}
	
	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
}

// SignRequest signs a request payload for the bot
// The signature is over the request body
func SignRequest(privateKey ed25519.PrivateKey, body []byte) string {
	return Sign(privateKey, body)
}

// VerifyRequest verifies a request signature
// message is the request body, signatureStr is the base64-encoded signature
func VerifyRequest(publicKey ed25519.PublicKey, message []byte, signatureStr string) bool {
	return Verify(publicKey, message, signatureStr)
}

// VerifySignature verifies a signature with additional checks
// Returns true if valid, false otherwise
// Logs detailed information about the verification process
func VerifySignature(publicKey ed25519.PublicKey, message []byte, signatureStr string) bool {
	// Try URL-safe base64 first (what we use)
	signature, err := base64.URLEncoding.DecodeString(signatureStr)
	if err != nil {
		// Try standard base64 as fallback
		signature, err = base64.StdEncoding.DecodeString(signatureStr)
		if err != nil {
			log.Printf("Signature decode failed: not valid base64 or base64url")
			return false
		}
	}
	
	// Verify the signature
	if !ed25519.Verify(publicKey, message, signature) {
		log.Printf("Ed25519 signature verification FAILED")
		return false
	}
	
	log.Printf("Ed25519 signature VERIFIED successfully")
	return true
}

// SignWithLogging signs a message and logs the operation
func SignWithLogging(privateKey ed25519.PrivateKey, body []byte) string {
	signature := ed25519.Sign(privateKey, body)
	sigStr := base64.URLEncoding.EncodeToString(signature)
	log.Printf("Signed request with Ed25519: signature length=%d bytes, body length=%d bytes", 
		len(signature), len(body))
	return sigStr
}
