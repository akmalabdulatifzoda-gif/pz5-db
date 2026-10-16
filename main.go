package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	bench := flag.Bool("bench", false, "замер производительности при разных настройках пула")
	flag.Parse()

	// .env не обязателен; если файла нет — ошибка игнорируется
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// fallback — подключение через unix-сокет с peer-аутентификацией:
		// пароль не нужен, роль PostgreSQL = текущий пользователь Linux (как у psql без параметров)
		dsn = "postgres:///todo?host=/var/run/postgresql"
	}

	if *bench {
		runPoolBench(dsn)
		return
	}

	db, err := openDB(dsn)
	if err != nil {
		log.Fatalf("openDB error: %v", err)
	}
	defer db.Close()

	repo := NewRepo(db)

	// 1) Вставим пару задач
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	titles := []string{"Сделать ПЗ №5", "Купить кофе", "Проверить отчёты"}
	var lastID int
	for _, title := range titles {
		id, err := repo.CreateTask(ctx, title)
		if err != nil {
			log.Fatalf("CreateTask error: %v", err)
		}
		lastID = id
		log.Printf("Inserted task id=%d (%s)", id, title)
	}

	// 2) Прочитаем список задач
	ctxList, cancelList := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelList()

	tasks, err := repo.ListTasks(ctxList)
	if err != nil {
		log.Fatalf("ListTasks error: %v", err)
	}

	// 3) Напечатаем
	fmt.Println("=== Tasks ===")
	printTasks(tasks)

	// ---------- Проверочные задания ----------

	// Задание 3: массовая вставка через транзакцию
	ctxTx, cancelTx := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelTx()
	batch := []string{"Прочитать про транзакции", "Настроить пул соединений", "Залить код в Git"}
	if err := repo.CreateMany(ctxTx, batch); err != nil {
		log.Fatalf("CreateMany error: %v", err)
	}
	log.Printf("CreateMany: committed %d tasks in one transaction", len(batch))

	// Демонстрация отката: вторая строка пустая -> вся пачка откатывается
	before, _ := repo.Count(ctxTx)
	err = repo.CreateMany(ctxTx, []string{"Эта строка не сохранится", ""})
	after, _ := repo.Count(ctxTx)
	log.Printf("CreateMany with bad data: err=%q, rows before=%d, after=%d (rollback OK)", err, before, after)

	// Отметим последнюю задачу из первой пачки как выполненную — для ListDone(true)
	ctxUpd, cancelUpd := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelUpd()
	if err := repo.SetDone(ctxUpd, lastID, true); err != nil {
		log.Fatalf("SetDone error: %v", err)
	}

	// Задание 1: фильтр по done
	ctxDone, cancelDone := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelDone()
	doneTasks, err := repo.ListDone(ctxDone, true)
	if err != nil {
		log.Fatalf("ListDone(true) error: %v", err)
	}
	fmt.Println("=== Done tasks (done=true) ===")
	printTasks(doneTasks)

	notDone, err := repo.ListDone(ctxDone, false)
	if err != nil {
		log.Fatalf("ListDone(false) error: %v", err)
	}
	fmt.Println("=== Not done tasks (done=false) ===")
	printTasks(notDone)

	// Задание 2: поиск по id
	ctxFind, cancelFind := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFind()
	for _, id := range []int{lastID, 9999} {
		t, err := repo.FindByID(ctxFind, id)
		switch {
		case errors.Is(err, ErrNotFound):
			fmt.Printf("=== FindByID(%d) === not found\n", id)
		case err != nil:
			log.Fatalf("FindByID error: %v", err)
		default:
			fmt.Printf("=== FindByID(%d) ===\n  ID:        %d\n  Title:     %s\n  Done:      %v\n  CreatedAt: %s\n",
				id, t.ID, t.Title, t.Done, t.CreatedAt.Format(time.RFC3339))
		}
	}
}

func printTasks(tasks []Task) {
	for _, t := range tasks {
		fmt.Printf("#%d | %-24s | done=%-5v | %s\n",
			t.ID, t.Title, t.Done, t.CreatedAt.Format(time.RFC3339))
	}
}
