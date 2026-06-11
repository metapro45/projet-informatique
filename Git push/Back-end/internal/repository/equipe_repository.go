package repository

import (
	"context"

	"github.com/google/uuid"
	"suivi-projets-backend/internal/domain"
)

// EquipeRepository définit le contrat de persistance pour l'entité Equipe.
// L'implémentation concrète se trouve dans internal/repository/postgres/equipe_repository.go
//
// TODO Étape 4 : Implémenter l'ensemble de ces méthodes.
type EquipeRepository interface {

	// --- Étape 4 : CRUD Équipes ---

	// Create insère une nouvelle équipe liée à un projet. Réservé à l'enseignant.
	Create(ctx context.Context, equipe *domain.Equipe) error

	// GetByID récupère une équipe par son UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Equipe, error)

	// GetByProjetID liste toutes les équipes d'un projet donné.
	GetByProjetID(ctx context.Context, projetID uuid.UUID) ([]*domain.Equipe, error)

	// Update met à jour le nom d'une équipe.
	Update(ctx context.Context, equipe *domain.Equipe) error

	// Delete supprime une équipe et ses membres associés (cascade BDD).
	Delete(ctx context.Context, id uuid.UUID) error
}
