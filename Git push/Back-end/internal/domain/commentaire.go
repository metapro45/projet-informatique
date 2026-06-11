package domain

import (
	"time"

	"github.com/google/uuid"
)

// Commentaire mappe la table `public.tache_commentaires`.
// Un commentaire est lié à une tâche et à son auteur.
type Commentaire struct {
	ID        uuid.UUID  `json:"id"`
	TacheID   uuid.UUID  `json:"tache_id"`
	AuteurID  uuid.UUID  `json:"auteur_id"`
	Contenu   string     `json:"contenu"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	// Enrichissement — nom de l'auteur (joint depuis profiles)
	AuteurNom    string `json:"auteur_nom,omitempty"`
	AuteurPrenom string `json:"auteur_prenom,omitempty"`
}
