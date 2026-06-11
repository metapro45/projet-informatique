package domain

import (
	"time"

	"github.com/google/uuid"
)

// Equipe mappe la table `public.equipes`.
// Une équipe est liée à un projet et regroupe plusieurs étudiants (via la table membres).
//
// TODO Étape 4 : Implémenter le CRUD Équipes dans le handler et le repository.
type Equipe struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	Nom       string     `json:"nom" db:"nom"`
	ProjetID  *uuid.UUID `json:"projet_id,omitempty" db:"projet_id"`
	CreatedAt *time.Time `json:"created_at,omitempty" db:"created_at"`
}
