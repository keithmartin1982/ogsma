package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	
	_ "embed"
	
	ecies "github.com/ecies/go/v2"
)

type Keystore struct {
	Username   []byte          `json:"username"`
	ID         []byte          `json:"id"`
	PublicKey  []byte          `json:"publicKey"`
	PrivateKey []byte          `json:"privateKey"`
	Contacts   []*StoreContact `json:"contacts"`
}

type StoreContact struct {
	PublicKey []byte `json:"publicKey"`
	ID        []byte `json:"id"`
	Username  []byte `json:"username"`
}

type Keys struct {
	Username   string `json:"username"`
	ID         string `json:"id"`
	PublicKey  *ecies.PublicKey
	PrivateKey *ecies.PrivateKey
	Contacts   []*Contact
}

type Contact struct {
	PublicKey *ecies.PublicKey
	ID        string
	Username  string
}

type Encryption struct {
	configKeystore []byte
	password       string
	iter           int
	keys           *Keys
}

func (e *Encryption) passwordDecrypt(cipherText []byte) ([]byte, error) {
	if !bytes.ContainsAny(cipherText, "-") {
		return nil, errors.New("invalid data")
	}
	data := bytes.Split(cipherText, []byte("-"))
	salt := make([]byte, hex.DecodedLen(len(data[0])))
	if _, err := hex.Decode(salt, data[0]); err != nil {
		return nil, errors.New("invalid data: salt" + err.Error())
	}
	iv := make([]byte, hex.DecodedLen(len(data[1])))
	if _, err := hex.Decode(iv, data[1]); err != nil {
		return nil, errors.New("invalid data: iv" + err.Error())
	}
	ciphertext := make([]byte, hex.DecodedLen(len(data[2])))
	if _, err := hex.Decode(ciphertext, data[2]); err != nil {
		return nil, errors.New("invalid data: ciphertext" + err.Error())
	}
	key, err := pbkdf2.Key(sha256.New, e.password, salt, e.iter, 32)
	if err != nil {
		return nil, errors.New("error generating pbkdf2 key:" + err.Error())
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("decrypt failed:" + err.Error())
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.New("decrypt failed:" + err.Error())
	}
	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decrypt failed:" + err.Error())
	}
	return plaintext, nil
}

func (e *Encryption) loadKeys() error {
	keystoreFileBytes, err := e.passwordDecrypt(e.configKeystore)
	if err != nil {
		return errors.New("Error decrypting keystore:" + err.Error())
	}
	e.password = ""
	ks := &Keystore{}
	if err := json.Unmarshal(keystoreFileBytes, ks); err != nil {
		return errors.New("Error unmarshaling keystore:" + err.Error())
	}
	publicKeyFromBytes, err := ecies.NewPublicKeyFromBytes(ks.PublicKey)
	if err != nil {
		return errors.New("Error decrypting public key:" + err.Error())
	}
	e.keys = &Keys{
		Username:   string(ks.Username),
		ID:         string(ks.ID),
		PublicKey:  publicKeyFromBytes,
		PrivateKey: ecies.NewPrivateKeyFromBytes(ks.PrivateKey),
		Contacts:   []*Contact{},
	}
	for _, i := range ks.Contacts {
		contactPublicKeyFromBytes, err := ecies.NewPublicKeyFromBytes(i.PublicKey)
		if err != nil {
			return errors.New("Error decrypting contact:" + err.Error())
		}
		e.keys.Contacts = append(e.keys.Contacts, &Contact{
			PublicKey: contactPublicKeyFromBytes,
			ID:        string(i.ID),
			Username:  string(i.Username),
		})
	}
	return nil
}

func (e *Encryption) publicEncrypt(plaintext []byte, publicKey *ecies.PublicKey) ([]byte, error) {
	ciphertext, err := ecies.Encrypt(publicKey, plaintext)
	if err != nil {
		return nil, errors.New("error encrypting with public key" + err.Error())
	}
	return ciphertext, nil
}

func (e *Encryption) privateDecrypt(ciphertext []byte) ([]byte, error) {
	plaintext, err := ecies.Decrypt(e.keys.PrivateKey, ciphertext)
	if err != nil {
		return nil, errors.New("error decrypting with private key" + err.Error())
	}
	return plaintext, nil
}
