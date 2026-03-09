package db

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/johncave/terraform-the-game/models"
)

// migrations holds the embedded SQL migration files.
//
//go:embed migrations/001_initial.sql
var migrationSQL string

type DB struct {
	conn *sql.DB
}

func New() (*DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := getEnv("POSTGRES_HOST", "localhost")
		port := getEnv("POSTGRES_PORT", "5432")
		user := getEnv("POSTGRES_USER", "terraform")
		pass := getEnv("POSTGRES_PASSWORD", "terraform")
		dbname := getEnv("POSTGRES_DB", "terraform_game")
	sslMode := getEnv("POSTGRES_SSLMODE", "disable")
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, pass, dbname, sslMode)
	}
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return &DB{conn: conn}, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (d *DB) RunMigrations() error {
	_, err := d.conn.Exec(migrationSQL)
	return err
}

func (d *DB) CreateGame(userID string) (*models.GameState, error) {
	gameID := uuid.New()
	_, err := d.conn.Exec(`INSERT INTO games (game_id, user_id) VALUES ($1, $2)`, gameID, userID)
	if err != nil {
		return nil, err
	}
	state := models.NewGameState(gameID)
	if err := d.SaveSnapshot(state); err != nil {
		return nil, err
	}
	return state, nil
}

func (d *DB) GetGame(gameID uuid.UUID) (*models.GameState, error) {
	var count int
	err := d.conn.QueryRow(`SELECT COUNT(*) FROM games WHERE game_id = $1`, gameID).Scan(&count)
	if err != nil || count == 0 {
		return nil, fmt.Errorf("game not found")
	}
	return d.LoadState(gameID)
}

func (d *DB) SaveSnapshot(state *models.GameState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(
		`INSERT INTO snapshots (game_id, state_json) VALUES ($1, $2)`,
		state.GameID, data,
	)
	return err
}

func (d *DB) LoadState(gameID uuid.UUID) (*models.GameState, error) {
	var stateJSON []byte
	var snapshotTime time.Time
	err := d.conn.QueryRow(
		`SELECT state_json, timestamp FROM snapshots WHERE game_id = $1 ORDER BY timestamp DESC LIMIT 1`,
		gameID,
	).Scan(&stateJSON, &snapshotTime)

	var state *models.GameState
	if err == sql.ErrNoRows {
		state = models.NewGameState(gameID)
		snapshotTime = time.Time{}
	} else if err != nil {
		return nil, err
	} else {
		state = &models.GameState{}
		if err := json.Unmarshal(stateJSON, state); err != nil {
			return nil, err
		}
	}

	rows, err := d.conn.Query(
		`SELECT event_type, payload_json FROM events WHERE game_id = $1 AND timestamp > $2 ORDER BY timestamp ASC`,
		gameID, snapshotTime,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var eventType string
		var payloadJSON []byte
		if err := rows.Scan(&eventType, &payloadJSON); err != nil {
			return nil, err
		}
		state.ApplyEvent(eventType, payloadJSON)
	}
	return state, nil
}

func (d *DB) AppendEvent(gameID uuid.UUID, eventType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(
		`INSERT INTO events (game_id, event_type, payload_json) VALUES ($1, $2, $3)`,
		gameID, eventType, data,
	)
	return err
}

func (d *DB) SaveFactory(gameID uuid.UUID, factory *models.Factory) error {
	data, err := json.Marshal(factory)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(
		`INSERT INTO factories (id, game_id, state_json) VALUES ($1, $2, $3)
         ON CONFLICT (id, game_id) DO UPDATE SET state_json = $3`,
		factory.ID, gameID, data,
	)
	return err
}

func (d *DB) GetFactory(gameID uuid.UUID, factoryID string) (*models.Factory, error) {
	var stateJSON []byte
	err := d.conn.QueryRow(
		`SELECT state_json FROM factories WHERE id = $1 AND game_id = $2`,
		factoryID, gameID,
	).Scan(&stateJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f models.Factory
	if err := json.Unmarshal(stateJSON, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (d *DB) ListFactories(gameID uuid.UUID) ([]*models.Factory, error) {
	rows, err := d.conn.Query(
		`SELECT state_json FROM factories WHERE game_id = $1`,
		gameID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var factories []*models.Factory
	for rows.Next() {
		var stateJSON []byte
		if err := rows.Scan(&stateJSON); err != nil {
			return nil, err
		}
		var f models.Factory
		if err := json.Unmarshal(stateJSON, &f); err != nil {
			return nil, err
		}
		factories = append(factories, &f)
	}
	return factories, nil
}

func (d *DB) DeleteFactory(gameID uuid.UUID, factoryID string) error {
	_, err := d.conn.Exec(
		`DELETE FROM factories WHERE id = $1 AND game_id = $2`,
		factoryID, gameID,
	)
	return err
}

func (d *DB) ListGames() ([]uuid.UUID, error) {
	rows, err := d.conn.Query(`SELECT game_id FROM games ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
