package storage

import (
	"context"
	"database/sql"
	"fmt"
	"iter"
	"net/url"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/db"
	"github.com/calvinmclean/babyapi"
)

// NoteStorage implements babyapi.Storage interface for Notes using SQL
type NoteStorage struct {
	q *db.Queries
}

var _ babyapi.Storage[*pkg.Note] = &NoteStorage{}

// NewNoteStorage creates a new NoteStorage instance
func NewNoteStorage(sqlDB *sql.DB) *NoteStorage {
	return &NoteStorage{
		q: db.New(sqlDB),
	}
}

// Get retrieves a Note from storage by ID
func (s *NoteStorage) Get(ctx context.Context, id string) (*pkg.Note, error) {
	dbNote, err := s.q.GetNote(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, babyapi.ErrNotFound
		}
		return nil, fmt.Errorf("error getting note: %w", err)
	}

	return dbNoteToNote(dbNote)
}

// Search returns all Notes from storage
func (s *NoteStorage) Search(ctx context.Context, _ string, _ url.Values) iter.Seq2[*pkg.Note, error] {
	return func(yield func(*pkg.Note, error) bool) {
		dbNotes, err := s.q.ListNotes(ctx)
		if err != nil {
			yield(nil, fmt.Errorf("error listing notes: %w", err))
			return
		}

		for _, dbNote := range dbNotes {
			note, err := dbNoteToNote(dbNote)
			if err != nil {
				if !yield(nil, fmt.Errorf("invalid note: %w", err)) {
					return
				}
				continue
			}
			if !yield(note, nil) {
				return
			}
		}
	}
}

// Set saves a Note to storage (creates or updates)
func (s *NoteStorage) Set(ctx context.Context, note *pkg.Note) error {
	var content sql.NullString
	if note.Content != "" {
		content = sql.NullString{String: note.Content, Valid: true}
	}

	var gardenID sql.NullString
	if note.GardenID != nil {
		gardenID = sql.NullString{String: *note.GardenID, Valid: true}
	}

	var zoneID sql.NullString
	if note.ZoneID != nil {
		zoneID = sql.NullString{String: *note.ZoneID, Valid: true}
	}

	createdAt := time.Now().Format(time.RFC3339)
	if note.CreatedAt != nil {
		createdAt = note.CreatedAt.Format(time.RFC3339)
	}

	return s.q.UpsertNote(ctx, db.UpsertNoteParams{
		ID:        note.ID.String(),
		Title:     note.Title,
		Content:   content,
		CreatedAt: createdAt,
		GardenID:  gardenID,
		ZoneID:    zoneID,
	})
}

// Delete removes a Note from storage
func (s *NoteStorage) Delete(ctx context.Context, id string) error {
	return s.q.DeleteNote(ctx, id)
}

func dbNoteToNote(dbNote db.Note) (*pkg.Note, error) {
	noteID, err := parseID(dbNote.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid note ID: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, dbNote.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid created_at: %w", err)
	}

	note := &pkg.Note{
		ID:        noteID,
		Title:     dbNote.Title,
		CreatedAt: &createdAt,
	}

	if dbNote.Content.Valid {
		note.Content = dbNote.Content.String
	}

	if dbNote.GardenID.Valid {
		note.GardenID = &dbNote.GardenID.String
	}

	if dbNote.ZoneID.Valid {
		note.ZoneID = &dbNote.ZoneID.String
	}

	return note, nil
}
