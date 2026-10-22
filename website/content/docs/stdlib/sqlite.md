---
title: "sqlite"
description: "Embedded relational database powered by SQLite."
icon: "database"
weight: 350
---

```trux
import "sqlite"
```

**What it is.** Full embedded SQLite relational database support, bundled with SQLite 3.47.2. Operates on in-memory databases (`":memory:"`) or disk files with zero external dependencies.

**Why.** Applications often need structured storage, indexes, relational queries, and ACID transactions without running external database daemons.

**When you don't need it.** For simple structured flat files or tables, [`csv`](/docs/stdlib/csv/) or [`io`](/docs/stdlib/io/) may suffice.

| Function | Returns | Description |
| --- | --- | --- |
| `sqlite.open(path string)` | `(*sqlite.DB, error)` | Opens or creates a database connection at `path` (or `":memory:"`). |
| `sqlite.close(db *sqlite.DB)` | `error` | Closes the database connection. |
| `sqlite.exec(db *sqlite.DB, sql string)` | `error` | Executes one or more DDL/DML statements (e.g. `CREATE TABLE`, `INSERT`, `UPDATE`). |
| `sqlite.query(db *sqlite.DB, sql string)` | `(list[map[string]string], error)` | Executes a query returning all rows as string-keyed maps. |
| `sqlite.lastInsertRowId(db *sqlite.DB)` | `int` | Returns the `ROWID` of the most recent successful `INSERT`. |
| `sqlite.changes(db *sqlite.DB)` | `int` | Returns the number of database rows modified by the most recent statement. |
| `sqlite.totalChanges(db *sqlite.DB)` | `int` | Returns total rows modified since the database was opened. |
| `sqlite.begin(db *sqlite.DB)` | `error` | Begins a database transaction. |
| `sqlite.commit(db *sqlite.DB)` | `error` | Commits the active transaction. |
| `sqlite.rollback(db *sqlite.DB)` | `error` | Rolls back the active transaction. |
| `sqlite.prepare(db *sqlite.DB, sql string)` | `(*sqlite.Stmt, error)` | Compiles a SQL statement into a prepared statement. |
| `sqlite.step(stmt *sqlite.Stmt)` | `(bool, error)` | Advances to the next result row. Returns `(true, nil)` on a row, `(false, nil)` when done, or `(false, err)` on failure. |
| `sqlite.reset(stmt *sqlite.Stmt)` | `error` | Resets a prepared statement back to its initial state for re-execution. |
| `sqlite.clearBindings(stmt *sqlite.Stmt)` | `error` | Clears all bound parameter values back to `NULL`. |
| `sqlite.finalize(stmt *sqlite.Stmt)` | `error` | Destroys a prepared statement. |
| `sqlite.bindInt(stmt *sqlite.Stmt, index int, val int)` | `error` | Binds an integer to a 1-based parameter placeholder. |
| `sqlite.bindFloat(stmt *sqlite.Stmt, index int, val float)` | `error` | Binds a float to a 1-based parameter placeholder. |
| `sqlite.bindText(stmt *sqlite.Stmt, index int, val string)` | `error` | Binds text to a 1-based parameter placeholder. |
| `sqlite.bindNull(stmt *sqlite.Stmt, index int)` | `error` | Binds `NULL` to a 1-based parameter placeholder. |
| `sqlite.bindIndex(stmt *sqlite.Stmt, name string)` | `int` | Returns the 1-based index of a named parameter placeholder (e.g. `":id"`). |
| `sqlite.columnCount(stmt *sqlite.Stmt)` | `int` | Returns the number of columns in the result set. |
| `sqlite.columnName(stmt *sqlite.Stmt, col int)` | `string` | Returns the name of the 0-based column. |
| `sqlite.columnIndex(stmt *sqlite.Stmt, name string)` | `int` | Returns the 0-based index of a column by name, or `-1` if not found. |
| `sqlite.columnType(stmt *sqlite.Stmt, col int)` | `string` | Returns the SQLite datatype name (`"INTEGER"`, `"FLOAT"`, `"TEXT"`, `"BLOB"`, `"NULL"`). |
| `sqlite.columnInt(stmt *sqlite.Stmt, col int)` | `int` | Returns column value as an integer. |
| `sqlite.columnFloat(stmt *sqlite.Stmt, col int)` | `float` | Returns column value as a float. |
| `sqlite.columnText(stmt *sqlite.Stmt, col int)` | `string` | Returns column value as text. |
| `sqlite.columnIsNull(stmt *sqlite.Stmt, col int)` | `bool` | Returns `true` if the column value is `NULL`. |

## Quick Queries

For straightforward queries, `sqlite.query` executes the statement and returns all rows as a list of string maps:

```trux
package main

import "sqlite"

func main() int {
    let db *sqlite.DB, err error = sqlite.open(":memory:")
    if err != nil {
        return 1
    }

    sqlite.exec(db, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INT);")
    sqlite.exec(db, "INSERT INTO users (name, age) VALUES ('Alice', 30);")
    sqlite.exec(db, "INSERT INTO users (name, age) VALUES ('Bob', 25);")

    let rows list[map[string]string], qerr error = sqlite.query(db, "SELECT id, name, age FROM users ORDER BY age;")
    if qerr != nil {
        return 1
    }

    for let i int = 0; i < len(rows); i = i + 1 {
        let row map[string]string = rows[i]
        print(row["name"], " ", row["age"])
    }

    sqlite.close(db)
    return 0
}
```

## Prepared Statements

For parameterized queries and typed column retrieval, use `sqlite.prepare`, `sqlite.bind*`, and `sqlite.step`:

```trux
package main

import "sqlite"

func main() int {
    let db *sqlite.DB, err error = sqlite.open("app.db")
    if err != nil {
        return 1
    }

    let stmt *sqlite.Stmt, perr error = sqlite.prepare(db, "SELECT name, age FROM users WHERE age > ?;")
    if perr != nil {
        return 1
    }

    sqlite.bindInt(stmt, 1, 20)

    for {
        let hasRow bool, serr error = sqlite.step(stmt)
        if serr != nil || !hasRow {
            break
        }
        let name string = sqlite.columnText(stmt, 0)
        let age int = sqlite.columnInt(stmt, 1)
        print(name, " ", age)
    }

    sqlite.finalize(stmt)
    sqlite.close(db)
    return 0
}
```

## Transactions

Transactions allow bundling multiple statements atomically:

```trux
sqlite.begin(db)
let err1 error = sqlite.exec(db, "UPDATE accounts SET balance = balance - 100 WHERE id = 1;")
let err2 error = sqlite.exec(db, "UPDATE accounts SET balance = balance + 100 WHERE id = 2;")
if err1 != nil || err2 != nil {
    sqlite.rollback(db)
} else {
    sqlite.commit(db)
}
```
