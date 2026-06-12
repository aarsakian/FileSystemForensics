package ccm

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"testing"
)

func TestOpenXTS128_RoundtripFullBlocks(t *testing.T) {
	key := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
	}
	tweak := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	plaintext := []byte("Example plaintext for AES-XTS full block test.")
	plaintext = append(plaintext, make([]byte, 16-(len(plaintext)%16))...)

	ciphertext, err := encryptXTS128(key, tweak, plaintext)
	if err != nil {
		t.Fatalf("encryptXTS128 returned error: %v", err)
	}

	result, err := OpenXTS128(key, tweak, ciphertext)
	if err != nil {
		t.Fatalf("OpenXTS128 returned error: %v", err)
	}

	if !bytes.Equal(result, plaintext) {
		t.Fatalf("decrypted plaintext mismatch\nexpected: %x\nactual:   %x", plaintext, result)
	}
}

func TestOpenXTS128_RoundtripPartialBlock(t *testing.T) {
	key := []byte{
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
		0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f,
		0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47,
		0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f,
	}
	tweak := []byte{
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	plaintext := []byte("Short partial block plaintext, length not multiple of 16.")

	ciphertext, err := encryptXTS128(key, tweak, plaintext)
	if err != nil {
		t.Fatalf("encryptXTS128 returned error: %v", err)
	}

	result, err := OpenXTS128(key, tweak, ciphertext)
	if err != nil {
		t.Fatalf("OpenXTS128 returned error: %v", err)
	}

	if !bytes.Equal(result, plaintext) {
		t.Fatalf("decrypted plaintext mismatch\nexpected: %x\nactual:   %x", plaintext, result)
	}
}

func encryptXTS128(key, tweak, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("xts: key must be 32 bytes for AES-XTS-128")
	}
	if len(tweak) != 16 {
		return nil, errors.New("xts: tweak must be 16 bytes")
	}
	if len(plaintext) < 16 {
		return nil, errors.New("xts: plaintext too short")
	}

	dataKey := key[:16]
	tweakKey := key[16:]

	dataBlock, err := aes.NewCipher(dataKey)
	if err != nil {
		return nil, err
	}

	tweakBlock, err := aes.NewCipher(tweakKey)
	if err != nil {
		return nil, err
	}

	t := make([]byte, 16)
	copy(t, tweak)
	tweakBlock.Encrypt(t, t)

	fullBlocks := len(plaintext) / 16
	remainder := len(plaintext) % 16
	ciphertext := make([]byte, 0, len(plaintext)+16)

	if remainder == 0 {
		for i := 0; i < fullBlocks; i++ {
			ciphertext = append(ciphertext, encryptXTSBlock(dataBlock, t, plaintext[i*16:(i+1)*16])...) //nolint:gocritic
			if i+1 < fullBlocks {
				xtsMulByX(t)
			}
		}
		return ciphertext, nil
	}

	for i := 0; i < fullBlocks-1; i++ {
		ciphertext = append(ciphertext, encryptXTSBlock(dataBlock, t, plaintext[i*16:(i+1)*16])...) //nolint:gocritic
		xtsMulByX(t)
	}

	lastPlainBlock := plaintext[(fullBlocks-1)*16 : fullBlocks*16]
	partialPlaintext := plaintext[fullBlocks*16:]

	cLast := encryptXTSBlock(dataBlock, t, lastPlainBlock)
	ciphertext = append(ciphertext, cLast[:len(partialPlaintext)]...)

	pmPrime := make([]byte, 16)
	copy(pmPrime, partialPlaintext)
	copy(pmPrime[len(partialPlaintext):], lastPlainBlock[len(partialPlaintext):])

	ciphertext = append(ciphertext, encryptXTSBlock(dataBlock, t, pmPrime)...)
	return ciphertext, nil
}

func encryptXTSBlock(block cipher.Block, tweak, plaintext []byte) []byte {
	xored := make([]byte, 16)
	xorBlock(xored, plaintext, tweak)
	encrypted := make([]byte, 16)
	block.Encrypt(encrypted, xored)
	xorBlock(encrypted, encrypted, tweak)
	return encrypted
}
