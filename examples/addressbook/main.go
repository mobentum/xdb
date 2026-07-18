package main

import (
	"context"
	"embed"
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/mobentum/kern"
	"github.com/mobentum/xdb"

	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Contact struct {
	ID        int    `db:"id"`
	Name      string `db:"name"`
	Email     string `db:"email"`
	Phone     string `db:"phone"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}

func main() {
	ctx := context.Background()

	dbPath := "addressbook.db"
	if v := os.Getenv("DB_PATH"); v != "" {
		dbPath = v
	}

	db, err := xdb.New(xdb.DBConfig{
		Driver: "sqlite3",
		DSN:    dbPath,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		db.RawExec(ctx, "PRAGMA journal_mode=WAL")
	}

	if err := db.MigrateUp(migrations, "migrations"); err != nil {
		log.Fatal(err)
	}

	app := kern.New(kern.WithSlogLogger(slog.Default()))

	api := app.Group("/api", kern.Logger())
	api.GET("/contacts", handleList(db))
	api.GET("/contacts/{id}", handleGet(db))
	api.POST("/contacts", handleCreate(db))
	api.PUT("/contacts/{id}", handleUpdate(db))
	api.DELETE("/contacts/{id}", handleDelete(db))

	addr := ":8080"
	log.Printf("Listening on %s", addr)
	if err := app.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func handleList(db *xdb.DB) kern.HandlerFunc {
	return func(c *kern.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
		q := c.DefaultQuery("q", "")

		b := db.Select("id", "name", "email", "phone", "created_at", "updated_at").
			From("contacts")

		if q != "" {
			b = b.Where(xdb.Cond.Like("name", "%"+q+"%"))
		}

		b = b.OrderBy("name", xdb.ASC)

		result, err := xdb.PaginateWithCount[Contact](c.Context(), b, xdb.Page{Number: page, Size: size})
		if err != nil {
			c.JSON(500, map[string]string{"error": err.Error()})
			return
		}

		c.JSON(200, map[string]any{
			"data":        result.Items,
			"total":       result.Total,
			"page":        result.Page,
			"size":        result.PageSize,
			"total_pages": result.TotalPages,
		})
	}
}

func handleGet(db *xdb.DB) kern.HandlerFunc {
	return func(c *kern.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, map[string]string{"error": "invalid id"})
			return
		}

		var contact Contact
		err = db.Select("id", "name", "email", "phone", "created_at", "updated_at").
			From("contacts").
			Where(xdb.Cond.Eq("id", id)).
			One(c.Context(), &contact)
		if err != nil {
			c.JSON(404, map[string]string{"error": "not found"})
			return
		}

		c.JSON(200, contact)
	}
}

func handleCreate(db *xdb.DB) kern.HandlerFunc {
	return func(c *kern.Context) {
		var input struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Phone string `json:"phone"`
		}
		if err := c.DecodeJSON(&input); err != nil {
			c.JSON(400, map[string]string{"error": "invalid json"})
			return
		}
		if input.Name == "" {
			c.JSON(422, map[string]string{"error": "name is required"})
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		_, err := db.Insert("contacts").
			Columns("name", "email", "phone", "created_at", "updated_at").
			Values(input.Name, input.Email, input.Phone, now, now).
			Exec(c.Context())
		if err != nil {
			c.JSON(500, map[string]string{"error": err.Error()})
			return
		}

		c.JSON(201, map[string]string{"status": "created"})
	}
}

func handleUpdate(db *xdb.DB) kern.HandlerFunc {
	return func(c *kern.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, map[string]string{"error": "invalid id"})
			return
		}

		var input struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Phone string `json:"phone"`
		}
		if err := c.DecodeJSON(&input); err != nil {
			c.JSON(400, map[string]string{"error": "invalid json"})
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		err = db.Update("contacts").
			Set("name", input.Name).
			Set("email", input.Email).
			Set("phone", input.Phone).
			Set("updated_at", now).
			Where(xdb.Cond.Eq("id", id)).
			ExecMustAffect(c.Context())
		if err != nil {
			c.JSON(404, map[string]string{"error": "not found"})
			return
		}

		c.JSON(200, map[string]string{"status": "updated"})
	}
}

func handleDelete(db *xdb.DB) kern.HandlerFunc {
	return func(c *kern.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(400, map[string]string{"error": "invalid id"})
			return
		}

		err = db.Delete("contacts").
			Where(xdb.Cond.Eq("id", id)).
			ExecMustAffect(c.Context())
		if err != nil {
			c.JSON(404, map[string]string{"error": "not found"})
			return
		}

		c.JSON(200, map[string]string{"status": "deleted"})
	}
}
