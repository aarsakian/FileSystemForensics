package ccm

import (
	"bytes"
	"crypto/aes"
	"testing"
)

func TestOpenCCM_Roundtrip(t *testing.T) {
	key := []byte{
		0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47,
		0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f,
		0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57,
		0x58, 0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f,
	}
	nonce := []byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b}
	aad := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	plaintext := []byte("The quick brown fox jumps over the lazy dog")

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("aes.NewCipher: %v", err)
	}

	ciphertext := make([]byte, len(plaintext))
	stream := newCTR(block, nonce, 1)
	stream.XORKeyStream(ciphertext, plaintext)

	mac, err := cbcMac(block, nonce, aad, plaintext)
	if err != nil {
		t.Fatalf("cbcMac: %v", err)
	}

	tag := xor16(mac, ctrBlock(block, nonce, 0))
	ctTag := append(ciphertext, tag...)

	result, err := OpenCCM(key, nonce, aad, ctTag)
	if err != nil {
		t.Fatalf("OpenCCM returned error: %v", err)
	}

	if !bytes.Equal(result, plaintext) {
		t.Fatalf("decrypted plaintext mismatch\nexpected: %x\nactual:   %x", plaintext, result)
	}
}
