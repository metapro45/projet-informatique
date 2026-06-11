package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"suivi-projets-backend/internal/domain"
)

type JalonRepository struct{ db *sql.DB }

func NewJalonRepository(db *sql.DB) *JalonRepository {
	return &JalonRepository{db: db}
}

func (r *JalonRepository) GetByProjetID(ctx context.Context, projetID uuid.UUID) ([]*domain.Jalon, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, titre, description, date_limite, projet_id, created_at
		FROM jalons WHERE projet_id = $1
		ORDER BY date_limite ASC
	`, projetID)
	if err != nil {
		return nil, fmt.Errorf("jalon.GetByProjetID: %w", err)
	}
	defer rows.Close()
	var jalons []*domain.Jalon
	for rows.Next() {
		j, err := scanJalon(rows)
		if err != nil {
			return nil, err
		}
		jalons = append(jalons, j)
	}
	return jalons, nil
}

func (r *JalonRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Jalon, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, titre, description, date_limite, projet_id, created_at
		FROM jalons WHERE id = $1
	`, id)
	j, err := scanJalonRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("jalon.GetByID: %w", err)
	}
	return j, nil
}

func (r *JalonRepository) Create(ctx context.Context, j *domain.Jalon) error {
	j.ID = uuid.New()
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO jalons (id, titre, description, date_limite, projet_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, titre, description, date_limite, projet_id, created_at
	`, j.ID, j.Titre, j.Description, j.DateLimite, j.ProjetID)
	created, err := scanJalonRow(row)
	if err != nil {
		return fmt.Errorf("jalon.Create: %w", err)
	}
	*j = *created
	return nil
}

func (r *JalonRepository) Update(ctx context.Context, j *domain.Jalon) error {
	row := r.db.QueryRowContext(ctx, `
		UPDATE jalons SET titre=$2, description=$3, date_limite=$4
		WHERE id=$1
		RETURNING id, titre, description, date_limite, projet_id, created_at
	`, j.ID, j.Titre, j.Description, j.DateLimite)
	updated, err := scanJalonRow(row)
	if err == sql.ErrNoRows {
		return fmt.Errorf("jalon.Update: jalon %s introuvable", j.ID)
	}
	if err != nil {
		return fmt.Errorf("jalon.Update: %w", err)
	}
	*j = *updated
	return nil
}

func (r *JalonRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		WITH del_livrables AS (
			DELETE FROM livrables WHERE jalon_id = $1
		)
		DELETE FROM jalons WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("jalon.Delete: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("jalon.Delete: jalon %s introuvable", id)
	}
	return nil
}

func scanJalon(rows *sql.Rows) (*domain.Jalon, error)  { return scanJalonFields(rows) }
func scanJalonRow(row *sql.Row) (*domain.Jalon, error)  { return scanJalonFields(row) }

func scanJalonFields(s scanRow) (*domain.Jalon, error) {
	j := &domain.Jalon{}
	var description sql.NullString
	var dateLimite sql.NullTime
	var projetID uuid.NullUUID
	var createdAt sql.NullTime

	err := s.Scan(&j.ID, &j.Titre, &description, &dateLimite, &projetID, &createdAt)
	if err != nil {
		return nil, err
	}
	if description.Valid { j.Description = &description.String }
	if dateLimite.Valid {
		j.DateLimite = domain.Date{Time: dateLimite.Time}
	}
	if projetID.Valid    { j.ProjetID = &projetID.UUID }
	if createdAt.Valid   { t := createdAt.Time.UTC(); j.CreatedAt = &t }
	return j, nil
}
