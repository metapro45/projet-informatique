package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/google/uuid"
	"suivi-projets-backend/internal/domain"
)

type LivrableRepository struct{ db *sql.DB }

func NewLivrableRepository(db *sql.DB) *LivrableRepository {
	return &LivrableRepository{db: db}
}

const sqlGetLivrablesByJalon = `
	SELECT id, nom, fichier_url, equipe_id, jalon_id,
	       note, commentaire, depose_par, created_at
	FROM livrables WHERE jalon_id = $1
	ORDER BY created_at DESC
`
const sqlGetLivrablesByEquipe = `
	SELECT id, nom, fichier_url, equipe_id, jalon_id,
	       note, commentaire, depose_par, created_at
	FROM livrables WHERE equipe_id = $1
	ORDER BY created_at DESC
`
const sqlGetLivrableByID = `
	SELECT id, nom, fichier_url, equipe_id, jalon_id,
	       note, commentaire, depose_par, created_at
	FROM livrables WHERE id = $1
`
const sqlCreateLivrable = `
	INSERT INTO livrables (id, nom, fichier_url, equipe_id, jalon_id, depose_par)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, nom, fichier_url, equipe_id, jalon_id,
	          note, commentaire, depose_par, created_at
`
const sqlNoterLivrable = `
	UPDATE livrables SET note=$2, commentaire=$3 WHERE id=$1
	RETURNING id, nom, fichier_url, equipe_id, jalon_id,
	          note, commentaire, depose_par, created_at
`
const sqlDeleteLivrable = `DELETE FROM livrables WHERE id=$1`

func (r *LivrableRepository) GetByJalonID(ctx context.Context, jalonID uuid.UUID) ([]*domain.Livrable, error) {
	rows, err := r.db.QueryContext(ctx, sqlGetLivrablesByJalon, jalonID)
	if err != nil {
		return nil, fmt.Errorf("livrable.GetByJalonID: %w", err)
	}
	defer rows.Close()
	return scanLivrables(rows)
}

func (r *LivrableRepository) GetByEquipeID(ctx context.Context, equipeID uuid.UUID) ([]*domain.Livrable, error) {
	rows, err := r.db.QueryContext(ctx, sqlGetLivrablesByEquipe, equipeID)
	if err != nil {
		return nil, fmt.Errorf("livrable.GetByEquipeID: %w", err)
	}
	defer rows.Close()
	return scanLivrables(rows)
}

func (r *LivrableRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Livrable, error) {
	row := r.db.QueryRowContext(ctx, sqlGetLivrableByID, id)
	l, err := scanLivrableRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("livrable.GetByID: %w", err)
	}
	return l, nil
}

func (r *LivrableRepository) Create(ctx context.Context, l *domain.Livrable) error {
	l.ID = uuid.New()
	row := r.db.QueryRowContext(ctx, sqlCreateLivrable,
		l.ID, l.Nom, l.FichierURL, l.EquipeID, l.JalonID, l.DeposePar,
	)
	created, err := scanLivrableRow(row)
	if err != nil {
		return fmt.Errorf("livrable.Create: %w", err)
	}
	*l = *created
	return nil
}

func (r *LivrableRepository) Noter(ctx context.Context, id uuid.UUID, note float64, commentaire *string) (*domain.Livrable, error) {
	// Arrondir à 2 décimales
	note = math.Round(note*100) / 100
	row := r.db.QueryRowContext(ctx, sqlNoterLivrable, id, note, commentaire)
	l, err := scanLivrableRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("livrable.Noter: %w", err)
	}
	return l, nil
}

func (r *LivrableRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteLivrable, id)
	if err != nil {
		return fmt.Errorf("livrable.Delete: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("livrable.Delete: livrable %s introuvable", id)
	}
	return nil
}

func scanLivrables(rows *sql.Rows) ([]*domain.Livrable, error) {
	var livrables []*domain.Livrable
	for rows.Next() {
		l, err := scanLivrableFields(rows)
		if err != nil {
			return nil, err
		}
		livrables = append(livrables, l)
	}
	return livrables, nil
}

func scanLivrableRow(row *sql.Row) (*domain.Livrable, error) {
	return scanLivrableFields(row)
}

func scanLivrableFields(s scanRow) (*domain.Livrable, error) {
	l := &domain.Livrable{}
	var equipeID, jalonID, deposePar uuid.NullUUID
	var note    sql.NullFloat64
	var commentaire sql.NullString
	var createdAt   sql.NullTime

	err := s.Scan(
		&l.ID, &l.Nom, &l.FichierURL,
		&equipeID, &jalonID, &note, &commentaire, &deposePar, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	if equipeID.Valid  { l.EquipeID   = &equipeID.UUID }
	if jalonID.Valid   { l.JalonID    = &jalonID.UUID }
	if deposePar.Valid { l.DeposePar  = &deposePar.UUID }
	if note.Valid      { l.Note       = &note.Float64 }
	if commentaire.Valid { l.Commentaire = &commentaire.String }
	if createdAt.Valid   { t := createdAt.Time.UTC(); l.CreatedAt = &t }
	return l, nil
}
