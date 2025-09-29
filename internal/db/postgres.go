package db

import (
    "context"
    "log"

    "carousel/internal/cache"
    "github.com/jackc/pgx/v5"                  // [ADDED] Import pgx for pgx.Tx
    "github.com/jackc/pgx/v5/pgxpool"          // Ensure pgxpool is imported
)

// // Postgres manages database interactions
// type Postgres struct {
//     pool *pgxpool.Pool
//     cache *cache.Redis
// }

// // NewPostgres initializes a Postgres client
// func NewPostgres(dsn string, redisAddr string) (*Postgres, error){
//     pool, err := pgxpool.New(context.Background(), dsn)
//     if err != nil {
//         log.Fatalf("Failed to connect to Postgres: %v", err)
//     }
//     cache := cache.NewRedis(redisAddr)
// 	return &Postgres{pool: pool, cache: cache}, nil
// }

// Postgres manages database and cache connections
type Postgres struct {
    Pool  *pgxpool.Pool
    Cache *cache.Redis
}

// NewPostgres initializes Postgres and Redis
func NewPostgres(dsn, redisAddr string) (*Postgres, error) {
    pool, err := pgxpool.New(context.Background(), dsn)
    if err != nil {
        log.Fatalf("Failed to connect to Postgres: %v", err)
    }
    return &Postgres{
        Pool:  pool,
        Cache: cache.NewRedis(redisAddr),
    }, nil
}

// Begin starts a transaction
func (p *Postgres) Begin(ctx context.Context) (pgx.Tx, error) { // Uses pgx.Tx
    tx, err := p.Pool.Begin(ctx)
    if err != nil {
        return nil, err
    }
    return tx, nil
}