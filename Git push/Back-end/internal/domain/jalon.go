package domain

import (
	"time"

	"github.com/google/uuid"
)

// Jalon mappe la table `public.jalons`.
// Représente une échéance clé d'un projet (ex: "Rendu intermédiaire", "Soutenance finale").
// Un jalon est associé à un projet et peut recevoir des livrables d'équipes.
//
// Note : DateLimite est de type Date (non nullable — contrainte SQL NOT NULL)
// car la colonne SQL est de type `date` — sérialisation JSON au format "YYYY-MM-DD".
//
// TODO Étape 5 : Implémenter la gestion des jalons dans le contexte du suivi de projet.
type Jalon struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Titre       string     `json:"titre" db:"titre"`
	Description *string    `json:"description,omitempty" db:"description"`
	DateLimite  Date       `json:"date_limite" db:"date_limite"`
	ProjetID    *uuid.UUID `json:"projet_id,omitempty" db:"projet_id"`
	CreatedAt   *time.Time `json:"created_at,omitempty" db:"created_at"`
}
