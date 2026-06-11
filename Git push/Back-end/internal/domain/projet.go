package domain

import (
	"time"

	"github.com/google/uuid"
)

// StatutProjet représente le cycle de vie d'un projet.
// Valeurs attendues en BDD : "actif" | "archive" | "termine"
type StatutProjet string

const (
	StatutProjetActif   StatutProjet = "actif"
	StatutProjetArchive StatutProjet = "archive"
	StatutProjetTermine StatutProjet = "termine"
)

// Projet mappe la table `public.projets`.
// Un projet est créé par un enseignant et regroupe plusieurs équipes d'étudiants.
//
// Note : DateDebut et DateFin sont de type Date (pas time.Time) car la colonne
// SQL est de type `date` — sérialisation JSON au format "YYYY-MM-DD".
type Projet struct {
	ID           uuid.UUID    `json:"id" db:"id"`
	Titre        string       `json:"titre" db:"titre"`
	Description  *string      `json:"description,omitempty" db:"description"`
	Statut       StatutProjet `json:"statut" db:"statut"`
	DateDebut    *Date        `json:"date_debut,omitempty" db:"date_debut"`
	DateFin      *Date        `json:"date_fin,omitempty" db:"date_fin"`
	EnseignantID *uuid.UUID   `json:"enseignant_id,omitempty" db:"enseignant_id"`
	GroupID      *uuid.UUID   `json:"group_id,omitempty" db:"group_id"`
	CreatedAt    *time.Time   `json:"created_at,omitempty" db:"created_at"`
}
