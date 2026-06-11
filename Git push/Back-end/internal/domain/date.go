package domain

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Date représente un type date SQL (YYYY-MM-DD) sans composante horaire.
// Utilisé pour mapper les colonnes PostgreSQL de type `date` afin d'éviter
// la sérialisation parasite de l'heure ("2024-01-15T00:00:00Z" → "2024-01-15").
//
// Implémente :
//   - json.Marshaler   → sérialise en "YYYY-MM-DD"
//   - json.Unmarshaler → désérialise depuis "YYYY-MM-DD"
//   - driver.Valuer    → envoi vers PostgreSQL
//   - sql.Scanner      → lecture depuis PostgreSQL
type Date struct {
	time.Time
}

const dateFormat = "2006-01-02"

// MarshalJSON sérialise la date au format "YYYY-MM-DD".
func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Time.Format(dateFormat) + `"`), nil
}

// UnmarshalJSON désérialise une chaîne "YYYY-MM-DD" en Date.
func (d *Date) UnmarshalJSON(data []byte) error {
	// Retirer les guillemets
	s := string(data)
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return fmt.Errorf("date: format JSON invalide : %s", s)
	}
	s = s[1 : len(s)-1]

	t, err := time.Parse(dateFormat, s)
	if err != nil {
		return fmt.Errorf("date: impossible de parser %q (attendu YYYY-MM-DD) : %w", s, err)
	}
	d.Time = t
	return nil
}

// Value implémente driver.Valuer pour l'envoi vers PostgreSQL.
func (d Date) Value() (driver.Value, error) {
	return d.Time.Format(dateFormat), nil
}

// Scan implémente sql.Scanner pour la lecture depuis PostgreSQL.
func (d *Date) Scan(value interface{}) error {
	if value == nil {
		d.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		d.Time = v
		return nil
	case []byte:
		t, err := time.Parse(dateFormat, string(v))
		if err != nil {
			return fmt.Errorf("date: scan []byte échoué : %w", err)
		}
		d.Time = t
		return nil
	case string:
		t, err := time.Parse(dateFormat, v)
		if err != nil {
			return fmt.Errorf("date: scan string échoué : %w", err)
		}
		d.Time = t
		return nil
	}
	return fmt.Errorf("date: type SQL non supporté : %T", value)
}

// NewDate crée une Date à partir d'une année, mois et jour.
func NewDate(year int, month time.Month, day int) Date {
	return Date{Time: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}
