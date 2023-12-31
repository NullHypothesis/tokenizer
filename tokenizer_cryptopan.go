package main

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"sync"

	"github.com/Yawning/cryptopan"
	uuid "github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ipv4Len = 4
	ipv6Len = 16
)

var (
	errNoKey      = errors.New("key has not been initialized yet")
	errBadBlobLen = errors.New("blob length not supported")
)

// cryptoPAnTokenizer implements a tokenizer that uses Crypto-PAn to anonymize
// IP addresses.
type cryptoPAnTokenizer struct {
	sync.RWMutex
	cryptoPAn *cryptopan.Cryptopan
	key       []byte
}

func newCryptoPAnTokenizer() tokenizer {
	return &cryptoPAnTokenizer{}
}

func (c *cryptoPAnTokenizer) isBlobSupported(b []byte) bool {
	return len(b) == ipv4Len || len(b) == ipv6Len
}

func (c *cryptoPAnTokenizer) tokenize(s serializer) (token, error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.key) == 0 {
		m.numTokenized.With(prometheus.Labels{outcome: failBecause(errNoKey)}).Inc()
		return nil, errNoKey
	}
	blob := s.bytes()
	if !c.isBlobSupported(blob) {
		m.numTokenized.With(prometheus.Labels{outcome: failBecause(errBadBlobLen)}).Inc()
		return nil, errBadBlobLen
	}
	m.numTokenized.With(prometheus.Labels{outcome: success}).Inc()
	return token(c.cryptoPAn.Anonymize(blob)), nil
}

func (c *cryptoPAnTokenizer) tokenizeAndKeyID(s serializer) (token, *keyID, error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.key) == 0 {
		m.numTokenized.With(prometheus.Labels{outcome: failBecause(errNoKey)}).Inc()
		return nil, nil, errNoKey
	}
	blob := s.bytes()
	if !c.isBlobSupported(blob) {
		m.numTokenized.With(prometheus.Labels{outcome: failBecause(errBadBlobLen)}).Inc()
		return nil, nil, errBadBlobLen
	}
	m.numTokenized.With(prometheus.Labels{outcome: success}).Inc()
	return token(c.cryptoPAn.Anonymize(blob)), c.keyID(), nil
}

func (c *cryptoPAnTokenizer) keyID() *keyID {
	c.RLock()
	defer c.RUnlock()

	// A v5 UUID is supposed to hash the given name (in our case: the key)
	// using SHA-1 but let's be extra careful and hash the key using SHA-256
	// before handing it over to the uuid package.
	sum := sha256.Sum256(c.key)
	return &keyID{UUID: uuid.NewSHA1(uuidNamespace, sum[:])}
}

func (c *cryptoPAnTokenizer) resetKey() error {
	c.Lock()
	defer c.Unlock()

	var err error
	c.key = make([]byte, cryptopan.Size)
	if _, err = rand.Read(c.key); err != nil {
		return err
	}
	c.cryptoPAn, err = cryptopan.New(c.key)
	if err != nil {
		return err
	}
	return nil
}

func (c *cryptoPAnTokenizer) preservesLen() bool {
	return true
}
