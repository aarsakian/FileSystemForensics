package ccm

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

type FVEAESCCMEncryptedKey struct {
	Key        [32]byte // 32-byte AES-256 keys
	Nonce      [12]byte // 12-byte nonce
	Ciphertext [32]byte // 32-byte AES-CCM encrypted key
	AuthTag    [16]byte // 16-byte authentication tag
}

// OpenCCM decrypts ciphertext||tag using AES‑CCM with:
// key: 32 bytes (AES‑256)
// nonce: 12 bytes
// aad: additional authenticated data
// ctTag: ciphertext || tag (tag is last 16 bytes)
// Decryption requires the
// key, ciphertext, associated data, and tag in order to compute the plaintext
// and associated data, and it fails if either C or A has been corrupt
// s AD(K, C, A, T, N) = (P, A), where N is thenonce used to create C and T.
func OpenCCM(key, aesccmCiphertext, nonce []byte) ([]byte, error) {
	const (
		nonceLen = 12
	)

	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes (AES‑256)")
	}
	if len(nonce) != nonceLen {
		return nil, errors.New("nonce must be 12 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	decryptedKey := make([]byte, len(aesccmCiphertext))
	stream := newCTR(block, nonce, 0)
	stream.XORKeyStream(decryptedKey, aesccmCiphertext)

	return decryptedKey, nil
}

func GetExpectedTag(key, nonce, aad, plainText, tag []byte) ([]byte, error) {
	if len(tag) != 16 {
		return nil, errors.New("tag must be 16 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	a0 := make([]byte, 16)
	counterLen := 16 - len(nonce) - 1
	a0[0] = byte(counterLen - 1)
	copy(a0[1:1+len(nonce)], nonce)
	s0 := make([]byte, 16)
	block.Encrypt(s0, a0)

	x, err := cbcMac(block, nonce, aad, plainText)
	if err != nil {
		return nil, err
	}
	//encrypt Tag = X XOR S0
	return xor16(x, s0), nil
}

// cbcMac computes CBC‑MAC over B0 || AAD || C.
func cbcMac(block cipher.Block, nonce, aad, data []byte) ([]byte, error) {
	const (
		tagLen   = 16
		adataBit = 1 << 6
		L        = 3
	)

	counterLen := 16 - len(nonce) - 1

	if len(nonce) != 12 {
		return nil, errors.New("nonce must be 12 bytes")
	}

	b0 := make([]byte, 16)

	Mprime := (tagLen - 2) / 2
	Lprime := L - 1

	flags := byte((Mprime << 3) | Lprime)
	if len(aad) > 0 {
		flags |= adataBit
	}
	b0[0] = flags
	copy(b0[1:13], nonce)

	msgLen := len(data)
	for i := 0; i < counterLen; i++ {
		b0[15-i] = byte(msgLen >> (8 * i))
	}
	//x1 = AES(0 XOR B0)
	x := make([]byte, 16)
	block.Encrypt(x, xor16(make([]byte, 16), b0))

	if len(aad) > 0 {
		b1 := make([]byte, 0, 2+len(aad))
		b1 = append(b1, byte(len(aad)>>8), byte(len(aad)))
		b1 = append(b1, aad...)
		for len(b1)%16 != 0 {
			b1 = append(b1, 0)
		}
		for step := 0; step < len(b1); step += 16 {
			blockBuf := make([]byte, 16)
			end := step + 16
			if end > len(b1) {
				end = len(b1)
			}
			copy(blockBuf, b1[step:end])
			block.Encrypt(x, xor16(x, blockBuf))
		}
	}

	for step := 0; step < len(data); step += 16 {
		blockBuf := make([]byte, 16)
		end := step + 16
		if end > len(data) {
			end = len(data)
		}
		copy(blockBuf, data[step:end])
		block.Encrypt(x, xor16(x, blockBuf))
	}

	return x[:16], nil
}

// ctrBlock computes E_K(Ai) where Ai is the counter block with given counter => KSi.
func ctrBlock(block cipher.Block, nonce []byte, counter uint32) []byte {
	L := 16 - len(nonce) - 1
	a := make([]byte, 16)
	// Flags: only L' in CTR mode
	a[0] = byte(L - 1)
	copy(a[1:1+len(nonce)], nonce)
	for i := 0; i < L; i++ {
		a[15-i] = byte(counter >> (8 * i))
	}
	block.Encrypt(a, a)
	return a
}

// newCTR returns a CTR stream starting at given counter.
func newCTR(block cipher.Block, nonce []byte, startCounter uint32) cipher.Stream {
	iv := make([]byte, 16)
	L := 16 - len(nonce) - 1
	iv[0] = byte(L - 1)
	copy(iv[1:1+len(nonce)], nonce)
	for i := 0; i < L; i++ {
		iv[15-i] = byte(startCounter >> (8 * i))
	}
	return cipher.NewCTR(block, iv)
}

func xorBlock(dst, a, b []byte) {
	for i := 0; i < len(b); i++ {
		dst[i] = a[i] ^ b[i]
	}
}

func xor16(a, b []byte) []byte {
	out := make([]byte, 16)
	for i := 0; i < 16; i++ {
		out[i] = a[i] ^ b[i]
	}
	return out
}
