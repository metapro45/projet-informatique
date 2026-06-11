package domain

import (
	"time"

	"github.com/google/uuid"
)

// Membre mappe la table `public.membres`.
// Table de jointure N-N entre users et equipes, enrichie d'un rôle dans l'équipe.
// Exemples de role_equipe : "chef_de_projet" | "developpeur" | "designer"
//
// TODO Étape 4 : Implémenter l'ajout/suppression de membres dans une équipe.
type Membre struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	EquipeID   *uuid.UUID `json:"equipe_id,omitempty" db:"equipe_id"`
	UserID     *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	RoleEquipe *string    `json:"role_equipe,omitempty" db:"role_equipe"`
	JoinedAt   *time.Time `json:"joined_at,omitempty" db:"joined_at"`

	// Enrichissement — données du profil (joint depuis profiles)
	Nom    string `json:"nom,omitempty"`
	Prenom string `json:"prenom,omitempty"`
	Email  string `json:"email,omitempty"`
	Role   string `json:"role,omitempty"`
}
