-- ============================================================
-- MCD_Schema.sql — Schéma complet Suivi Projets Étudiants
-- Projet : mxvdfhafwpjdogmtfipd (Supabase)
-- Généré le : 2026-06-10
-- ============================================================

-- ── Hiérarchie académique ─────────────────────────────────

CREATE TABLE institutions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nom         TEXT NOT NULL,
    code        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMP DEFAULT now()
);

CREATE TABLE departments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nom             TEXT NOT NULL,
    institution_id  UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    created_at      TIMESTAMP DEFAULT now()
);

CREATE TABLE programs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nom             TEXT NOT NULL,
    code            TEXT,
    department_id   UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    created_at      TIMESTAMP DEFAULT now()
);

CREATE TABLE cohorts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nom          TEXT NOT NULL,
    annee_debut  INTEGER NOT NULL,
    annee_fin    INTEGER NOT NULL,
    program_id   UUID NOT NULL REFERENCES programs(id) ON DELETE CASCADE,
    created_at   TIMESTAMP DEFAULT now()
);

CREATE TABLE academic_groups (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nom        TEXT NOT NULL,
    cohort_id  UUID NOT NULL REFERENCES cohorts(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT now()
);

-- ── Utilisateurs ──────────────────────────────────────────

-- profiles : synchronisé avec auth.users (Supabase Auth)
CREATE TABLE profiles (
    id                   UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    email                TEXT NOT NULL,
    nom                  TEXT NOT NULL,
    prenom               TEXT,
    role                 TEXT NOT NULL DEFAULT 'etudiant'
                             CHECK (role IN ('admin', 'encadrant', 'etudiant')),
    group_id             UUID REFERENCES academic_groups(id),
    must_change_password BOOLEAN NOT NULL DEFAULT false,
    is_active            BOOLEAN NOT NULL DEFAULT true,
    created_at           TIMESTAMP DEFAULT now(),
    updated_at           TIMESTAMP DEFAULT now()
);

-- teacher_programs : association encadrant ↔ programme
CREATE TABLE teacher_programs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    program_id UUID NOT NULL REFERENCES programs(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT now(),
    UNIQUE (user_id, program_id)
);

-- ── Projets & équipes ─────────────────────────────────────

CREATE TABLE projets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titre         TEXT NOT NULL,
    description   TEXT,
    statut        TEXT NOT NULL DEFAULT 'actif'
                      CHECK (statut IN ('actif', 'archive', 'termine')),
    date_debut    DATE,
    date_fin      DATE,
    enseignant_id UUID REFERENCES profiles(id) ON DELETE SET NULL,
    group_id      UUID REFERENCES academic_groups(id),
    created_at    TIMESTAMP DEFAULT now()
);

CREATE TABLE equipes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nom        TEXT NOT NULL,
    projet_id  UUID REFERENCES projets(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE membres (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    equipe_id   UUID REFERENCES equipes(id) ON DELETE CASCADE,
    user_id     UUID REFERENCES profiles(id) ON DELETE CASCADE,
    role_equipe TEXT DEFAULT 'membre',
    joined_at   TIMESTAMP DEFAULT now()
);

-- ── Kanban ────────────────────────────────────────────────

CREATE TABLE taches (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titre         TEXT NOT NULL,
    description   TEXT,
    statut        TEXT NOT NULL DEFAULT 'todo'
                      CHECK (statut IN ('todo', 'en_cours', 'termine')),
    priorite      TEXT CHECK (priorite IN ('basse', 'moyenne', 'haute')),
    position      INTEGER NOT NULL DEFAULT 0,
    date_echeance DATE,
    equipe_id     UUID REFERENCES equipes(id) ON DELETE CASCADE,
    assignee_id   UUID REFERENCES profiles(id) ON DELETE SET NULL,
    created_at    TIMESTAMP DEFAULT now()
);

CREATE TABLE tache_commentaires (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tache_id   UUID NOT NULL REFERENCES taches(id) ON DELETE CASCADE,
    auteur_id  UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    contenu    TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

-- ── Jalons & livrables ────────────────────────────────────

CREATE TABLE jalons (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titre       TEXT NOT NULL,
    description TEXT,
    date_limite DATE NOT NULL,
    projet_id   UUID REFERENCES projets(id) ON DELETE CASCADE,
    created_at  TIMESTAMP DEFAULT now()
);

CREATE TABLE livrables (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nom         TEXT NOT NULL,
    fichier_url TEXT NOT NULL,
    equipe_id   UUID REFERENCES equipes(id) ON DELETE CASCADE,
    jalon_id    UUID REFERENCES jalons(id) ON DELETE SET NULL,
    note        NUMERIC CHECK (note >= 0 AND note <= 20),
    commentaire TEXT,
    depose_par  UUID REFERENCES profiles(id) ON DELETE SET NULL,
    created_at  TIMESTAMP DEFAULT now()
);

-- ── Table legacy (v1, conservée pour compatibilité) ───────

CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL,
    nom        TEXT NOT NULL,
    prenom     TEXT,
    role       TEXT NOT NULL DEFAULT 'etudiant',
    created_at TIMESTAMP DEFAULT now()
);

-- ── Index principaux ──────────────────────────────────────

-- profiles
CREATE INDEX idx_profiles_role      ON profiles(role);
CREATE INDEX idx_profiles_is_active ON profiles(is_active);
CREATE INDEX idx_profiles_group_id  ON profiles(group_id);

-- projets
CREATE INDEX idx_projets_statut       ON projets(statut);
CREATE INDEX idx_projets_enseignant_id ON projets(enseignant_id);
CREATE INDEX idx_projets_group_id     ON projets(group_id);

-- equipes
CREATE INDEX idx_equipes_projet_id ON equipes(projet_id);

-- membres
CREATE INDEX idx_membres_equipe_id ON membres(equipe_id);
CREATE INDEX idx_membres_user_id   ON membres(user_id);

-- taches
CREATE INDEX idx_taches_equipe_id   ON taches(equipe_id);
CREATE INDEX idx_taches_statut      ON taches(statut);
CREATE INDEX idx_taches_assignee_id ON taches(assignee_id);

-- tache_commentaires
CREATE INDEX idx_commentaires_tache  ON tache_commentaires(tache_id);
CREATE INDEX idx_commentaires_auteur ON tache_commentaires(auteur_id);

-- jalons
CREATE INDEX idx_jalons_projet_id ON jalons(projet_id);

-- livrables
CREATE INDEX idx_livrables_equipe_id  ON livrables(equipe_id);
CREATE INDEX idx_livrables_jalon_id   ON livrables(jalon_id);
CREATE INDEX idx_livrables_depose_par ON livrables(depose_par);

-- hiérarchie
CREATE INDEX idx_departments_institution ON departments(institution_id);
CREATE INDEX idx_programs_department     ON programs(department_id);
CREATE INDEX idx_cohorts_program         ON cohorts(program_id);
CREATE INDEX idx_academic_groups_cohort  ON academic_groups(cohort_id);

-- teacher_programs
CREATE INDEX idx_teacher_programs_user    ON teacher_programs(user_id);
CREATE INDEX idx_teacher_programs_program ON teacher_programs(program_id);

-- users (legacy)
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role  ON users(role);
