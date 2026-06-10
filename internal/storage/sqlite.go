package storage

import (
	_ "modernc.org/sqlite"

	"github.com/jmoiron/sqlx"
	ollama "github.com/ollama/ollama/api"
)

type DB struct {
	db *sqlx.DB
}

func NewDB(path string) (*DB, error) {
	db, err := sqlx.Connect("sqlite", path)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		in_context INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS dialog_summaries (
	    id INTEGER PRIMARY KEY AUTOINCREMENT,
    	chat_id INTEGER NOT NULL,
    	summary_text TEXT NOT NULL,
   		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	db.MustExec(schema)

	return &DB{db: db}, nil
}

func (s *DB) SaveMessage(chatID int64, role string, content string) error {
	query := `INSERT INTO messages (chat_id, role, content) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, chatID, role, content)
	return err
}

type dbRow struct {
	Role    string `db:"role"`
	Content string `db:"content"`
}

func (s *DB) GetHistory(chatID int64) ([]ollama.Message, error) {
	var rows []dbRow

	query := `SELECT role, content FROM messages WHERE chat_id = ? ORDER BY id ASC`
	err := s.db.Select(&rows, query, chatID)
	if err != nil {
		return nil, err
	}

	history := make([]ollama.Message, len(rows))
	for i, row := range rows {
		history[i] = ollama.Message{
			Role:    row.Role,
			Content: row.Content,
		}
	}

	return history, nil
}

func (s *DB) GetActiveChats() ([]int64, error) {
	var chats []int64
	query := `SELECT DISTINCT chat_id FROM messages`

	err := s.db.Select(&chats, query)
	if err != nil {
		return nil, err
	}
	return chats, nil
}

func (s *DB) GetActiveHistory(chatID int64) ([]ollama.Message, error) {
	var rows []dbRow
	query := `SELECT role, content FROM messages WHERE chat_id = ? AND in_context = ? ORDER BY id ASC`
	err := s.db.Select(&rows, query, chatID, 1)
	if err != nil {
		return nil, err
	}

	history := make([]ollama.Message, len(rows))
	for i, row := range rows {
		history[i] = ollama.Message{
			Role:    row.Role,
			Content: row.Content,
		}
	}

	return history, nil
}

func (s *DB) GetLatestSummary(chatID int64) (string, error) {
	var summary string
	query := `SELECT summary_text FROM dialog_summaries WHERE chat_id = ? ORDER BY id DESC LIMIT 1`
	err := s.db.Get(&summary, query, chatID)
	if err != nil {
		return "", err
	}
	return summary, nil
}

func (s *DB) ArchiveOldMessages(chatID int64, limit int) error {
	query := `
		UPDATE messages 
		SET in_context = 0 
		WHERE id IN (
			SELECT id FROM messages 
			WHERE chat_id = ? AND in_context = 1 
			ORDER BY id ASC 
			LIMIT ?
		)`
	_, err := s.db.Exec(query, chatID, limit)
	return err
}

func (s *DB) SaveSummary(chatID int64, text string) error {
	query := `INSERT INTO dialog_summaries (chat_id, summary_text) VALUES (?, ?)`
	_, err := s.db.Exec(query, chatID, text)
	return err
}
