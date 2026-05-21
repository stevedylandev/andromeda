package main

import (
	"database/sql"

	"github.com/stevedylandev/andromeda/apps/jotts/internal/store"
)

func openDB(path string) (*sql.DB, error) { return store.Open(path) }

func createNote(db *sql.DB, title, content string) (*Note, error) {
	return store.Create(db, title, content)
}

func getNoteByShortID(db *sql.DB, shortID string) (*Note, error) {
	return store.GetByShortID(db, shortID)
}

func listNotes(db *sql.DB) ([]Note, error) { return store.List(db) }

func updateNoteByShortID(db *sql.DB, shortID, title, content string) (*Note, error) {
	return store.UpdateByShortID(db, shortID, title, content)
}

func deleteNoteByShortID(db *sql.DB, shortID string) (bool, error) {
	return store.DeleteByShortID(db, shortID)
}
