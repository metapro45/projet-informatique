package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"suivi-projets-backend/internal/domain"
)

// ProjetRepository implémente repository.ProjetRepository via PostgreSQL.
type ProjetRepository struct {
	db *sql.DB
}

func NewProjetRepository(db *sql.DB) *ProjetRepository {
	return &ProjetRepository{db: db}
}

const sqlGetAllProjets = `
	SELECT id, titre, description, statut,
	       date_debut, date_fin, enseignant_id, group_id, created_at
	FROM projets
	ORDER BY created_at DESC
`
const sqlGetProjetByID = `
	SELECT id, titre, description, statut,
	       date_debut, date_fin, enseignant_id, group_id, created_at
	FROM projets WHERE id = $1
`
const sqlCreateProjet = `
	INSERT INTO projets (id, titre, description, statut, date_debut, date_fin, enseignant_id, group_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING id, titre, description, statut, date_debut, date_fin, enseignant_id, group_id, created_at
`
const sqlUpdateProjet = `
	UPDATE projets
	SET titre=$2, description=$3, statut=$4, date_debut=$5, date_fin=$6, group_id=$7
	WHERE id = $1
	RETURNING id, titre, description, statut, date_debut, date_fin, enseignant_id, group_id, created_at
`

func (r *ProjetRepository) GetAll(ctx context.Context) ([]*domain.Projet, error) {
	rows, err := r.db.QueryContext(ctx, sqlGetAllProjets)
	if err != nil {
		return nil, fmt.Errorf("projet.GetAll: %w", err)
	}
	defer rows.Close()

	var projets []*domain.Projet
	for rows.Next() {
		p, err := scanProjet(rows)
		if err != nil {
			return nil, fmt.Errorf("projet.GetAll scan: %w", err)
		}
		projets = append(projets, p)
	}
	return projets, nil
}

func (r *ProjetRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Projet, error) {
	row := r.db.QueryRowContext(ctx, sqlGetProjetByID, id)
	p, err := scanProjetRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("projet.GetByID: %w", err)
	}
	return p, nil
}

func (r *ProjetRepository) Create(ctx context.Context, projet *domain.Projet) error {
	projet.ID = uuid.New()
	row := r.db.QueryRowContext(ctx, sqlCreateProjet,
		projet.ID, projet.Titre, projet.Description,
		projet.Statut, projet.DateDebut, projet.DateFin, projet.EnseignantID,
		projet.GroupID,
	)
	created, err := scanProjetRow(row)
	if err != nil {
		return fmt.Errorf("projet.Create: %w", err)
	}
	*projet = *created
	return nil
}

func (r *ProjetRepository) Update(ctx context.Context, projet *domain.Projet) error {
	row := r.db.QueryRowContext(ctx, sqlUpdateProjet,
		projet.ID, projet.Titre, projet.Description,
		projet.Statut, projet.DateDebut, projet.DateFin, projet.GroupID,
	)
	updated, err := scanProjetRow(row)
	if err == sql.ErrNoRows {
		return fmt.Errorf("projet.Update: projet %s introuvable", projet.ID)
	}
	if err != nil {
		return fmt.Errorf("projet.Update: %w", err)
	}
	*projet = *updated
	return nil
}

// Delete supprime un projet et toutes ses dépendances via une seule requête SQL.
// On utilise WITH (CTE) pour tout supprimer en cascade en une seule passe atomique.
// Cela évite les problèmes de pooler Supabase avec les transactions multi-requêtes.
func (r *ProjetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		WITH
		del_taches AS (
			DELETE FROM taches
			WHERE equipe_id IN (SELECT id FROM equipes WHERE projet_id = $1)
		),
		del_livrables AS (
			DELETE FROM livrables
			WHERE equipe_id IN (SELECT id FROM equipes WHERE projet_id = $1)
		),
		del_membres AS (
			DELETE FROM membres
			WHERE equipe_id IN (SELECT id FROM equipes WHERE projet_id = $1)
		),
		del_equipes AS (
			DELETE FROM equipes WHERE projet_id = $1
		),
		del_jalons AS (
			DELETE FROM jalons WHERE projet_id = $1
		)
		DELETE FROM projets WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("projet.Delete: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("projet.Delete: projet %s introuvable", id)
	}
	return nil
}

// ── Helpers de scan ───────────────────────────────────────────────────────────

type scanRow interface {
	Scan(dest ...interface{}) error
}

func scanProjet(rows *sql.Rows) (*domain.Projet, error) {
	return scanProjetFields(rows)
}

func scanProjetRow(row *sql.Row) (*domain.Projet, error) {
	return scanProjetFields(row)
}

func scanProjetFields(s scanRow) (*domain.Projet, error) {
	p := &domain.Projet{}
	var description sql.NullString
	var dateDebut, dateFin sql.NullTime
	var enseignantID, groupID uuid.NullUUID
	var createdAt sql.NullTime

	err := s.Scan(
		&p.ID, &p.Titre, &description, &p.Statut,
		&dateDebut, &dateFin, &enseignantID, &groupID, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		p.Description = &description.String
	}
	if dateDebut.Valid {
		d := domain.Date{Time: dateDebut.Time}
		p.DateDebut = &d
	}
	if dateFin.Valid {
		d := domain.Date{Time: dateFin.Time}
		p.DateFin = &d
	}
	if enseignantID.Valid {
		p.EnseignantID = &enseignantID.UUID
	}
	if groupID.Valid {
		p.GroupID = &groupID.UUID
	}
	if createdAt.Valid {
		t := createdAt.Time.UTC()
		p.CreatedAt = &t
	}
	return p, nil
}
