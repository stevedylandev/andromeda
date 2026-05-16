package main

import (
	"database/sql"
	"errors"
	"strings"
)

type Category struct {
	ID        int64
	Name      string
	CreatedAt string
}

func listCategories(db *sql.DB) ([]Category, error) {
	rows, err := db.Query(`SELECT id, name, created_at FROM categories ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func getOrCreateCategory(db *sql.DB, name string) (*Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	var c Category
	err := db.QueryRow(`SELECT id, name, created_at FROM categories WHERE name = ?`, name).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err == nil {
		return &c, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	res, err := db.Exec(`INSERT INTO categories (name) VALUES (?)`, name)
	if err != nil {
		var existing Category
		if err2 := db.QueryRow(`SELECT id, name, created_at FROM categories WHERE name = ?`, name).Scan(&existing.ID, &existing.Name, &existing.CreatedAt); err2 == nil {
			return &existing, nil
		}
		return nil, err
	}
	id, _ := res.LastInsertId()
	return getCategory(db, id)
}

func getCategory(db *sql.DB, id int64) (*Category, error) {
	var c Category
	err := db.QueryRow(`SELECT id, name, created_at FROM categories WHERE id = ?`, id).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func deleteCategory(db *sql.DB, id int64) (bool, error) {
	res, err := db.Exec(`DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
