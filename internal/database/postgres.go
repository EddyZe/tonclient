package database

import (
	"fmt"
	"tonclient/internal/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var log = config.InitLogger()

type Postgres struct {
	Db *sqlx.DB
}

func NewPostgres(config *config.PostgresConfig) (*Postgres, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&client_encoding=%s",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.DBName,
		"UTF8",
	)

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database")
	}

	return &Postgres{
		Db: db,
	}, nil
}

//	func NewPostgres(config *config.PostgresConfig) (*Postgres, error) {
//		connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
//			config.User,
//			config.Password,
//			config.Host,
//			config.Port,
//			config.DBName,
//		)
//		db, err := sql.Open("postgres", connStr)
//		if err != nil {
//			log.Error("Error opening database: ", err)
//			return nil, err
//		}
//		return &Postgres{
//			db: db,
//		}, nil
//
// }
func (p *Postgres) Close() error {
	err := p.Db.Close()
	if err != nil {
		log.Error("Error closing database: ", err)
		return err
	}

	return nil
}

func (p *Postgres) Ping() error {
	return p.Db.Ping()
}

//
//func (p *Postgres) Query(query string, args ...interface{}) (*sql.Rows, error) {
//	return p.db.Query(query, args...)
//}
//
//func (p *Postgres) QueryRow(query string, args ...interface{}) *sql.Row {
//	return p.db.QueryRow(query, args...)
//}
//
//func (p *Postgres) Exec(query string, args ...interface{}) (sql.Result, error) {
//	return p.db.Exec(query, args...)
//}
//
//func (p *Postgres) Prepare(query string) (*sql.Stmt, error) {
//	return p.db.Prepare(query)
//}
//
//func (p *Postgres) Begin() (*sql.Tx, error) {
//	return p.db.Begin()
//}
//
//func (p *Postgres) BeginTx() (*sql.Tx, error) {
//	return p.db.Begin()
//}
//
//func (p *Postgres) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
//	return p.db.ExecContext(ctx, query, args...)
//}
//
//func (p *Postgres) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
//	return p.db.PrepareContext(ctx, query)
//}
//
//func (p *Postgres) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
//	return p.db.QueryContext(ctx, query, args...)
//}
//
//func (p *Postgres) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
//	return p.db.QueryRowContext(ctx, query, args...)
//}
