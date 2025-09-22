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
	apiKey  string
	port    string
	records map[string]providerRecord
}

func NewStore(settings Settings) (*Store, error) {
	secret := []byte(settings.Secret)
	if len(secret) == 0 {
		return nil, errors.New("secret must not be empty")
	}

	s := &Store{
		path:    settings.ConfigPath,
		secret:  secret,
		apiKey:  settings.APIKey,
		port:    settings.Port,
		records: make(map[string]providerRecord),
	}

	needsFlush := false
	for _, rec := range settings.providers {
		if rec.Name == "" {
			continue
		}

		if _, err := crypto.Decrypt(secret, rec.Password); err != nil {
			encrypted, encErr := crypto.Encrypt(secret, rec.Password)
			if encErr != nil {
				return nil, encErr
			}
			rec.Password = encrypted
			needsFlush = true
		}

		s.records[rec.Name] = rec
	}

	if needsFlush {
		if err := s.flush(); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *Store) flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flushLocked()
}

func (s *Store) flushLocked() error {
	list := make([]providerRecord, 0, len(s.records))
	for _, rec := range s.records {
		list = append(list, rec)
	}

	cfg := fileConfig{
		APIKey:    s.apiKey,
		Secret:    string(s.secret),
		Port:      s.port,
		Providers: list,
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
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
