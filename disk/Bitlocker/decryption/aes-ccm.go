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
func OpenCCM(key, nonce, aad, ctTag []byte) ([]byte, error) {
	const (
		nonceLen = 12
		tagLen   = 16
	)

	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes (AES‑256)")
	}
	if len(nonce) != nonceLen {
		return nil, errors.New("nonce must be 12 bytes")
	}
	if len(ctTag) < tagLen {
		return nil, errors.New("ciphertext+tag too short")
	}

	ct := ctTag[:len(ctTag)-tagLen]
	tag := ctTag[len(ctTag)-tagLen:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 1. Decrypt ciphertext using CTR with counters starting at 1
	pt := make([]byte, len(ct))
	stream := newCTR(block, nonce, 1)
	stream.XORKeyStream(pt, ct)

	// 2. Compute CBC‑MAC over B0 || AAD || P
	mac, err := cbcMac(block, nonce, aad, pt)
	if err != nil {
		return nil, err
	}

	// 3. Compute S0 = E_K(A0) and derive expected tag = mac XOR S0
	s0 := ctrBlock(block, nonce, 0)
	expectedTag := xor16(mac, s0)

	// 4. Verify tag
	if !equal16(expectedTag, tag) {
		return nil, errors.New("ccm: authentication failed")
	}

	return pt, nil
}

// cbcMac computes CBC‑MAC over B0 || AAD || C.
func cbcMac(block cipher.Block, nonce, aad, ct []byte) ([]byte, error) {
	const (
		tagLen   = 16
		L        = 2 // length field size
		adataBit = 1 << 6
	)

	if len(nonce) != 12 {
		return nil, errors.New("nonce must be 12 bytes")
	}

	// Build B0
	b0 := make([]byte, 16)
	// Flags: Adata present + M' + L'
	// M' = (tagLen-2)/2 = 7, L' = L-1 = 1
	b0[0] = adataBit | (7 << 3) | (L - 1)
	copy(b0[1:1+12], nonce)
	msgLen := len(ct)
	b0[13] = byte(msgLen >> 8)
	b0[14] = byte(msgLen)

	// Start CBC‑MAC
	x := make([]byte, 16) // X0 = 0
	buf := make([]byte, 16)

	// Process B0
	xorBlock(buf, x, b0)
	block.Encrypt(x, buf)

	// Process AAD (length‑prefixed, then padded)
	if len(aad) > 0 {
		// Encode AAD length as 2 bytes (enough for our use)
		aadLenBlock := make([]byte, 0, 16)
		aadLenBlock = append(aadLenBlock, byte(len(aad)>>8), byte(len(aad)))
		aadLenBlock = append(aadLenBlock, aad...)
		// Pad to 16‑byte blocks
		for len(aadLenBlock)%16 != 0 {
			aadLenBlock = append(aadLenBlock, 0)
		}
		for i := 0; i < len(aadLenBlock); i += 16 {
			xorBlock(buf, x, aadLenBlock[i:i+16])
			block.Encrypt(x, buf)
		}
	}

	// Process ciphertext
	if len(ct) > 0 {
		ctPadded := ct
		if len(ctPadded)%16 != 0 {
			ctPadded = append(append([]byte{}, ctPadded...), make([]byte, 16-len(ctPadded)%16)...)
		}
		for i := 0; i < len(ctPadded); i += 16 {
			xorBlock(buf, x, ctPadded[i:i+16])
			block.Encrypt(x, buf)
		}
	}

	return x[:tagLen], nil
}

// ctrBlock computes E_K(Ai) where Ai is the counter block with given counter.
func ctrBlock(block cipher.Block, nonce []byte, counter uint16) []byte {
	const L = 2
	a := make([]byte, 16)
	// Flags: only L' in CTR mode
	a[0] = (L - 1)
	copy(a[1:1+12], nonce)
	a[13] = byte(counter >> 8)
	a[14] = byte(counter)
	// a[15] unused for our sizes
	block.Encrypt(a, a)
	return a
}

// newCTR returns a CTR stream starting at given counter.
func newCTR(block cipher.Block, nonce []byte, startCounter uint16) cipher.Stream {
	iv := make([]byte, 16)
	const L = 2
	iv[0] = (L - 1)
	copy(iv[1:1+12], nonce)
	iv[13] = byte(startCounter >> 8)
	iv[14] = byte(startCounter)
	return cipher.NewCTR(block, iv)
}

func xorBlock(dst, a, b []byte) {
	for i := 0; i < 16; i++ {
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

func equal16(a, b []byte) bool {
	if len(a) != 16 || len(b) != 16 {
		return false
	}
	var v byte
	for i := 0; i < 16; i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
