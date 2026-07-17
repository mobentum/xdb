package xdb_test

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/mobentum/xdb"
)

func exampleDB() *xdb.DB {
	return xdb.Wrap(&sqlx.DB{})
}

// ─────────────────────────────────────────────────────────────
// Suite 1: Setup — connection configuration, SQL inspection
// ─────────────────────────────────────────────────────────────

// Example_new shows how to configure and open a database connection.
// Driver defaults to "postgres" when empty.
// Supported drivers: postgres, pgx, mysql, mariadb, sqlite3.
func Example_new() {
	fmt.Println(`db, err := xdb.New(xdb.DBConfig{
    DSN: "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
})`)
	// Output:
	// db, err := xdb.New(xdb.DBConfig{
	//     DSN: "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
	// })
}

// Example_to_sql_inspection shows how to inspect generated SQL and args
// without executing the query. Useful for debugging or testing.
func Example_to_sql_inspection() {
	db := exampleDB()
	sql, args, _ := db.Select("id").
		From("users").
		Where(xdb.Cond.Eq("id", 1)).
		ToSQL()
	fmt.Println("SQL: ", sql)
	fmt.Println("Args:", args)
	// Output:
	// SQL:  SELECT id FROM users WHERE id = $1
	// Args: [1]
}

// ─────────────────────────────────────────────────────────────
// Suite 2: Basic SELECT — One, All, Count, Exists
// ─────────────────────────────────────────────────────────────

// Example_select_basic shows SELECT with a WHERE clause and scanning
// a single row with One(). Returns ErrNotFound when no row matches.
func Example_select_basic() {
	db := exampleDB()
	sql, args, _ := db.Select("id", "name", "email").
		From("users").
		Where(xdb.Cond.Eq("id", "user-id-42")).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT id, name, email FROM users WHERE id = $1
	// [user-id-42]
}

// Example_select_all shows SELECT with ORDER BY and scanning multiple
// rows with All(). dest must be a pointer to a struct slice.
func Example_select_all() {
	db := exampleDB()
	sql, args, _ := db.Select("*").
		From("users").
		Where(xdb.Cond.Gt("age", 18)).
		OrderBy("name", xdb.ASC).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT * FROM users WHERE age > $1 ORDER BY name ASC
	// [18]
}

// Example_select_count shows COUNT(*) via the Count() helper.
// It clones the builder, replaces columns with COUNT(*),
// and removes ORDER BY, LIMIT, and OFFSET.
func Example_select_count() {
	db := exampleDB()
	sql, args, _ := db.Select("*").
		From("users").
		Where(xdb.Cond.Eq("active", true)).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT * FROM users WHERE active = $1
	// [true]
}

// Example_select_exists shows EXISTS via the Exists() helper.
// Wraps the inner query in SELECT EXISTS (...).
func Example_select_exists() {
	db := exampleDB()
	sql, args, _ := db.Select("1").
		From("users").
		Where(xdb.Cond.Eq("email", "alice@example.com")).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT 1 FROM users WHERE email = $1
	// [alice@example.com]
}

// ─────────────────────────────────────────────────────────────
// Suite 3: Basic INSERT — Exec, Returning, OnConflict, SetMap, multi-row
// ─────────────────────────────────────────────────────────────

// Example_insert_basic shows INSERT with explicit Columns and Values.
// Exec returns the number of rows affected.
func Example_insert_basic() {
	db := exampleDB()
	sql, args, _ := db.Insert("users").
		Columns("id", "name", "email").
		Values("u1", "Alice", "alice@example.com").
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// INSERT INTO users (id,name,email) VALUES ($1,$2,$3)
	// [u1 Alice alice@example.com]
}

// Example_insert_returning shows INSERT … RETURNING.
// One() scans the returned row into dest.
// Only supported on PostgreSQL (SupportsReturning dialect flag).
func Example_insert_returning() {
	db := exampleDB()
	sql, args, _ := db.Insert("users").
		Columns("id", "name", "email").
		Values("u1", "Alice", "alice@example.com").
		Returning("id", "name").
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// INSERT INTO users (id,name,email) VALUES ($1,$2,$3) RETURNING id, name
	// [u1 Alice alice@example.com]
}

// Example_insert_on_conflict shows INSERT … ON CONFLICT (PostgreSQL)
// or ON DUPLICATE KEY (MySQL/MariaDB) via the OnConflict helper.
func Example_insert_on_conflict() {
	db := exampleDB()
	sql, _, _ := db.Insert("users").
		Columns("id", "name").
		Values("u1", "Alice").
		OnConflict("(id) DO UPDATE SET name = EXCLUDED.name").
		ToSQL()
	fmt.Println(sql)
	// Output:
	// INSERT INTO users (id,name) VALUES ($1,$2) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name
}

// Example_insert_set_map shows INSERT from a map[column]value.
// Column order is non-deterministic; prefer Columns + Values
// when ordering matters.
func Example_insert_set_map() {
	db := exampleDB()
	sql, args, _ := db.Insert("users").
		SetMap(map[string]any{"id": "u1", "name": "Alice"}).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// INSERT INTO users (id,name) VALUES ($1,$2)
	// [u1 Alice]
}

// Example_insert_multi_row shows INSERT with multiple value rows.
// Each Values() call adds one row; they are comma-separated in SQL.
func Example_insert_multi_row() {
	db := exampleDB()
	sql, args, _ := db.Insert("users").
		Columns("id", "name").
		Values("u1", "Alice").
		Values("u2", "Bob").
		Values("u3", "Charlie").
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// INSERT INTO users (id,name) VALUES ($1,$2), ($3,$4), ($5,$6)
	// [u1 Alice u2 Bob u3 Charlie]
}

// ─────────────────────────────────────────────────────────────
// Suite 4: Basic UPDATE — Set, SetExpr, WhereIf, Returning, ExecMustAffect
// ─────────────────────────────────────────────────────────────

// Example_update_basic shows UPDATE with SET and WHERE.
// Exec returns rows affected. ExecMustAffect returns ErrNoRows
// when zero rows match.
func Example_update_basic() {
	db := exampleDB()
	sql, args, _ := db.Update("users").
		Set("name", "Bob").
		Where(xdb.Cond.Eq("id", "user-id")).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// UPDATE users SET name = $1 WHERE id = $2
	// [Bob user-id]
}

// Example_update_set_expr shows UPDATE with a raw SQL expression
// on the right-hand side of SET. The expression's ? placeholders
// are bound to the provided args.
func Example_update_set_expr() {
	db := exampleDB()
	sql, args, _ := db.Update("products").
		SetExpr("price", "price * ?", 1.10).
		Where(xdb.Cond.Eq("category", "electronics")).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// UPDATE products SET price = price * $1 WHERE category = $2
	// [1.1 electronics]
}

// Example_update_where_if shows conditional WHERE clauses.
// WhereIf only appends the predicate when cond is true —
// useful for building dynamic filters without if blocks.
func Example_update_where_if() {
	db := exampleDB()
	role := "admin"
	sql, args, _ := db.Update("users").
		Set("name", "Alice").
		Where(xdb.Cond.Eq("id", "user-id")).
		WhereIf(role != "", xdb.Cond.Eq("role", role)).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// UPDATE users SET name = $1 WHERE id = $2 AND role = $3
	// [Alice user-id admin]
}

// Example_update_returning shows UPDATE … RETURNING.
// One() scans the updated row into dest.
func Example_update_returning() {
	db := exampleDB()
	sql, args, _ := db.Update("users").
		Set("name", "Bob").
		Where(xdb.Cond.Eq("id", "user-id")).
		Returning("id", "name").
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// UPDATE users SET name = $1 WHERE id = $2 RETURNING id, name
	// [Bob user-id]
}

// Example_update_exec_must_affect shows the ExecMustAffect pattern.
// Returns ErrNoRows when the UPDATE matches zero rows.
func Example_update_exec_must_affect() {
	db := exampleDB()
	sql, args, _ := db.Update("users").
		Set("name", "Bob").
		Where(xdb.Cond.Eq("id", "user-id")).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// UPDATE users SET name = $1 WHERE id = $2
	// [Bob user-id]
}

// ─────────────────────────────────────────────────────────────
// Suite 5: Basic DELETE — Exec, Returning, ExecMustAffect
// ─────────────────────────────────────────────────────────────

// Example_delete_basic shows DELETE with a WHERE clause.
// Omitting Where deletes all rows (use with caution).
func Example_delete_basic() {
	db := exampleDB()
	sql, args, _ := db.Delete("sessions").
		Where(xdb.Cond.Lt("expires_at", "2024-01-01")).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// DELETE FROM sessions WHERE expires_at < $1
	// [2024-01-01]
}

// Example_delete_returning shows DELETE … RETURNING.
// One() scans the deleted row into dest.
func Example_delete_returning() {
	db := exampleDB()
	sql, args, _ := db.Delete("sessions").
		Where(xdb.Cond.Lt("expires_at", "2024-01-01")).
		Returning("id").
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// DELETE FROM sessions WHERE expires_at < $1 RETURNING id
	// [2024-01-01]
}

// Example_delete_exec_must_affect shows ExecMustAffect on DELETE.
// Returns ErrNoRows when zero rows are deleted.
func Example_delete_exec_must_affect() {
	db := exampleDB()
	sql, args, _ := db.Delete("users").
		Where(xdb.Cond.Eq("id", "user-id")).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// DELETE FROM users WHERE id = $1
	// [user-id]
}

// ─────────────────────────────────────────────────────────────
// Suite 6: Conditions — Eq, Gt, Like, In, Between, IsNull, Search
// ─────────────────────────────────────────────────────────────

// Example_conditions demonstrates the most common condition builders.
// Each condition implements the Predicate interface with ToSQL().
func Example_conditions() {
	sql, args, _ := xdb.Cond.Eq("col", 1).ToSQL()
	fmt.Println(sql, args)

	sql, args, _ = xdb.Cond.Gt("age", 18).ToSQL()
	fmt.Println(sql, args)

	sql, args, _ = xdb.Cond.Like("name", "%foo%").ToSQL()
	fmt.Println(sql, args)

	sql, args, _ = xdb.Cond.In("status", "active", "pending").ToSQL()
	fmt.Println(sql, args)

	sql, args, _ = xdb.Cond.Between("age", 18, 65).ToSQL()
	fmt.Println(sql, args)

	sql, _, _ = xdb.Cond.IsNull("deleted_at").ToSQL()
	fmt.Println(sql)
	// Output:
	// col = ? [1]
	// age > ? [18]
	// name LIKE ? [%foo%]
	// status IN (?, ?) [active pending]
	// age >= ? AND age <= ? [18 65]
	// deleted_at IS NULL
}

// Example_cond_search shows the Search condition.
// It builds (col1 ILIKE %term% OR col2 ILIKE %term% …)
// for case-insensitive full-text search across multiple columns.
func Example_cond_search() {
	sql, args, _ := xdb.Cond.Search("john", "name", "email").ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// (name ILIKE ? OR email ILIKE ?)
	// [%john% %john%]
}

// ─────────────────────────────────────────────────────────────
// Suite 7: Intermediate SELECT — GroupBy, Having, Pagination, AllowSort
// ─────────────────────────────────────────────────────────────

// Example_select_group_by_having shows GROUP BY with HAVING.
// HAVING filters after aggregation, unlike WHERE which
// filters before aggregation.
func Example_select_group_by_having() {
	db := exampleDB()
	sql, args, _ := db.Select("category", "COUNT(*) AS count", "SUM(amount) AS total").
		From("orders").
		Where(xdb.Cond.Gt("created_at", "2024-01-01")).
		GroupBy("category").
		Having(xdb.Cond.Gt("COUNT(*)", 5)).
		OrderBy("total", xdb.DESC).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT category, COUNT(*) AS count, SUM(amount) AS total FROM orders WHERE created_at > $1 GROUP BY category HAVING COUNT(*) > $2 ORDER BY total DESC
	// [2024-01-01 5]
}

// Example_pagination shows the generic Paginate[T] helper.
// It runs a COUNT(*) query and a LIMIT/OFFSET data query,
// returning a PageResult with total, items, and page metadata.
func Example_pagination() {
	db := exampleDB()
	sql, args, _ := db.Select("id", "title").
		From("posts").
		Where(xdb.Cond.Eq("published", true)).
		OrderBy("created_at", xdb.DESC).
		Paginate(xdb.Page{Number: 2, Size: 20}).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT id, title FROM posts WHERE published = $1 ORDER BY created_at DESC LIMIT 20 OFFSET 20
	// [true]
}

// Example_allow_sort shows the AllowSort whitelist for safe ORDER BY.
// Columns not in the whitelist are silently rejected, preventing
// SQL injection through user-controlled sort parameters.
func Example_allow_sort() {
	db := exampleDB()
	sql, _, _ := db.Select("id", "name", "email").
		From("users").
		AllowSort("name", "email", "created_at").
		OrderBy("name", xdb.ASC).        // allowed: in whitelist
		OrderBy("password", xdb.DESC).   // rejected: not in whitelist
		ToSQL()
	fmt.Println(sql)
	// Output:
	// SELECT id, name, email FROM users ORDER BY name ASC
}

// ─────────────────────────────────────────────────────────────
// Suite 8: Advanced JOINs — 9 relationship patterns
// ─────────────────────────────────────────────────────────────

// Example_join_1tom shows a 1:M relationship: one user has many
// orders, each order has many items. Uses JOIN + GROUP BY +
// aggregation + HAVING to filter aggregated results.
func Example_join_1tom() {
	db := exampleDB()
	sql, args, _ := db.Select(
		"o.id AS order_id",
		"u.name AS user_name",
		"COUNT(oi.id) AS item_count",
		"SUM(oi.quantity * oi.unit_price) AS total",
	).
		From("orders o").
		Join("users u ON u.id = o.user_id").
		Join("order_items oi ON oi.order_id = o.id").
		Where(xdb.Cond.Eq("o.status", "shipped")).
		GroupBy("o.id", "u.name").
		Having(xdb.Cond.Gt("SUM(oi.quantity * oi.unit_price)", 100)).
		OrderBy("total", xdb.DESC).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT o.id AS order_id, u.name AS user_name, COUNT(oi.id) AS item_count, SUM(oi.quantity * oi.unit_price) AS total FROM orders o JOIN users u ON u.id = o.user_id JOIN order_items oi ON oi.order_id = o.id WHERE o.status = $1 GROUP BY o.id, u.name HAVING SUM(oi.quantity * oi.unit_price) > $2 ORDER BY total DESC
	// [shipped 100]
}

// Example_join_mto1 shows a M:1 relationship: many posts belong to
// one author. Uses LEFT JOIN because the author may be deleted
// (soft-delete pattern). The author fields use *string to
// represent NULL when no author exists.
func Example_join_mto1() {
	db := exampleDB()
	sql, _, _ := db.Select(
		"p.id AS post_id",
		"p.title",
		"a.id AS author_id",
		"a.name AS author_name",
	).
		From("posts p").
		LeftJoin("authors a ON a.id = p.author_id").
		Where(xdb.Cond.Eq("p.published", true)).
		OrderBy("p.published_at", xdb.DESC).
		ToSQL()
	fmt.Println(sql)
	// Output:
	// SELECT p.id AS post_id, p.title, a.id AS author_id, a.name AS author_name FROM posts p LEFT JOIN authors a ON a.id = p.author_id WHERE p.published = $1 ORDER BY p.published_at DESC
}

// Example_join_self shows a self-join for hierarchical data:
// one manager has many employees. The same table is joined with
// different aliases (e, m).
func Example_join_self() {
	db := exampleDB()
	sql, _, _ := db.Select(
		"e.id AS employee_id",
		"e.name AS employee_name",
		"m.id AS manager_id",
		"m.name AS manager_name",
	).
		From("employees e").
		LeftJoin("employees m ON m.id = e.manager_id").
		OrderBy("e.name", xdb.ASC).
		ToSQL()
	fmt.Println(sql)
	// Output:
	// SELECT e.id AS employee_id, e.name AS employee_name, m.id AS manager_id, m.name AS manager_name FROM employees e LEFT JOIN employees m ON m.id = e.manager_id ORDER BY e.name ASC
}

// Example_join_lateral shows a LATERAL subquery for per-row
// computation: each category's recent transaction stats.
// LATERAL lets the subquery reference columns from the outer
// query (c.id), which a regular JOIN cannot do.
func Example_join_lateral() {
	db := exampleDB()
	sql, args, _ := db.Select("c.name AS category", "s.revenue", "s.count").
		From("categories c").
		LeftJoin("LATERAL (SELECT SUM(amount) AS revenue, COUNT(*) AS count FROM transactions t WHERE t.category_id = c.id AND t.status = 'completed') s ON true").
		Where(xdb.Cond.Gt("s.revenue", 0)).
		OrderBy("s.revenue", xdb.DESC).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT c.name AS category, s.revenue, s.count FROM categories c LEFT JOIN LATERAL (SELECT SUM(amount) AS revenue, COUNT(*) AS count FROM transactions t WHERE t.category_id = c.id AND t.status = 'completed') s ON true WHERE s.revenue > $1 ORDER BY s.revenue DESC
	// [0]
}

// Example_join_m2n_junction shows a M:N relationship through a
// junction table: students enrolled in many courses, courses
// have many students.
func Example_join_m2n_junction() {
	db := exampleDB()
	sql, args, _ := db.Select(
		"s.id AS student_id",
		"s.name AS student_name",
		"c.id AS course_id",
		"c.name AS course_name",
		"e.created_at AS enrolled_at",
	).
		From("enrollments e").
		Join("students s ON s.id = e.student_id").
		Join("courses c ON c.id = e.course_id").
		Where(xdb.Cond.Eq("e.status", "active")).
		OrderBy("e.created_at", xdb.DESC).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT s.id AS student_id, s.name AS student_name, c.id AS course_id, c.name AS course_name, e.created_at AS enrolled_at FROM enrollments e JOIN students s ON s.id = e.student_id JOIN courses c ON c.id = e.course_id WHERE e.status = $1 ORDER BY e.created_at DESC
	// [active]
}

// Example_join_m2n_aggregated shows an aggregated M:N count:
// how many students are enrolled in each course. Uses
// LEFT JOIN so courses with zero enrollments still appear.
func Example_join_m2n_aggregated() {
	db := exampleDB()
	sql, args, _ := db.Select(
		"c.id AS course_id",
		"c.name AS course_name",
		"COUNT(e.student_id) AS student_count",
	).
		From("courses c").
		LeftJoin("enrollments e ON e.course_id = c.id AND e.status = 'active'").
		GroupBy("c.id", "c.name").
		OrderBy("student_count", xdb.DESC).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT c.id AS course_id, c.name AS course_name, COUNT(e.student_id) AS student_count FROM courses c LEFT JOIN enrollments e ON e.course_id = c.id AND e.status = 'active' GROUP BY c.id, c.name ORDER BY student_count DESC
	// []
}

// Example_join_1to1_required shows a required 1:1 relationship:
// every user has exactly one profile. Uses INNER JOIN because
// both rows always exist together.
func Example_join_1to1_required() {
	db := exampleDB()
	sql, _, _ := db.Select(
		"u.id AS user_id",
		"u.name AS user_name",
		"p.avatar_url",
		"p.bio",
	).
		From("users u").
		Join("user_profiles p ON p.user_id = u.id").
		OrderBy("u.name", xdb.ASC).
		ToSQL()
	fmt.Println(sql)
	// Output:
	// SELECT u.id AS user_id, u.name AS user_name, p.avatar_url, p.bio FROM users u JOIN user_profiles p ON p.user_id = u.id ORDER BY u.name ASC
}

// Example_join_1to1_optional shows an optional 1:1 relationship:
// some users may lack a profile. Uses LEFT JOIN; profile fields
// should use *string or sql.Null* types to handle NULL.
func Example_join_1to1_optional() {
	db := exampleDB()
	sql, _, _ := db.Select(
		"u.id AS user_id",
		"u.name AS user_name",
		"p.avatar_url",
		"p.bio",
	).
		From("users u").
		LeftJoin("user_profiles p ON p.user_id = u.id").
		OrderBy("u.name", xdb.ASC).
		ToSQL()
	fmt.Println(sql)
	// Output:
	// SELECT u.id AS user_id, u.name AS user_name, p.avatar_url, p.bio FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id ORDER BY u.name ASC
}

// Example_join_m2n_cross shows a self-referencing M:N cross-match:
// finding similar articles by comparing every pair. Uses
// a.id < b.id to avoid duplicate pairs (a,b) and (b,a).
func Example_join_m2n_cross() {
	db := exampleDB()
	sql, args, _ := db.Select(
		"a.id AS left_id",
		"b.id AS right_id",
		"similarity(a.title, b.title) AS score",
	).
		From("articles a").
		Join("articles b ON a.id < b.id").
		Where(xdb.Cond.Gt("similarity(a.title, b.title)", 0.8)).
		OrderBy("score", xdb.DESC).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT a.id AS left_id, b.id AS right_id, similarity(a.title, b.title) AS score FROM articles a JOIN articles b ON a.id < b.id WHERE similarity(a.title, b.title) > $1 ORDER BY score DESC
	// [0.8]
}

// ─────────────────────────────────────────────────────────────
// Suite 9: Advanced CTE — raw multi-CTE, SelectBuilder CTE,
//         multi-stage pipeline
// ─────────────────────────────────────────────────────────────

// Example_cte_raw shows a raw multi-CTE pipeline: regional_sales
// aggregates orders by region, then top_regions filters the top
// 10% by revenue. CTEs are defined with WithCTE(name, rawSQL).
func Example_cte_raw() {
	db := exampleDB()
	sql, _, _ := db.CTE().
		WithCTE("regional_sales", "SELECT region, SUM(amount) AS total_sales FROM orders GROUP BY region").
		WithCTE("top_regions", "SELECT region FROM regional_sales WHERE total_sales > (SELECT SUM(total_sales)/10 FROM regional_sales)").
		Select("region", "product", "SUM(quantity) AS product_units", "SUM(amount) AS product_sales").
		From("orders").
		Where(xdb.Cond.Raw("region IN (SELECT region FROM top_regions)")).
		GroupBy("region", "product").
		OrderBy("product_sales", xdb.DESC).
		ToSQL()
	fmt.Println(sql)
	// Output:
	// WITH regional_sales AS (SELECT region, SUM(amount) AS total_sales FROM orders GROUP BY region), top_regions AS (SELECT region FROM regional_sales WHERE total_sales > (SELECT SUM(total_sales)/10 FROM regional_sales)) SELECT region, product, SUM(quantity) AS product_units, SUM(amount) AS product_sales FROM orders WHERE region IN (SELECT region FROM top_regions) GROUP BY region, product ORDER BY product_sales DESC
}

// Example_cte_select_builder shows a CTE defined from an existing
// SelectBuilder via WithSelectCTE. This enables composable subqueries
// that can be reused across multiple queries.
func Example_cte_select_builder() {
	db := exampleDB()
	activeCTE := db.Select("id", "name", "email").
		From("users").
		Where(xdb.Cond.Eq("status", "active"))

	sql, _, _ := db.CTE().
		WithSelectCTE("active_users", activeCTE).
		Select("u.id AS user_id", "u.name AS user_name").
		From("active_users u").
		OrderBy("u.name", xdb.ASC).
		ToSQL()
	fmt.Println(sql)
	// Output:
	// WITH active_users AS (SELECT id, name, email FROM users WHERE status = $1) SELECT u.id AS user_id, u.name AS user_name FROM active_users u ORDER BY u.name ASC
}

// Example_cte_multi_stage shows a three-stage CTE pipeline:
// filter → aggregate → rank. This pattern is useful for
// complex data processing without subquery nesting.
func Example_cte_multi_stage() {
	db := exampleDB()
	sql, _, _ := db.CTE().
		WithCTE("sales_by_category", "SELECT category, SUM(amount) AS total FROM sales WHERE date >= NOW() - INTERVAL '90 days' GROUP BY category").
		WithCTE("ranked_categories", "SELECT category, total, ROW_NUMBER() OVER (ORDER BY total DESC) AS rank FROM sales_by_category").
		Select("category", "total", "rank").
		From("ranked_categories").
		Where(xdb.Cond.LtOrEq("rank", 10)).
		OrderBy("rank", xdb.ASC).
		ToSQL()
	fmt.Println(sql)
	// Output:
	// WITH sales_by_category AS (SELECT category, SUM(amount) AS total FROM sales WHERE date >= NOW() - INTERVAL '90 days' GROUP BY category), ranked_categories AS (SELECT category, total, ROW_NUMBER() OVER (ORDER BY total DESC) AS rank FROM sales_by_category) SELECT category, total, rank FROM ranked_categories WHERE rank <= $1 ORDER BY rank ASC
}

// ─────────────────────────────────────────────────────────────
// Suite 10: Transactions & Locking
// ─────────────────────────────────────────────────────────────

// Example_transaction shows the Tx() helper.
// The callback receives a *TxDB with the same builder API.
// Returning nil commits; returning an error or panicking
// triggers an automatic rollback.
func Example_transaction() {
	db := exampleDB()
	sql, args, _ := db.Update("accounts").
		SetExpr("balance", "balance - ?", 100).
		Where(xdb.Cond.Eq("id", "from-account")).
		ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// UPDATE accounts SET balance = balance - $1 WHERE id = $2
	// [100 from-account]
}

// Example_locking shows the Lock() builder for row-level locking.
// ForUpdate, ForNoKeyUpdate, ForShare, and forKeyShare are
// available, optionally combined with SkipLocked, NoWait,
// and Of(tables...).
func Example_locking() {
	db := exampleDB()
	sql, _, _ := db.Select("*").
		From("orders").
		Where(xdb.Cond.Eq("status", "pending")).
		Lock(xdb.ForUpdate().SkipLocked()).
		ToSQL()
	fmt.Println(sql)
	// Output:
	// SELECT * FROM orders WHERE status = $1 FOR UPDATE SKIP LOCKED
}

// ─────────────────────────────────────────────────────────────
// Suite 11: Migrations — Up, Down, To, Step
// ─────────────────────────────────────────────────────────────

// Example_migrate shows the migration helpers.
// MigrateUp/Down/To/Step use golang-migrate under the hood.
// The caller must blank-import the appropriate database driver:
//
//	import _ "github.com/golang-migrate/migrate/v4/database/postgres"
//
// Migration files follow the convention:
//
//	000001_create_users.up.sql
//	000001_create_users.down.sql
func Example_migrate() {
	fmt.Println(`db.MigrateUp(embedFS, "migrations")
db.MigrateDown(embedFS, "migrations")
db.MigrateTo(embedFS, "migrations", 3)
db.MigrateStep(embedFS, "migrations", 1)`)
	// Output:
	// db.MigrateUp(embedFS, "migrations")
	// db.MigrateDown(embedFS, "migrations")
	// db.MigrateTo(embedFS, "migrations", 3)
	// db.MigrateStep(embedFS, "migrations", 1)
}

// ─────────────────────────────────────────────────────────────
// Suite 12: Raw SQL (escape hatch)
// ─────────────────────────────────────────────────────────────

// Example_raw_sql shows the RawOne / RawAll / RawExec escape
// hatches for queries that the builder cannot express.
// These pass SQL directly to the driver without builder processing.
func Example_raw_sql() {
	sql := "SELECT COUNT(*) FROM users WHERE active = $1"
	fmt.Println(sql)
	// Output:
	// SELECT COUNT(*) FROM users WHERE active = $1
}

// ─────────────────────────────────────────────────────────────
// Suite 13: UNION / INTERSECT / EXCEPT
// ─────────────────────────────────────────────────────────────

func Example_union() {
	db := exampleDB()
	active := db.Select("id", "name").From("users").Where(xdb.Cond.Eq("status", "active"))
	archived := db.Select("id", "name").From("users_archive").Where(xdb.Cond.Eq("status", "active"))

	sql, args, _ := active.Union(archived).OrderBy("name", xdb.ASC).ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT id, name FROM users WHERE status = $1
	// UNION
	// SELECT id, name FROM users_archive WHERE status = $2
	// ORDER BY name ASC
	// [active active]
}

func Example_union_all() {
	db := exampleDB()
	a := db.Select("id").From("t1")
	b := db.Select("id").From("t2")

	sql, _, _ := a.UnionAll(b).ToSQL()
	fmt.Println(sql)
	// Output:
	// SELECT id FROM t1
	// UNION ALL
	// SELECT id FROM t2
}

func Example_intersect() {
	db := exampleDB()
	a := db.Select("id").From("t1")
	b := db.Select("id").From("t2")

	sql, _, _ := a.Intersect(b).ToSQL()
	fmt.Println(sql)
	// Output:
	// SELECT id FROM t1
	// INTERSECT
	// SELECT id FROM t2
}

func Example_except() {
	db := exampleDB()
	a := db.Select("id").From("t1")
	b := db.Select("id").From("t2")

	sql, _, _ := a.Except(b).ToSQL()
	fmt.Println(sql)
	// Output:
	// SELECT id FROM t1
	// EXCEPT
	// SELECT id FROM t2
}

// ─────────────────────────────────────────────────────────────
// Suite 14: Subquery in FROM
// ─────────────────────────────────────────────────────────────

func Example_from_subquery() {
	db := exampleDB()
	inner := db.Select("id", "name").From("users").Where(xdb.Cond.Eq("active", true))

	sql, args, _ := db.Select("*").FromSubquery(inner, "active_users").OrderBy("name", xdb.ASC).ToSQL()
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT * FROM (SELECT id, name FROM users WHERE active = $1) AS active_users ORDER BY name ASC
	// [true]
}

// ─────────────────────────────────────────────────────────────
// Suite 15: Window functions
// ─────────────────────────────────────────────────────────────

func Example_window_row_number() {
	db := exampleDB()
	sql, _, _ := db.Select(
		"name",
		xdb.RowNumber().Over().PartitionBy("department").OrderBy("salary", xdb.DESC).As("rank"),
	).From("employees").ToSQL()
	fmt.Println(sql)
	// Output:
	// SELECT name, ROW_NUMBER() OVER (PARTITION BY department ORDER BY salary DESC) AS rank FROM employees
}

func Example_window_rank() {
	db := exampleDB()
	sql, _, _ := db.Select(
		"product",
		"SUM(amount) AS total",
		xdb.Rank().Over().OrderBy("SUM(amount)", xdb.DESC).As("rank"),
	).From("sales").GroupBy("product").ToSQL()
	fmt.Println(sql)
	// Output:
	// SELECT product, SUM(amount) AS total, RANK() OVER (ORDER BY SUM(amount) DESC) AS rank FROM sales GROUP BY product
}

func Example_window_lag() {
	db := exampleDB()
	sql, _, _ := db.Select(
		"date",
		"amount",
		xdb.Lag().Args("amount", 1).Over().OrderBy("date", xdb.ASC).As("prev_amount"),
	).From("daily_revenue").ToSQL()
	fmt.Println(sql)
	// Output:
	// SELECT date, amount, LAG(amount, 1) OVER (ORDER BY date ASC) AS prev_amount FROM daily_revenue
}

// ─────────────────────────────────────────────────────────────
// Suite 16: JSONB & Array operators
// ─────────────────────────────────────────────────────────────

func Example_cond_json_contains() {
	sql, args, _ := xdb.Cond.JsonContains("metadata", `{"status": "active"}`).ToSQL()
	fmt.Println(sql, args)
	// Output:
	// metadata @> ? [{"status": "active"}]
}

func Example_cond_json_has_key() {
	sql, args, _ := xdb.Cond.JsonHasKey("metadata", "description").ToSQL()
	fmt.Println(sql, args)
	// Output:
	// jsonb_exists(metadata, ?) [description]
}

func Example_cond_array_overlaps() {
	sql, args, _ := xdb.Cond.ArrayOverlaps("roles", "admin", "editor").ToSQL()
	fmt.Println(sql, args)
	// Output:
	// roles && ARRAY[?,?] [admin editor]
}

func Example_cond_array_contains() {
	sql, args, _ := xdb.Cond.ArrayContains("tags", "go", "database").ToSQL()
	fmt.Println(sql, args)
	// Output:
	// tags @> ARRAY[?,?] [go database]
}

// ─────────────────────────────────────────────────────────────
// Suite 17: PaginateWithCount (single-round-trip pagination)
// ─────────────────────────────────────────────────────────────

func Example_paginate_with_count() {
	// PaginateWithCount executes a single query with COUNT(*) OVER()
	// instead of two separate queries (COUNT + data).
	// The count column (xdb_total) is extracted programmatically.
	fmt.Println(`result, err := xdb.PaginateWithCount[User](ctx,
    db.Select("id", "name").From("users").OrderBy("name", xdb.ASC),
    xdb.Page{Number: 1, Size: 20},
)`)
	// Output:
	// result, err := xdb.PaginateWithCount[User](ctx,
	//     db.Select("id", "name").From("users").OrderBy("name", xdb.ASC),
	//     xdb.Page{Number: 1, Size: 20},
	// )
}
