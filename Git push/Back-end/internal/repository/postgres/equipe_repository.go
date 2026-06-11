package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"suivi-projets-backend/internal/domain"
)

type EquipeRepository struct{ db *sql.DB }

func NewEquipeRepository(db *sql.DB) *EquipeRepository {
	return &EquipeRepository{db: db}
}

const sqlGetEquipesByProjet = `
	SELECT id, nom, projet_id, created_at
	FROM equipes WHERE projet_id = $1
	ORDER BY created_at ASC
`
const sqlGetEquipeByID = `
	SELECT id, nom, projet_id, created_at
	FROM equipes WHERE id = $1
`
const sqlCreateEquipe = `
	INSERT INTO equipes (id, nom, projet_id)
	VALUES ($1, $2, $3)
	RETURNING id, nom, projet_id, created_at
`
const sqlUpdateEquipe = `
	UPDATE equipes SET nom = $2 WHERE id = $1
	RETURNING id, nom, projet_id, created_at
`

func (r *EquipeRepository) GetByProjetID(ctx context.Context, projetID uuid.UUID) ([]*domain.Equipe, error) {
	rows, err := r.db.QueryContext(ctx, sqlGetEquipesByProjet, projetID)
	if err != nil {
		return nil, fmt.Errorf("equipe.GetByProjetID: %w", err)
	}
	defer rows.Close()
	var equipes []*domain.Equipe
	for rows.Next() {
		e, err := scanEquipe(rows)
		if err != nil {
			return nil, fmt.Errorf("equipe.GetByProjetID scan: %w", err)
		}
		equipes = append(equipes, e)
	}
	return equipes, nil
}

func (r *EquipeRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Equipe, error) {
	row := r.db.QueryRowContext(ctx, sqlGetEquipeByID, id)
	e, err := scanEquipeRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("equipe.GetByID: %w", err)
	}
	return e, nil
}

func (r *EquipeRepository) Create(ctx context.Context, equipe *domain.Equipe) error {
	equipe.ID = uuid.New()
	row := r.db.QueryRowContext(ctx, sqlCreateEquipe,
		equipe.ID, equipe.Nom, equipe.ProjetID)
	created, err := scanEquipeRow(row)
	if err != nil {
		return fmt.Errorf("equipe.Create: %w", err)
	}
	*equipe = *created
	return nil
}

func (r *EquipeRepository) Update(ctx context.Context, equipe *domain.Equipe) error {
	row := r.db.QueryRowContext(ctx, sqlUpdateEquipe, equipe.ID, equipe.Nom)
	updated, err := scanEquipeRow(row)
	if err == sql.ErrNoRows {
		return fmt.Errorf("equipe.Update: equipe %s introuvable", equipe.ID)
	}
	if err != nil {
		return fmt.Errorf("equipe.Update: %w", err)
	}
	*equipe = *updated
	return nil
}

func (r *EquipeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		WITH
		del_taches    AS (DELETE FROM taches    WHERE equipe_id = $1),
		del_livrables AS (DELETE FROM livrables WHERE equipe_id = $1),
		del_membres   AS (DELETE FROM membres   WHERE equipe_id = $1)
		DELETE FROM equipes WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("equipe.Delete: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("equipe.Delete: equipe %s introuvable", id)
	}
	return nil
}

// ── Scan helpers ──────────────────────────────────────────────────────────────

func scanEquipe(rows *sql.Rows) (*domain.Equipe, error) {
	return scanEquipeFields(rows)
}
func scanEquipeRow(row *sql.Row) (*domain.Equipe, error) {
	return scanEquipeFields(row)
}
func scanEquipeFields(s scanRow) (*domain.Equipe, error) {
	e := &domain.Equipe{}
	var projetID uuid.NullUUID
	var createdAt sql.NullTime
	err := s.Scan(&e.ID, &e.Nom, &projetID, &createdAt)
	if err != nil {
		return nil, err
	}
	if projetID.Valid {
		e.ProjetID = &projetID.UUID
	}
	if createdAt.Valid {
		t := createdAt.Time.UTC()
		e.CreatedAt = &t
	}
	return e, nil
}
