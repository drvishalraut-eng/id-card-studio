// Package clients manages the roster of client companies: their Excel
// import code, card-front logo (validated, sanitized SVG) and footer
// tagline.
package clients

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"idcardstudio/internal/storage"
)

// Client is one row of clients.json.
type Client struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Code    string   `json:"code"`
	Logo    string   `json:"logo"`
	Tagline []string `json:"tagline"`
}

// Store is the mutex-protected, atomically-written clients.json file.
type Store struct {
	store *storage.Store[[]Client]
}

// NewStore opens (or prepares to create) the clients.json file at path.
func NewStore(path string) *Store {
	return &Store{store: storage.New[[]Client](path)}
}

// List returns every client, in no particular order.
func (s *Store) List() ([]Client, error) {
	return s.store.Load()
}

// Find returns the client with the given id, if any.
func (s *Store) Find(id string) (Client, bool, error) {
	clients, err := s.store.Load()
	if err != nil {
		return Client{}, false, err
	}
	for _, c := range clients {
		if c.ID == id {
			return c, true, nil
		}
	}
	return Client{}, false, nil
}

// FindByCode looks up a client by its Excel import code, case-insensitively.
func (s *Store) FindByCode(code string) (Client, bool, error) {
	clients, err := s.store.Load()
	if err != nil {
		return Client{}, false, err
	}
	for _, c := range clients {
		if strings.EqualFold(c.Code, code) {
			return c, true, nil
		}
	}
	return Client{}, false, nil
}

// Create adds a new client, rejecting a duplicate id or a code already used
// by another client (case-insensitively).
func (s *Store) Create(c Client) error {
	_, err := s.store.Update(func(clients []Client) ([]Client, error) {
		for _, existing := range clients {
			if existing.ID == c.ID {
				return clients, fmt.Errorf("client id %q already exists", c.ID)
			}
			if strings.EqualFold(existing.Code, c.Code) {
				return clients, fmt.Errorf("client code %q is already used by %q", c.Code, existing.Name)
			}
		}
		return append(clients, c), nil
	})
	return err
}

// Update replaces the stored client matching id with the value returned by
// fn, atomically, rejecting a code collision with a different client.
func (s *Store) Update(id string, fn func(Client) (Client, error)) (Client, error) {
	var updated Client
	_, err := s.store.Update(func(clients []Client) ([]Client, error) {
		for i, c := range clients {
			if c.ID != id {
				continue
			}
			next, err := fn(c)
			if err != nil {
				return clients, err
			}
			for j, other := range clients {
				if j != i && strings.EqualFold(other.Code, next.Code) {
					return clients, fmt.Errorf("client code %q is already used by %q", next.Code, other.Name)
				}
			}
			clients[i] = next
			updated = next
			return clients, nil
		}
		return clients, fmt.Errorf("no client %q", id)
	})
	return updated, err
}

// Manager validates client input (name, code, tagline, logo SVG) before
// writing it to the Store.
type Manager struct {
	Store *Store
}

// NewManager returns a Manager backed by the given clients.json path.
func NewManager(path string) *Manager {
	return &Manager{Store: NewStore(path)}
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	s := nonSlugChars.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(s, "-")
}

func (m *Manager) uniqueID(base string) (string, error) {
	if base == "" {
		base = "client"
	}
	id := base
	for n := 2; ; n++ {
		_, found, err := m.Store.Find(id)
		if err != nil {
			return "", err
		}
		if !found {
			return id, nil
		}
		id = fmt.Sprintf("%s-%d", base, n)
	}
}

func normalizeTagline(lines []string) ([]string, error) {
	out := make([]string, 0, 3)
	for _, l := range lines {
		l = strings.ToUpper(strings.TrimSpace(l))
		if l != "" {
			out = append(out, l)
		}
	}
	if len(out) != 3 {
		return nil, errors.New("tagline must have exactly 3 non-empty lines")
	}
	return out, nil
}

// Create validates and adds a new client, generating its id from name.
func (m *Manager) Create(name, code string, tagline []string, logoSVG string) (Client, error) {
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(code)
	if name == "" {
		return Client{}, errors.New("client name is required")
	}
	if code == "" {
		return Client{}, errors.New("client code is required")
	}
	tags, err := normalizeTagline(tagline)
	if err != nil {
		return Client{}, err
	}
	if err := ValidateSVG([]byte(logoSVG)); err != nil {
		return Client{}, err
	}

	id, err := m.uniqueID(slugify(name))
	if err != nil {
		return Client{}, err
	}
	c := Client{ID: id, Name: name, Code: code, Logo: logoSVG, Tagline: tags}
	if err := m.Store.Create(c); err != nil {
		return Client{}, err
	}
	return c, nil
}

// Update edits an existing client's name, code and tagline. An empty
// logoSVG keeps the client's current logo.
func (m *Manager) Update(id, name, code string, tagline []string, logoSVG string) (Client, error) {
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(code)
	if name == "" {
		return Client{}, errors.New("client name is required")
	}
	if code == "" {
		return Client{}, errors.New("client code is required")
	}
	tags, err := normalizeTagline(tagline)
	if err != nil {
		return Client{}, err
	}
	if logoSVG != "" {
		if err := ValidateSVG([]byte(logoSVG)); err != nil {
			return Client{}, err
		}
	}

	return m.Store.Update(id, func(c Client) (Client, error) {
		c.Name = name
		c.Code = code
		c.Tagline = tags
		if logoSVG != "" {
			c.Logo = logoSVG
		}
		return c, nil
	})
}

// SeedHelios adds the built-in Helios Material Handling client if the
// client list is empty (first run).
func (m *Manager) SeedHelios() error {
	clients, err := m.Store.List()
	if err != nil {
		return err
	}
	if len(clients) > 0 {
		return nil
	}
	_, err = m.Create("Helios Material Handling", "Helios", []string{"ASSETS", "PEOPLE", "PERFORMANCE"}, HeliosLogoSVG())
	return err
}
