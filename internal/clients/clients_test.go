package clients

import (
	"path/filepath"
	"testing"
)

const validLogo = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 40"><path d="M0 0h100v40h-100z" fill="#fff"/></svg>`

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	return NewManager(filepath.Join(t.TempDir(), "clients.json"))
}

func TestCreateGeneratesSlugID(t *testing.T) {
	m := newTestManager(t)
	c, err := m.Create("Anulekha Hospital", "ANUL", []string{"care", "trust", "community"}, validLogo)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.ID != "anulekha-hospital" {
		t.Fatalf("got ID %q, want anulekha-hospital", c.ID)
	}
	if c.Tagline[0] != "CARE" || c.Tagline[1] != "TRUST" || c.Tagline[2] != "COMMUNITY" {
		t.Fatalf("expected tagline to be uppercased, got %v", c.Tagline)
	}
}

func TestCreateDeduplicatesSlugID(t *testing.T) {
	m := newTestManager(t)
	c1, err := m.Create("Acme", "ACME1", []string{"a", "b", "c"}, validLogo)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	c2, err := m.Create("Acme", "ACME2", []string{"a", "b", "c"}, validLogo)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c1.ID == c2.ID {
		t.Fatalf("expected distinct ids, both got %q", c1.ID)
	}
	if c2.ID != "acme-2" {
		t.Fatalf("got second ID %q, want acme-2", c2.ID)
	}
}

func TestCreateRejectsDuplicateCodeCaseInsensitively(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Create("Helios Material Handling", "Helios", []string{"a", "b", "c"}, validLogo); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := m.Create("Helios Two", "HELIOS", []string{"a", "b", "c"}, validLogo); err == nil {
		t.Fatal("expected an error for a case-insensitively duplicate code")
	}
}

func TestCreateRejectsBadTagline(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Create("Acme", "ACME", []string{"only one"}, validLogo); err == nil {
		t.Fatal("expected an error for fewer than 3 tagline lines")
	}
}

func TestCreateRejectsInvalidSVG(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Create("Acme", "ACME", []string{"a", "b", "c"}, "<not-svg/>"); err == nil {
		t.Fatal("expected an error for an invalid logo SVG")
	}
}

func TestFindByCodeIsCaseInsensitive(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Create("Helios Material Handling", "Helios", []string{"a", "b", "c"}, validLogo); err != nil {
		t.Fatalf("Create: %v", err)
	}
	c, ok, err := m.Store.FindByCode("HELIOS")
	if err != nil {
		t.Fatalf("FindByCode: %v", err)
	}
	if !ok || c.Name != "Helios Material Handling" {
		t.Fatalf("got (%+v, %v)", c, ok)
	}
}

func TestUpdateChangesFieldsAndKeepsLogoWhenEmpty(t *testing.T) {
	m := newTestManager(t)
	c, err := m.Create("Acme", "ACME", []string{"a", "b", "c"}, validLogo)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := m.Update(c.ID, "Acme Corp", "ACMECORP", []string{"x", "y", "z"}, "")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Acme Corp" || updated.Code != "ACMECORP" {
		t.Fatalf("got %+v", updated)
	}
	if updated.Logo != validLogo {
		t.Fatal("expected the logo to be unchanged when logoSVG is empty")
	}
}

func TestUpdateRejectsCodeCollisionWithAnotherClient(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Create("Acme", "ACME", []string{"a", "b", "c"}, validLogo); err != nil {
		t.Fatalf("Create: %v", err)
	}
	c2, err := m.Create("Beta", "BETA", []string{"a", "b", "c"}, validLogo)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := m.Update(c2.ID, "Beta", "acme", []string{"a", "b", "c"}, ""); err == nil {
		t.Fatal("expected an error updating to a code already used by another client")
	}
}

func TestSeedHeliosOnEmptyStore(t *testing.T) {
	m := newTestManager(t)
	if err := m.SeedHelios(); err != nil {
		t.Fatalf("SeedHelios: %v", err)
	}

	c, ok, err := m.Store.FindByCode("Helios")
	if err != nil {
		t.Fatalf("FindByCode: %v", err)
	}
	if !ok {
		t.Fatal("expected a seeded Helios client")
	}
	if c.Name != "Helios Material Handling" {
		t.Fatalf("got Name=%q", c.Name)
	}
	want := []string{"ASSETS", "PEOPLE", "PERFORMANCE"}
	for i, w := range want {
		if c.Tagline[i] != w {
			t.Fatalf("got Tagline=%v, want %v", c.Tagline, want)
		}
	}
	if err := ValidateSVG([]byte(c.Logo)); err != nil {
		t.Fatalf("seeded Helios logo should itself be valid SVG: %v", err)
	}
}

func TestSeedHeliosSkipsWhenClientsExist(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Create("Acme", "ACME", []string{"a", "b", "c"}, validLogo); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := m.SeedHelios(); err != nil {
		t.Fatalf("SeedHelios: %v", err)
	}

	clients, err := m.Store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("got %d clients, want 1 (seeding should have been skipped)", len(clients))
	}
}
