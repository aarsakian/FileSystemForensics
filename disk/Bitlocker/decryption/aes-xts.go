package ccm

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

// OpenXTS128 decrypts ciphertext using AES-XTS-128.
//
// key must be 32 bytes containing two concatenated AES-128 keys:
//
//	key1 = data unit key
//	key2 = tweak key
//
// tweak must be 16 bytes.
// ciphertext can be any length >= 16. If the ciphertext length is not a
// multiple of 16, AES-XTS ciphertext stealing is applied to recover the final
// partial plaintext block.
func OpenXTS128(key, tweak, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("xts: key must be 32 bytes for AES-XTS-128")
	}
	return OpenXTS(key, tweak, ciphertext)
}

// OpenXTS decrypts ciphertext using AES-XTS with a key length of 32 or 64 bytes.
// The key is split in half: first half for data encryption, second half for tweak generation.
func OpenXTS(key, tweak, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 && len(key) != 64 {
		return nil, errors.New("xts: key must be 32 or 64 bytes")
	}
	if len(tweak) != 16 {
		return nil, errors.New("xts: tweak must be 16 bytes")
	}
	if len(ciphertext) < 16 {
		return nil, errors.New("xts: ciphertext too short")
	}

	dataKey := key[:len(key)/2]
	tweakKey := key[len(key)/2:]

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

	fullBlocks := len(ciphertext) / 16
	remainder := len(ciphertext) % 16
	plaintext := make([]byte, 0, len(ciphertext))

	if remainder == 0 {
		for i := 0; i < fullBlocks; i++ {
			ciphertextBlock := ciphertext[i*16 : (i+1)*16]
			plaintext = append(plaintext, decryptXTSBlock(dataBlock, t, ciphertextBlock)...) //nolint:gocritic

			xtsMulByX(t)

		}
		return plaintext, nil
	}

	if fullBlocks < 1 {
		return nil, errors.New("xts: ciphertext too short for ciphertext stealing")
	}

	// Decrypt all blocks before the final ciphertext-stealing block.
	for i := 0; i < fullBlocks-1; i++ {
		ciphertextBlock := ciphertext[i*16 : (i+1)*16]
		plaintext = append(plaintext, decryptXTSBlock(dataBlock, t, ciphertextBlock)...) //nolint:gocritic
		xtsMulByX(t)
	}

	// The final full block is the stolen block Cn'. The partial ciphertext follows.
	finalCiphertextBlock := ciphertext[(fullBlocks-1)*16 : fullBlocks*16]
	partialCiphertext := ciphertext[fullBlocks*16:]

	pmPrime := decryptXTSBlock(dataBlock, t, finalCiphertextBlock)
	partialPlaintext := append([]byte(nil), pmPrime[:remainder]...)

	cLast := make([]byte, 16)
	copy(cLast, partialCiphertext)
	copy(cLast[remainder:], finalCiphertextBlock[remainder:])
	ptLast := decryptXTSBlock(dataBlock, t, cLast)

	plaintext = append(plaintext, ptLast...)
	plaintext = append(plaintext, partialPlaintext...)

	return plaintext, nil
}

func decryptXTSBlock(block cipher.Block, tweak, ciphertext []byte) []byte {
	xored := make([]byte, 16)
	xorBlock(xored, ciphertext, tweak)
	decrypted := make([]byte, 16)
	block.Decrypt(decrypted, xored)
	xorBlock(decrypted, decrypted, tweak)
	return decrypted
}

func xtsMulByX(tweak []byte) {
	carry := byte(0)
	for blockIndex := 0; blockIndex < 16; blockIndex++ {
		val := (tweak[blockIndex] << 1) | carry

		carry = tweak[blockIndex] >> 7
		tweak[blockIndex] = val

	}
	if carry > 0 {
		tweak[0] ^= 0x87
	}
}
