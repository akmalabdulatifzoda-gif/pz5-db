package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Task — модель для сканирования результатов SELECT
type Task struct {
	ID        int
	Title     string
	Done      bool
	CreatedAt time.Time
}

// ErrNotFound возвращается, если задача с указанным id не найдена.
var ErrNotFound = errors.New("task not found")

type Repo struct {
	DB *sql.DB
}

func NewRepo(db *sql.DB) *Repo { return &Repo{DB: db} }

// CreateTask — параметризованный INSERT с возвратом id
func (r *Repo) CreateTask(ctx context.Context, title string) (int, error) {
	var id int
	const q = `INSERT INTO tasks (title) VALUES ($1) RETURNING id;`
	err := r.DB.QueryRowContext(ctx, q, title).Scan(&id)
	return id, err
}

// ListTasks — базовый SELECT всех задач
func (r *Repo) ListTasks(ctx context.Context) ([]Task, error) {
	const q = `SELECT id, title, done, created_at FROM tasks ORDER BY id;`
	return r.queryTasks(ctx, q)
}

// ListDone — задание 1: только выполненные (done=true) или невыполненные (done=false) задачи.
func (r *Repo) ListDone(ctx context.Context, done bool) ([]Task, error) {
	const q = `SELECT id, title, done, created_at FROM tasks WHERE done = $1 ORDER BY id;`
	return r.queryTasks(ctx, q, done)
}

// FindByID — задание 2: поиск одной задачи по id.
func (r *Repo) FindByID(ctx context.Context, id int) (*Task, error) {
	const q = `SELECT id, title, done, created_at FROM tasks WHERE id = $1;`
	var t Task
	err := r.DB.QueryRowContext(ctx, q, id).Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateMany — задание 3: массовая вставка в одной транзакции.
// Либо вставляются все строки, либо (при любой ошибке) ни одной.
func (r *Repo) CreateMany(ctx context.Context, titles []string) (err error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// при ошибке — откат; после успешного Commit Rollback ничего не делает
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO tasks (title) VALUES ($1);`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, title := range titles {
		if title == "" {
			return fmt.Errorf("empty title: transaction rolled back")
		}
		if _, err = stmt.ExecContext(ctx, title); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetDone — вспомогательный UPDATE, чтобы было что показать в ListDone(true).
func (r *Repo) SetDone(ctx context.Context, id int, done bool) error {
	res, err := r.DB.ExecContext(ctx, `UPDATE tasks SET done = $1 WHERE id = $2;`, done, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Count — количество строк в таблице (для проверки отката транзакции).
func (r *Repo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks;`).Scan(&n)
	return n, err
}

func (r *Repo) queryTasks(ctx context.Context, q string, args ...any) ([]Task, error) {
	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
