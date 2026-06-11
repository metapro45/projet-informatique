package domain

import (
	"time"

	"github.com/google/uuid"
)

// RoleUtilisateur représente les 3 profils possibles dans le système v2.
// Valeurs en BDD : "admin" | "encadrant" | "etudiant"
// Note : l'ancien rôle "enseignant" est maintenant "encadrant"
type RoleUtilisateur string

const (
	RoleAdmin     RoleUtilisateur = "admin"
	RoleEncadrant RoleUtilisateur = "encadrant"
	RoleEtudiant  RoleUtilisateur = "etudiant"
)

// Profile mappe la table `public.profiles` de Supabase (v2).
// Remplace l'ancienne table `public.users`.
// L'authentification est déléguée à Supabase Auth (JWT ES256).
// id = même UUID que auth.users.id
type Profile struct {
	ID                  uuid.UUID       `json:"id"`
	Email               string          `json:"email"`
	Nom                 string          `json:"nom"`
	Prenom              *string         `json:"prenom,omitempty"`
	Role                RoleUtilisateur `json:"role"`
	GroupID             *uuid.UUID      `json:"group_id,omitempty"`
	MustChangePassword  bool            `json:"must_change_password"`
	IsActive            bool            `json:"is_active"`
	CreatedAt           *time.Time      `json:"created_at,omitempty"`
	UpdatedAt           *time.Time      `json:"updated_at,omitempty"`
}

// IsAdmin vérifie si le profil est un administrateur.
func (p *Profile) IsAdmin() bool { return p.Role == RoleAdmin }

// IsEncadrant vérifie si le profil est un encadrant.
func (p *Profile) IsEncadrant() bool { return p.Role == RoleEncadrant }

// IsEtudiant vérifie si le profil est un étudiant.
func (p *Profile) IsEtudiant() bool { return p.Role == RoleEtudiant }
