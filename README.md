# Go Linked List with SQL Support

[![Go Reference](https://pkg.go.dev/badge/github.com/ifanwar/go-linkedlist.svg)](https://pkg.go.dev/github.com/ifanwar/go-linkedlist)
[![Go Report Card](https://goreportcard.com/badge/github.com/ifanwar/go-linkedlist)](https://goreportcard.com/report/github.com/ifanwar/go-linkedlist)

A hybrid linked list implementation in Go that combines generic data storage with SQL query result processing capabilities.

## Features

- **Generic Linked List**:
  - Store any data type using `interface{}`
  - Standard linked list operations (append, prepend, insert, remove)
  - Bidirectional navigation

- **SQL Integration**:
  - Load data directly from `sqlx` query results
  - Scan SQL data into structs with `StructScan`
  - Supports `db` and `json` struct tags
  - Automatic type conversion

- **Flexible Usage**:
  - Use as pure linked list
  - Use as SQL result processor
  - Mix both generic and SQL data in same list

## Installation

```bash
go get github.com/ifanwar/go-linkedlist
```

Basic Usage
As Generic Linked List
```bash
package main

import (
	"fmt"
	"github.com/ifanwar/go-linkedlist"
)

func main() {
	// Create new list
	list := linkedlist.New()

	// Add elements
	list.Append("Hello")
	list.Append(42)
	list.Append(true)

	// Iterate through list
	list.ResetIterator()
	for node := list.Next(); node != nil; node = list.Next() {
		fmt.Println(node.Value)
	}
}
```
With SQL Integration
```bash
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/ifanwar/go-linkedlist"
)

type Product struct {
	ID    int     `db:"id"`
	Name  string  `db:"name"`
	Price float64 `db:"price"`
}

func main() {
	// Initialize database connection
	db, err := sqlx.Connect("postgres", "user=postgres dbname=test sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create list and load SQL data
	list := linkedlist.New()
	rows, err := db.Queryx("SELECT * FROM products WHERE price > $1", 10.0)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	err = list.LoadFromSQLx(rows)
	if err != nil {
		log.Fatal(err)
	}

	// Convert to struct slice
	var products []Product
	err = list.ToStructSlice(&products)
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range products {
		fmt.Printf("%d: %s ($%.2f)\n", p.ID, p.Name, p.Price)
	}
}
```

## API Reference

### Core Methods

| Method | Description |
|--------|-------------|
| `New()` | Creates new linked list |
| `Append(value interface{})` | Adds value to end of list |
| `Prepend(value interface{})` | Adds value to beginning of list |
| `InsertAt(index int, value interface{}) error` | Inserts value at position |
| `Remove(index int) error` | Removes node at position |
| `Get(index int) (interface{}, error)` | Gets value at position |
| `Len() int` | Returns list length |

### SQL Methods

| Method | Description |
|--------|-------------|
| `LoadFromSQLx(rows *sqlx.Rows) error` | Loads data from SQL query |
| `(n *Node) StructScan(dest interface{}) error` | Scans node data into struct |
| `ToStructSlice(destSlice interface{}) error` | Converts all SQL nodes to struct slice |

### Navigation Methods

| Method | Description |
|--------|-------------|
| `First() *Node` | Gets first node |
| `Last() *Node` | Gets last node |
| `Next() *Node` | Gets next node (iterator) |
| `ResetIterator()` | Resets iterator |

## Struct Tags

The library supports these struct tags for SQL data mapping:
```bash
type User struct {
    ID        int       `db:"user_id"`   // Maps to "user_id" column
    Name      string    `db:"name"`      // Maps to "name" column
    Email     string    `json:"email"`   // Can use json tag as fallback
    CreatedAt time.Time // Uses field name if no tag
}
```
## Performance

### Time Complexities

| Operation | Complexity | Notes |
|-----------|------------|-------|
| `Append()` | O(1) | Constant time addition to end |
| `Prepend()` | O(1) | Constant time addition to start |
| `First()`/`Last()` | O(1) | Immediate head/tail access |
| `Next()` iteration | O(1) | Per-element during traversal |
| `Get(index)` | O(n) | Linear scan for positional access |
| `InsertAt(index)` | O(n) | Requires traversal to position |
| `Remove(index)` | O(n) | Requires traversal to position |

### SQL Performance

| Operation | Performance Comparison |
|-----------|------------------------|
| `LoadFromSQLx()` | ~5-10% overhead vs direct sqlx |
| `StructScan()` | Comparable to sqlx's StructScan |
| `ToStructSlice()` | Similar to looping with sqlx |

### Real-World Benchmarks

```text
BenchmarkAppend-8          15.2 ns/op      0 B/op      0 allocs/op
BenchmarkPrepend-8         14.8 ns/op      0 B/op      0 allocs/op  
BenchmarkInsertAt_100-8    182.5 ns/op    32 B/op      1 allocs/op
BenchmarkStructScan-8      245.0 ns/op    80 B/op      3 allocs/op