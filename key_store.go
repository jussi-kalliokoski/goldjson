package goldjson

import (
	"unsafe"

	"github.com/jussi-kalliokoski/goldjson/tokens"
)

type keyStore struct {
	keys map[uintptr][]byte
}

func (s keyStore) Clone() keyStore {
	if s.keys == nil {
		return s
	}
	keys := make(map[uintptr][]byte)
	for k, v := range s.keys {
		keys[k] = v
	}
	return keyStore{keys}
}

func (s *keyStore) Put(key string) {
	if s.keys == nil {
		s.keys = make(map[uintptr][]byte)
	}

	if b := tokens.AppendString(nil, key); len(b) == len(key)+2 {
		s.keys[s.key(key)] = b
	}
}

func (s *keyStore) Append(buf []byte, key string) []byte {
	if b := s.keys[s.key(key)]; len(b) == len(key)+2 {
		return append(buf, b...)
	}
	return tokens.AppendString(buf, key)
}

func (s *keyStore) key(key string) uintptr {
	return *(*uintptr)(unsafe.Pointer(&key))
}
