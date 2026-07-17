module github.com/mobentum/xdb/examples/basic

go 1.26.5

require (
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/mobentum/xdb v0.0.0
)

require (
	github.com/jmoiron/sqlx v1.4.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
)

replace github.com/mobentum/xdb => ../../
