package repository

import (
	"context"

	"github.com/google/uuid"
	"suivi-projets-backend/internal/domain"
)

// ProjetRepository définit le contrat de persistance pour l'entité Projet.
// L'implémentation concrète se trouve dans internal/repository/postgres/projet_repository.go
type ProjetRepository interface {

	// --- Étape 4 : CRUD Projets ---

	// Create insère un nouveau projet. Réservé au rôle "enseignant".
	Create(ctx context.Context, projet *domain.Projet) error

	// GetByID récupère un projet par son UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Projet, error)

	// GetAll récupère tous les projets. Filtrage par rôle géré au niveau usecase.
	GetAll(ctx context.Context) ([]*domain.Projet, error)

	// Update met à jour les métadonnées d'un projet. Réservé à l'enseignant propriétaire.
	Update(ctx context.Context, projet *domain.Projet) error

	// Delete supprime un projet. Réservé à l'enseignant propriétaire.
	Delete(ctx context.Context, id uuid.UUID) error

	// --- Étape 8 : Dashboard enseignant ---

	// GetByEnseignantID liste les projets créés par un enseignant donné.
	// TODO Étape 8 : Utilisé pour la vue de supervision macro de l'enseignant.
	// GetByEnseignantID(ctx context.Context, enseignantID uuid.UUID) ([]*domain.Projet, error)

	// --- Étape 8 : Dashboard étudiant ---

	// GetByEquipeID liste les projets auxquels une équipe participe.
	// TODO Étape 8 : Utilisé pour la vue "mes projets" de l'étudiant.
	// GetByEquipeID(ctx context.Context, equipeID uuid.UUID) ([]*domain.Projet, error)
}
