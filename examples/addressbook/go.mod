module github.com/mobentum/xdb/examples/addressbook

go 1.26.5

require (
	github.com/mobentum/kern v0.1.3
	github.com/mobentum/xdb v0.1.0
)

require (
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/mattn/go-sqlite3 v1.14.22
)

require github.com/jmoiron/sqlx v1.4.0 // indirect

replace github.com/mobentum/xdb => ../../
