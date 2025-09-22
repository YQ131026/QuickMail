package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"QuickMail/internal/crypto"
)

var ErrProviderNotFound = errors.New("provider not found")

// Provider contains provider settings with decrypted password for runtime use.
type Provider struct {
	Name     string
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
}

type providerRecord struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	UseTLS   bool   `json:"use_tls"`
}

type ProviderInput struct {
	Name     string `json:"name"`     // required
	Host     string `json:"host"`     // required
	Port     int    `json:"port"`     // required
	Username string `json:"username"` // required
	Password string `json:"password"` // required plain password
	From     string `json:"from"`     // optional but recommended
	UseTLS   bool   `json:"use_tls"`
}

type ProviderResponse struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	From     string `json:"from"`
	UseTLS   bool   `json:"use_tls"`
}

type Store struct {
	mu      sync.RWMutex
	path    string
	secret  []byte
	records map[string]providerRecord
}

func NewStore(path string, secret []byte) (*Store, error) {
	if len(secret) == 0 {
		return nil, errors.New("secret must not be empty")
	}

	s := &Store{
		path:    path,
		secret:  secret,
		records: make(map[string]providerRecord),
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s.flushLocked()
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	var list []providerRecord
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	for _, rec := range list {
		s.records[rec.Name] = rec
	}

	return nil
}

func (s *Store) flushLocked() error {
	list := make([]providerRecord, 0, len(s.records))
	for _, rec := range s.records {
		list = append(list, rec)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0o600)
}

func (s *Store) ListProviders() ([]ProviderResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]ProviderResponse, 0, len(s.records))
	for _, rec := range s.records {
		res = append(res, ProviderResponse{
			Name:     rec.Name,
			Host:     rec.Host,
			Port:     rec.Port,
			Username: rec.Username,
			From:     rec.From,
			UseTLS:   rec.UseTLS,
		})
	}

	return res, nil
}

func (s *Store) GetProvider(name string) (Provider, error) {
	s.mu.RLock()
	rec, ok := s.records[name]
	s.mu.RUnlock()
	if !ok {
		return Provider{}, ErrProviderNotFound
	}

	password, err := crypto.Decrypt(s.secret, rec.Password)
	if err != nil {
		return Provider{}, err
	}

	return Provider{
		Name:     rec.Name,
		Host:     rec.Host,
		Port:     rec.Port,
		Username: rec.Username,
		Password: password,
		From:     rec.From,
		UseTLS:   rec.UseTLS,
	}, nil
}

func (s *Store) UpsertProvider(input ProviderInput) error {
	if input.Name == "" || input.Host == "" || input.Username == "" || input.Password == "" || input.Port == 0 {
		return errors.New("missing required provider fields")
	}

	encrypted, err := crypto.Encrypt(s.secret, input.Password)
	if err != nil {
		return err
	}

	rec := providerRecord{
		Name:     input.Name,
		Host:     input.Host,
		Port:     input.Port,
		Username: input.Username,
		Password: encrypted,
		From:     input.From,
		UseTLS:   input.UseTLS,
	}

	s.mu.Lock()
	s.records[input.Name] = rec
	err = s.flushLocked()
	s.mu.Unlock()
	return err
}

func (s *Store) DeleteProvider(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.records[name]; !ok {
		return ErrProviderNotFound
	}

	delete(s.records, name)
	return s.flushLocked()
}
