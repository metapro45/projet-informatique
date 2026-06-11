package repository

import (
	"context"

	"github.com/google/uuid"
	"suivi-projets-backend/internal/domain"
)

// LivrableRepository définit le contrat de persistance pour l'entité Livrable.
// La gestion du fichier physique (upload/download) est traitée séparément
// via la passerelle Supabase Storage (pkg/storage/).
// Ce repository ne gère que les métadonnées en base de données.
//
// TODO Étape 6 : Implémenter l'ensemble de ces méthodes.
type LivrableRepository interface {

	// --- Étape 6 : Dépôt de livrables ---

	// Create insère les métadonnées d'un livrable après upload réussi vers Storage.
	// Le champ FichierURL est fourni par la passerelle Storage avant l'appel.
	Create(ctx context.Context, livrable *domain.Livrable) error

	// GetByID récupère un livrable par son UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Livrable, error)

	// GetByJalonID liste tous les livrables déposés pour un jalon donné.
	// Usage : vue de l'enseignant pour consulter les rendus d'une échéance.
	GetByJalonID(ctx context.Context, jalonID uuid.UUID) ([]*domain.Livrable, error)

	// GetByEquipeID liste tous les livrables déposés par une équipe.
	// Usage : historique des dépôts d'une équipe.
	GetByEquipeID(ctx context.Context, equipeID uuid.UUID) ([]*domain.Livrable, error)

	// NoterLivrable met à jour les champs `note` et `commentaire` d'un livrable.
	// Réservé au rôle "enseignant". Action distincte du dépôt initial.
	NoterLivrable(ctx context.Context, id uuid.UUID, note float64, commentaire string) error

	// Delete supprime les métadonnées. La suppression du fichier dans Storage
	// est gérée séparément par la passerelle (TODO Étape 6).
	Delete(ctx context.Context, id uuid.UUID) error
}
