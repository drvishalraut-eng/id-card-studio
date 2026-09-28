// Package clients manages the roster of client companies: their Excel
// import code, card-front logo (validated, sanitized SVG) and footer
// tagline.
package clients

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"idcardstudio/internal/storage"
)

// Client is one row of clients.json, plus its logo. Logo is never actually
// persisted as part of clients.json (Manager always blanks it before
// saving and repopulates it after loading) — the canonical bytes live at
// data/logos/<id>.svg, matching the runtime layout, and reading/writing
// that file is entirely Manager's job so callers see one unchanged shape.
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

// List returns every client, in no particular order. Never nil — see
// employees.Store.List's comment for why that matters for the API.
func (s *Store) List() ([]Client, error) {
	list, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []Client{}
	}
	return list, nil
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
// writing it to the Store, and owns each client's logo file.
type Manager struct {
	Store    *Store
	LogosDir string
}

// NewManager returns a Manager backed by the given clients.json path,
// storing each client's logo SVG as its own file under logosDir.
func NewManager(path, logosDir string) *Manager {
	return &Manager{Store: NewStore(path), LogosDir: logosDir}
}

func (m *Manager) logoPath(id string) string {
	return filepath.Join(m.LogosDir, id+".svg")
}

// readLogo returns the logo file's contents, or "" if it doesn't exist yet
// (e.g. a client record that failed partway through Create).
func (m *Manager) readLogo(id string) (string, error) {
	data, err := os.ReadFile(m.logoPath(id))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read logo for %q: %w", id, err)
	}
	return string(data), nil
}

func (m *Manager) writeLogo(id, svg string) error {
	return storage.WriteFileAtomic(m.logoPath(id), []byte(svg))
}

// List returns every client with its logo populated from data/logos/.
func (m *Manager) List() ([]Client, error) {
	list, err := m.Store.List()
	if err != nil {
		return nil, err
	}
	for i := range list {
		logo, err := m.readLogo(list[i].ID)
		if err != nil {
			return nil, err
		}
		list[i].Logo = logo
	}
	return list, nil
}

// FindByCode looks up a client by its Excel import code (case-insensitive)
// with its logo populated from data/logos/.
func (m *Manager) FindByCode(code string) (Client, bool, error) {
	c, ok, err := m.Store.FindByCode(code)
	if err != nil || !ok {
		return Client{}, ok, err
	}
	logo, err := m.readLogo(c.ID)
	if err != nil {
		return Client{}, false, err
	}
	c.Logo = logo
	return c, true, nil
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

	c := Client{ID: id, Name: name, Code: code, Tagline: tags}
	if err := m.Store.Create(c); err != nil {
		return Client{}, err
	}
	if err := m.writeLogo(id, logoSVG); err != nil {
		return Client{}, fmt.Errorf("client created but its logo could not be saved: %w", err)
	}
	c.Logo = logoSVG
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

	updated, err := m.Store.Update(id, func(c Client) (Client, error) {
		c.Name = name
		c.Code = code
		c.Tagline = tags
		return c, nil
	})
	if err != nil {
		return Client{}, err
	}

	if logoSVG != "" {
		if err := m.writeLogo(id, logoSVG); err != nil {
			return Client{}, fmt.Errorf("client updated but its new logo could not be saved: %w", err)
		}
		updated.Logo = logoSVG
		return updated, nil
	}
	logo, err := m.readLogo(id)
	if err != nil {
		return Client{}, err
	}
	updated.Logo = logo
	return updated, nil
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
