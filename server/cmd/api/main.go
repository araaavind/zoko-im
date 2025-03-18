package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/araaavind/zoko-im/internal/data"
	"github.com/araaavind/zoko-im/internal/queue"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

type config struct {
	port    int
	env     string
	limiter struct {
		rps     int
		burst   int
		enabled bool
	}
	cors struct {
		trustedOrigins []string
	}
	db struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
	redis struct {
		addr     string
		password string
		db       int
		stream   struct {
			key string
		}
	}
}

type application struct {
	config config
	logger *slog.Logger
	models data.Models
	redis  *redis.Client
	queue  *queue.MessageQueue
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	// Limiter configuration
	flag.IntVar(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	// CORS configuration
	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	// Database configuration
	flag.StringVar(&cfg.db.dsn, "dsn", os.Getenv("IM_DB_DSN"), "PostgreSQL connection string")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 1*time.Minute, "PostgreSQL max idle time")

	// Redis configuration
	flag.StringVar(&cfg.redis.addr, "redis-addr", "localhost:6379", "Redis server address")
	flag.StringVar(&cfg.redis.password, "redis-password", "", "Redis password")
	flag.IntVar(&cfg.redis.db, "redis-db", 0, "Redis database number")

	// Redis stream configuration
	flag.StringVar(&cfg.redis.stream.key, "redis-stream-key", "messages_stream", "Redis stream key name")

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	// Initialize Redis client
	rdb, err := initRedis(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("redis connection established")

	models := data.NewModels(db)

	// Initialize message queue
	messageQueue := queue.NewMessageQueue(
		rdb,
		queue.Config{
			StreamKey: cfg.redis.stream.key,
		},
		logger,
		models,
	)

	app := &application{
		config: cfg,
		logger: logger,
		models: models,
		redis:  rdb,
		queue:  messageQueue,
	}

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func initRedis(cfg config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.redis.addr,
		Password: cfg.redis.password,
		DB:       cfg.redis.db,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return rdb, nil
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test database connection
	err = db.PingContext(ctx)
	if err != nil {
		defer db.Close()
		return nil, err
	}

	return db, nil
}

// func openDBwithPGX(cfg config) (*pgxpool.Pool, error) {
// 	poolConfig, err := pgxpool.ParseConfig(cfg.db.dsn)
// 	if err != nil {
// 		return nil, err
// 	}

// 	poolConfig.MaxConns = int32(cfg.db.maxOpenConns)
// 	poolConfig.MaxConnLifetime = cfg.db.maxIdleTime

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = db.Ping(ctx)
// 	if err != nil {
// 		defer db.Close()
// 		return nil, err
// 	}

// 	return db, nil
// }
