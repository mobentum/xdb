package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/mobentum/xdb"
)

//go:embed migrations/*.sql
var migrations embed.FS

type User struct {
	ID    string  `db:"id"`
	Name  string  `db:"name"`
	Age   int     `db:"age"`
	Email *string `db:"email"`
}

func main() {
	ctx := context.Background()

	f := filepath.Join(os.TempDir(), "xdb_example.db")
	defer os.Remove(f)

	db, err := xdb.New(xdb.DBConfig{
		Driver: "sqlite3",
		DSN:    f,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// ── Migrations ──────────────────────────────────────

	fmt.Println("==> Running migrations...")
	if err := db.MigrateUp(migrations, "migrations"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("    ok")

	// ── Insert ──────────────────────────────────────────

	fmt.Println("\n==> Inserting users...")
	_, err = db.Insert("users").Columns("id", "name", "age").Values("u1", "Alice", 30).Exec(ctx)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Insert("users").Columns("id", "name", "age").Values("u2", "Bob", 25).Exec(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("    2 users inserted")

	// ── Select One ──────────────────────────────────────

	fmt.Println("\n==> SelectOne...")
	var alice User
	err = db.Select("id", "name", "age").From("users").
		Where(xdb.Cond.Eq("id", "u1")).
		One(ctx, &alice)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("    %+v\n", alice)

	// ── Select All ──────────────────────────────────────

	fmt.Println("\n==> SelectAll (ordered by age DESC)...")
	var users []User
	err = db.Select("id", "name", "age").From("users").
		OrderBy("age", xdb.DESC).
		All(ctx, &users)
	if err != nil {
		log.Fatal(err)
	}
	for _, u := range users {
		fmt.Printf("    %s — %s (%d)\n", u.ID, u.Name, u.Age)
	}

	// ── Count ───────────────────────────────────────────

	count, err := db.Select("*").From("users").Count(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n    Total users: %d\n", count)

	// ── Exists ──────────────────────────────────────────

	yes, err := db.Select("1").From("users").Where(xdb.Cond.Eq("id", "u1")).Exists(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("    User u1 exists: %t\n", yes)

	// ── Migration #2: add email column ──────────────────

	fmt.Println("\n==> Running migration 2 (add email)...")
	if err := db.MigrateUp(migrations, "migrations"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("    ok")

	// ── Update ──────────────────────────────────────────

	fmt.Println("\n==> Updating user email...")
	_, err = db.Update("users").
		Set("email", "alice@example.com").
		Where(xdb.Cond.Eq("id", "u1")).
		Exec(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("    ok")

	// ── Select with new column ──────────────────────────

	var updated User
	err = db.Select("id", "name", "age", "email").From("users").
		Where(xdb.Cond.Eq("id", "u1")).
		One(ctx, &updated)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("    %+v\n", updated)

	// ── Transaction ─────────────────────────────────────

	fmt.Println("\n==> Transaction (insert + rollback)...")
	err = db.Tx(ctx, func(tx *xdb.TxDB) error {
		_, err := tx.Insert("users").Columns("id", "name", "age").
			Values("tx1", "Charlie", 35).
			Exec(ctx)
		if err != nil {
			return err
		}
		return fmt.Errorf("force rollback")
	})
	fmt.Printf("    Expected error: %v\n", err)

	// Verify rollback
	charlieCount, err := db.Select("1").From("users").Where(xdb.Cond.Eq("id", "tx1")).Count(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("    Charlie visible after rollback: %d (expected 0)\n", charlieCount)

	// ── Pagination ──────────────────────────────────────

	fmt.Println("\n==> Pagination (page 1, size 2)...")
	pageResult, err := xdb.Paginate[User](ctx,
		db.Select("id", "name", "age").From("users").OrderBy("name", xdb.ASC),
		xdb.Page{Number: 1, Size: 2},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("    Page: %d, PageSize: %d, Total: %d, TotalPages: %d\n",
		pageResult.Page, pageResult.PageSize, pageResult.Total, pageResult.TotalPages)
	for _, u := range pageResult.Items {
		fmt.Printf("    %s — %s (%d)\n", u.ID, u.Name, u.Age)
	}

	// ── Streaming (Each) ────────────────────────────────

	fmt.Println("\n==> Streaming all users via Each...")
	err = db.Select("id", "name", "age").From("users").OrderBy("name", xdb.ASC).
		Each(ctx, func(rows *xdb.Rows) error {
			var u User
			if err := rows.StructScan(&u); err != nil {
				return err
			}
			fmt.Printf("    %s — %s (%d)\n", u.ID, u.Name, u.Age)
			return nil
		})
	if err != nil {
		log.Fatal(err)
	}

	// ── Raw queries ─────────────────────────────────────

	fmt.Println("\n==> Raw query...")
	var total int
	err = db.RawOne(ctx, &total, "SELECT COUNT(*) FROM users")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("    Total users (raw): %d\n", total)

	// ── Tear down: roll back all migrations ─────────────

	fmt.Println("\n==> Rolling back migrations...")
	if err := db.MigrateDown(migrations, "migrations"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("    ok")
}
