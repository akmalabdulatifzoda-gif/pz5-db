package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// runPoolBench — задание 4: замер пропускной способности при разных настройках пула.
// Запуск: go run . -bench
// Нагрузка: 50 горутин, 3 секунды, в каждой итерации — SELECT по id (короткий запрос).
func runPoolBench(dsn string) {
	configs := []struct {
		name string
		p    PoolConfig
	}{
		{"open=1  idle=1", PoolConfig{1, 1, 30 * time.Minute, 5 * time.Minute}},
		{"open=5  idle=2", PoolConfig{5, 2, 30 * time.Minute, 5 * time.Minute}},
		{"open=10 idle=5", PoolConfig{10, 5, 30 * time.Minute, 5 * time.Minute}},
		{"open=20 idle=10", PoolConfig{20, 10, 30 * time.Minute, 5 * time.Minute}},
		{"open=50 idle=25", PoolConfig{50, 25, 30 * time.Minute, 5 * time.Minute}},
		{"open=10 idle=0", PoolConfig{10, 0, 30 * time.Minute, 5 * time.Minute}},
	}

	const workers = 50
	const duration = 3 * time.Second

	fmt.Printf("CPU cores: %d, workers: %d, duration: %s\n", runtime.NumCPU(), workers, duration)
	fmt.Printf("%-16s | %9s | %8s | %8s | %9s | %10s\n", "pool", "queries", "qps", "avg, ms", "wait cnt", "idleClosed")
	fmt.Println("-----------------+-----------+----------+----------+-----------+-----------")

	for _, c := range configs {
		db, err := openDBWithPool(dsn, c.p)
		if err != nil {
			log.Fatalf("openDB: %v", err)
		}
		repo := NewRepo(db)

		var count, totalNs int64
		ctx, cancel := context.WithTimeout(context.Background(), duration)
		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for ctx.Err() == nil {
					start := time.Now()
					qctx, qcancel := context.WithTimeout(ctx, time.Second)
					_, err := repo.FindByID(qctx, 1)
					qcancel()
					if err != nil && ctx.Err() == nil {
						continue
					}
					if ctx.Err() == nil {
						atomic.AddInt64(&count, 1)
						atomic.AddInt64(&totalNs, int64(time.Since(start)))
					}
				}
			}()
		}
		wg.Wait()
		cancel()

		st := db.Stats()
		avg := float64(totalNs) / float64(count) / 1e6
		fmt.Printf("%-16s | %9d | %8.0f | %8.2f | %9d | %10d\n",
			c.name, count, float64(count)/duration.Seconds(), avg, st.WaitCount, st.MaxIdleClosed)
		db.Close()
	}
}
