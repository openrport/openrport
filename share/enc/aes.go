package enc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// AesEncryptStringToBase64String a user friendly wrapper of AesEncrypt which converts a password of any length to 32byte aes256 key
// and returns a base64 encoded encrypted data
func Aes256EncryptByPassToBase64String(payload []byte, password string) (encryptedBase64Data string, err error) {
	aes32Key, err := convertPasswordToFixedLength32ByteAesKey(password)
	if err != nil {
		return encryptedBase64Data, err
	}

	var encryptedBytes []byte
	encryptedBytes, err = Aes256Encrypt(payload, aes32Key)
	if err != nil {
		return
	}

	encryptedBase64Data = base64.StdEncoding.EncodeToString(encryptedBytes)
	return
}

func Aes256Encrypt(payload, aes32Key []byte) (encryptedData []byte, err error) {
	keyLen := len(aes32Key)
	if keyLen != 32 {
		err = fmt.Errorf("invalid aes32Key length: a 32 bytes key is expected but %d byts key is provided", keyLen)
		return
	}

	//Create a new Cipher Block from the aes32Key
	block, err := aes.NewCipher(aes32Key)
	if err != nil {
		return encryptedData, err
	}

	//Create a new GCM - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	//https://golang.org/pkg/crypto/cipher/#NewGCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return encryptedData, err
	}

	//Create a nonce. Nonce should be from GCM
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return encryptedData, err
	}

	//Encrypt the data using aesGCM.Seal
	//Since we don't want to save the nonce somewhere else in this case, we add it as a prefix to the encrypted data. The first nonce argument in Seal is the prefix.
	ciphertext := aesGCM.Seal(nonce, nonce, payload, nil)
	return ciphertext, nil
}

// Aes256DecryptByPassFromBase64String a user friendly wrapper of AesDecrypt and the reverse operation of AesEncryptStringToBase64String:
// it accepts base64 encrypted string and a password and returns a decrypted data
func Aes256DecryptByPassFromBase64String(encryptedBase64Data, password string) (decryptedBytes []byte, err error) {
	aes32Key, err := convertPasswordToFixedLength32ByteAesKey(password)
	if err != nil {
		return decryptedBytes, err
	}

	encryptedBytesData, err := base64.StdEncoding.DecodeString(encryptedBase64Data)
	if err != nil {
		return decryptedBytes, err
	}

	return AesDecrypt(encryptedBytesData, aes32Key)
}

func AesDecrypt(encryptedData, key []byte) (decryptedData []byte, err error) {
	//Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return decryptedData, err
	}

	//Create a new GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return decryptedData, err
	}

	//Get the nonce size
	nonceSize := aesGCM.NonceSize()

	if len(encryptedData) <= nonceSize {
		return decryptedData, fmt.Errorf("invalid encrypted value provided: invalid nonce length")
	}

	//Extract the nonce from the encrypted data
	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]

	//Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return decryptedData, err
	}

	return plaintext, nil
}

func convertPasswordToFixedLength32ByteAesKey(password string) ([]byte, error) {
	hasher := sha256.New()
	_, err := hasher.Write([]byte(password))
	if err != nil {
		return []byte{}, err
	}
	return hasher.Sum(nil), nil
}
