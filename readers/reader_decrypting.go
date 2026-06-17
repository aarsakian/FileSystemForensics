package readers

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	ccm "github.com/aarsakian/FileSystemForensics/disk/Bitlocker/decryption"
)

type DecryptingReader struct {
	underlying      DiskReader
	fvek            []byte
	sectorSize      int64
	partitionOffset int64
	method          string
}

func NewDecryptingReader(underlying DiskReader, fvek []byte, sectorSize int64, partitionOffset int64, method string) (*DecryptingReader, error) {
	cleanMethod := strings.ToUpper(strings.TrimSpace(method))
	if cleanMethod == "" {
		return nil, errors.New("encryption method is required")
	}

	if strings.Contains(cleanMethod, "XTS") {
		if len(fvek) != 32 && len(fvek) != 64 {
			return nil, fmt.Errorf("unsupported FVEK length %d for AES-XTS", len(fvek))
		}
	} else if strings.Contains(cleanMethod, "CBC") {
		if len(fvek) != 16 && len(fvek) != 32 {
			return nil, fmt.Errorf("unsupported FVEK length %d for AES-CBC", len(fvek))
		}
	} else {
		return nil, fmt.Errorf("unsupported BitLocker encryption method %q", method)
	}

	if sectorSize <= 0 {
		return nil, errors.New("sector size must be positive")
	}

	return &DecryptingReader{
		underlying:      underlying,
		fvek:            append([]byte(nil), fvek...),
		sectorSize:      sectorSize,
		partitionOffset: partitionOffset,
		method:          cleanMethod,
	}, nil
}

func (dr *DecryptingReader) CreateHandler() {
	dr.underlying.CreateHandler()
}

func (dr *DecryptingReader) CloseHandler() {
	dr.underlying.CloseHandler()
}

func (dr *DecryptingReader) ReadFile(offset int64, size int) ([]byte, error) {
	data, err := dr.underlying.ReadFile(offset, size)
	if err != nil {
		return nil, err
	}

	if strings.Contains(dr.method, "XTS") {
		return dr.decryptXTS(offset, data)
	}
	if strings.Contains(dr.method, "CBC") {
		return dr.decryptCBC(offset, data)
	}

	return nil, fmt.Errorf("unsupported decryption method %q", dr.method)
}

func (dr *DecryptingReader) GetDiskSize() int64 {
	return dr.underlying.GetDiskSize()
}

func (dr *DecryptingReader) decryptXTS(offset int64, data []byte) ([]byte, error) {
	plaintext := make([]byte, len(data))
	copy(plaintext, data)

	if len(plaintext) == 0 {
		return plaintext, nil
	}

	firstSector := (offset - dr.partitionOffset) / dr.sectorSize
	if firstSector < 0 {
		return nil, errors.New("read offset before partition start")
	}

	sectorCount := int64(len(plaintext)) / dr.sectorSize
	if int64(len(plaintext))%dr.sectorSize != 0 {
		sectorCount++
	}

	for i := int64(0); i < sectorCount; i++ {
		start := i * dr.sectorSize
		end := start + dr.sectorSize
		if end > int64(len(plaintext)) {
			end = int64(len(plaintext))
		}

		chunk := plaintext[start:end]
		tweak := make([]byte, 16)
		binary.LittleEndian.PutUint64(tweak, uint64(firstSector+i))
		decrypted, err := ccm.OpenXTS(dr.fvek, tweak, chunk)
		if err != nil {
			return nil, err
		}
		copy(plaintext[start:], decrypted)
	}

	return plaintext, nil
}

func (dr *DecryptingReader) decryptCBC(offset int64, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}
	if len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext length must be multiple of %d", aes.BlockSize)
	}

	sectorNum := (offset - dr.partitionOffset) / dr.sectorSize
	if sectorNum < 0 {
		return nil, errors.New("read offset before partition start")
	}

	iv := make([]byte, aes.BlockSize)
	binary.LittleEndian.PutUint64(iv, uint64(sectorNum))

	block, err := aes.NewCipher(dr.fvek)
	if err != nil {
		return nil, err
	}

	block.Encrypt(iv, iv)
	decrypted := make([]byte, len(data))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decrypted, data)

	return decrypted, nil
}
