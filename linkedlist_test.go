package linkedlist

import (
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestNew(t *testing.T) {
	ll := New()
	if ll == nil {
		t.Fatal("New() returned nil")
	}
	if ll.head != nil {
		t.Errorf("Expected head to be nil, got %v", ll.head)
	}
	if ll.tail != nil {
		t.Errorf("Expected tail to be nil, got %v", ll.tail)
	}
	if ll.current != nil {
		t.Errorf("Expected current to be nil, got %v", ll.current)
	}
	if ll.len != 0 {
		t.Errorf("Expected len to be 0, got %d", ll.len)
	}
}
func TestStructScan_BasicFields(t *testing.T) {
	type User struct {
		ID   int
		Name string
		Age  int
	}
	node := &Node{
		Data: map[string]interface{}{
			"ID":   1,
			"Name": "Alice",
			"Age":  30,
		},
	}
	var u User
	err := node.StructScan(&u)
	if err != nil {
		t.Fatalf("StructScan failed: %v", err)
	}
	if u.ID != 1 || u.Name != "Alice" || u.Age != 30 {
		t.Errorf("StructScan result mismatch: %+v", u)
	}
}

func TestStructScan_WithTags(t *testing.T) {
	type User struct {
		UserID   int    `db:"user_id"`
		FullName string `json:"full_name"`
	}
	node := &Node{
		Data: map[string]interface{}{
			"user_id":   42,
			"full_name": "Bob Smith",
		},
	}
	var u User
	err := node.StructScan(&u)
	if err != nil {
		t.Fatalf("StructScan failed: %v", err)
	}
	if u.UserID != 42 || u.FullName != "Bob Smith" {
		t.Errorf("StructScan result mismatch: %+v", u)
	}
}

func TestStructScan_CaseInsensitive(t *testing.T) {
	type User struct {
		Email string
	}
	node := &Node{
		Data: map[string]interface{}{
			"EMAIL": "test@example.com",
		},
	}
	var u User
	err := node.StructScan(&u)
	if err != nil {
		t.Fatalf("StructScan failed: %v", err)
	}
	if u.Email != "test@example.com" {
		t.Errorf("Expected Email to be 'test@example.com', got '%s'", u.Email)
	}
}

func TestStructScan_NilData(t *testing.T) {
	type Dummy struct{ X int }
	node := &Node{Data: nil}
	var d Dummy
	err := node.StructScan(&d)
	if err == nil {
		t.Error("Expected error for nil Data, got nil")
	}
}

func TestStructScan_NonPointerDest(t *testing.T) {
	type Dummy struct{ X int }
	node := &Node{Data: map[string]interface{}{"X": 1}}
	var d Dummy
	err := node.StructScan(d)
	if err == nil {
		t.Error("Expected error for non-pointer dest, got nil")
	}
}

func TestStructScan_NonStructPointer(t *testing.T) {
	node := &Node{Data: map[string]interface{}{"X": 1}}
	var x int
	err := node.StructScan(&x)
	if err == nil {
		t.Error("Expected error for non-struct pointer, got nil")
	}
}

func TestStructScan_FieldTypeConversion(t *testing.T) {
	type User struct {
		ID int
	}
	node := &Node{
		Data: map[string]interface{}{
			"ID": int64(123),
		},
	}
	var u User
	err := node.StructScan(&u)
	if err != nil {
		t.Fatalf("StructScan failed: %v", err)
	}
	if u.ID != 123 {
		t.Errorf("Expected ID to be 123, got %d", u.ID)
	}
}

func TestStructScan_PointerField(t *testing.T) {
	type User struct {
		Age *int
	}
	age := 25
	node := &Node{
		Data: map[string]interface{}{
			"Age": age,
		},
	}
	var u User
	err := node.StructScan(&u)
	if err != nil {
		t.Fatalf("StructScan failed: %v", err)
	}
	if u.Age == nil || *u.Age != 25 {
		t.Errorf("Expected Age pointer to 25, got %v", u.Age)
	}
}

func TestStructScan_IgnoreUnexportedFields(t *testing.T) {
	type User struct {
		ID   int
		name string // unexported
	}
	node := &Node{
		Data: map[string]interface{}{
			"ID":   7,
			"name": "hidden",
		},
	}
	var u User
	err := node.StructScan(&u)
	if err != nil {
		t.Fatalf("StructScan failed: %v", err)
	}
	if u.ID != 7 {
		t.Errorf("Expected ID to be 7, got %d", u.ID)
	}
	// can't check u.name as it's unexported, but should not panic or set
}

func TestStructScan_NullValue(t *testing.T) {
	type User struct {
		Name string
	}
	node := &Node{
		Data: map[string]interface{}{
			"Name": nil,
		},
	}
	var u User
	err := node.StructScan(&u)
	if err != nil {
		t.Fatalf("StructScan failed: %v", err)
	}
	if u.Name != "" {
		t.Errorf("Expected Name to be empty string, got '%s'", u.Name)
	}
}
func TestAppend_FirstNode(t *testing.T) {
	ll := New()
	data := map[string]interface{}{"ID": 1, "Name": "Alice"}
	ll.Append(data)

	if ll.head == nil || ll.tail == nil {
		t.Fatal("After first Append, head or tail is nil")
	}
	if ll.head != ll.tail {
		t.Error("After first Append, head and tail should be the same")
	}
	if ll.head.Data["ID"] != 1 || ll.head.Data["Name"] != "Alice" {
		t.Errorf("Head node data mismatch: %+v", ll.head.Data)
	}
	if ll.len != 1 {
		t.Errorf("Expected len to be 1, got %d", ll.len)
	}
}

func TestAppend_MultipleNodes(t *testing.T) {
	ll := New()
	data1 := map[string]interface{}{"ID": 1}
	data2 := map[string]interface{}{"ID": 2}
	data3 := map[string]interface{}{"ID": 3}

	ll.Append(data1)
	ll.Append(data2)
	ll.Append(data3)

	if ll.len != 3 {
		t.Errorf("Expected len to be 3, got %d", ll.len)
	}
	if ll.head.Data["ID"] != 1 {
		t.Errorf("Expected head ID to be 1, got %v", ll.head.Data["ID"])
	}
	if ll.tail.Data["ID"] != 3 {
		t.Errorf("Expected tail ID to be 3, got %v", ll.tail.Data["ID"])
	}
	// Check linkage
	if ll.head.next == nil || ll.head.next.Data["ID"] != 2 {
		t.Error("Second node not linked properly from head")
	}
	if ll.head.next.next == nil || ll.head.next.next.Data["ID"] != 3 {
		t.Error("Third node not linked properly from second node")
	}
	if ll.head.next.next.next != nil {
		t.Error("There should be only three nodes in the list")
	}
}

func TestAppend_CurrentPointer(t *testing.T) {
	ll := New()
	data := map[string]interface{}{"ID": 10}
	ll.Append(data)
	if ll.current == nil {
		t.Error("Current pointer should not be nil after first append")
	}
	if ll.current.Data["ID"] != 10 {
		t.Errorf("Current pointer data mismatch: %+v", ll.current.Data)
	}
}

func TestAppend_EmptyData(t *testing.T) {
	ll := New()
	ll.Append(nil)
	if ll.len != 1 {
		t.Errorf("Expected len to be 1 after appending nil data, got %d", ll.len)
	}
	if ll.head == nil || ll.head.Data != nil {
		t.Errorf("Expected head.Data to be nil, got %+v", ll.head.Data)
	}
}
func TestFirstAndLast(t *testing.T) {
	ll := New()
	if ll.First() != nil {
		t.Error("First() should return nil on empty list")
	}
	if ll.Last() != nil {
		t.Error("Last() should return nil on empty list")
	}

	data1 := map[string]interface{}{"ID": 1}
	data2 := map[string]interface{}{"ID": 2}
	ll.Append(data1)
	ll.Append(data2)

	first := ll.First()
	last := ll.Last()
	if first == nil || last == nil {
		t.Fatal("First() or Last() returned nil after appends")
	}
	if first.Data["ID"] != 1 {
		t.Errorf("First() returned wrong node: %+v", first.Data)
	}
	if last.Data["ID"] != 2 {
		t.Errorf("Last() returned wrong node: %+v", last.Data)
	}
}

func TestNextAndResetIterator(t *testing.T) {
	ll := New()
	data1 := map[string]interface{}{"ID": 1}
	data2 := map[string]interface{}{"ID": 2}
	data3 := map[string]interface{}{"ID": 3}
	ll.Append(data1)
	ll.Append(data2)
	ll.Append(data3)

	// Test iteration
	ll.ResetIterator()
	var ids []int
	for node := ll.Next(); node != nil; node = ll.Next() {
		ids = append(ids, node.Data["ID"].(int))
	}
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Errorf("Next() iteration failed, got ids: %v", ids)
	}

	// Test ResetIterator
	ll.ResetIterator()
	node := ll.Next()
	if node == nil || node.Data["ID"] != 1 {
		t.Errorf("After ResetIterator, expected first node with ID 1, got %+v", node)
	}
}

func TestNextOnEmptyList(t *testing.T) {
	ll := New()
	if ll.Next() != nil {
		t.Error("Next() should return nil on empty list")
	}
}

func TestLen(t *testing.T) {
	ll := New()
	if ll.Len() != 0 {
		t.Errorf("Expected Len() to be 0, got %d", ll.Len())
	}
	ll.Append(map[string]interface{}{"ID": 1})
	if ll.Len() != 1 {
		t.Errorf("Expected Len() to be 1, got %d", ll.Len())
	}
	ll.Append(map[string]interface{}{"ID": 2})
	if ll.Len() != 2 {
		t.Errorf("Expected Len() to be 2, got %d", ll.Len())
	}
}
func TestToSlice_Basic(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}
	ll := New()
	ll.Append(map[string]interface{}{"ID": 1, "Name": "Alice"})
	ll.Append(map[string]interface{}{"ID": 2, "Name": "Bob"})

	var users []User
	err := ll.ToSlice(&users)
	if err != nil {
		t.Fatalf("ToSlice failed: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(users))
	}
	if users[0].ID != 1 || users[0].Name != "Alice" {
		t.Errorf("First user mismatch: %+v", users[0])
	}
	if users[1].ID != 2 || users[1].Name != "Bob" {
		t.Errorf("Second user mismatch: %+v", users[1])
	}
}

func TestToSlice_EmptyList(t *testing.T) {
	type Item struct{ X int }
	ll := New()
	var items []Item
	err := ll.ToSlice(&items)
	if err != nil {
		t.Fatalf("ToSlice failed on empty list: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Expected empty slice, got %d elements", len(items))
	}
}

func TestToSlice_NonPointerDest(t *testing.T) {
	type User struct{ ID int }
	ll := New()
	ll.Append(map[string]interface{}{"ID": 1})
	var users []User
	err := ll.ToSlice(users) // not a pointer
	if err == nil {
		t.Error("Expected error for non-pointer dest, got nil")
	}
}

func TestToSlice_NonSlicePointer(t *testing.T) {
	type User struct{ ID int }
	ll := New()
	ll.Append(map[string]interface{}{"ID": 1})
	var user User
	err := ll.ToSlice(&user) // pointer, but not to slice
	if err == nil {
		t.Error("Expected error for pointer to non-slice, got nil")
	}
}

func TestToSlice_FieldTypeConversion(t *testing.T) {
	type User struct{ ID int }
	ll := New()
	ll.Append(map[string]interface{}{"ID": int64(123)})
	var users []User
	err := ll.ToSlice(&users)
	if err != nil {
		t.Fatalf("ToSlice failed: %v", err)
	}
	if len(users) != 1 || users[0].ID != 123 {
		t.Errorf("Expected one user with ID 123, got %+v", users)
	}
}

func TestToSlice_StructScanError(t *testing.T) {
	type User struct{ ID int }
	ll := New()
	ll.Append(map[string]interface{}{"ID": "not-an-int"})
	var users []User
	err := ll.ToSlice(&users)
	if err == nil {
		t.Error("Expected error from StructScan, got nil")
	}
}
func TestLoadFromSQLx_AppendsRows(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}
	// Use github.com/DATA-DOG/go-sqlmock for mocking database/sql and sqlx
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock DB: %v", err)
	}
	defer sqlDB.Close()
	db := sqlx.NewDb(sqlDB, "sqlmock")

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "Alice").
		AddRow(2, "Bob")
	mock.ExpectQuery("SELECT id, name FROM users").WillReturnRows(rows)

	sqlxRows, err := db.Queryx("SELECT id, name FROM users")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	defer sqlxRows.Close()

	ll := New()
	err = ll.LoadFromSQLx(sqlxRows)
	if err != nil {
		t.Fatalf("LoadFromSQLx failed: %v", err)
	}
	if ll.Len() != 2 {
		t.Errorf("Expected 2 nodes, got %d", ll.Len())
	}
	ll.ResetIterator()
	node := ll.Next()
	if node == nil || node.Data["id"] != int64(1) || node.Data["name"] != "Alice" {
		t.Errorf("First node data mismatch: %+v", node.Data)
	}
	node = ll.Next()
	if node == nil || node.Data["id"] != int64(2) || node.Data["name"] != "Bob" {
		t.Errorf("Second node data mismatch: %+v", node.Data)
	}
}

func TestScanRowToMap_BytesConversion(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock DB: %v", err)
	}
	defer sqlDB.Close()
	db := sqlx.NewDb(sqlDB, "sqlmock")

	rows := sqlmock.NewRows([]string{"data"}).
		AddRow([]byte("hello"))
	mock.ExpectQuery("SELECT data FROM test").WillReturnRows(rows)

	sqlxRows, err := db.Queryx("SELECT data FROM test")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	defer sqlxRows.Close()

	if !sqlxRows.Next() {
		t.Fatal("No rows returned")
	}
	rowData, err := scanRowToMap(sqlxRows)
	if err != nil {
		t.Fatalf("scanRowToMap failed: %v", err)
	}
	if rowData["data"] != "hello" {
		t.Errorf("Expected 'hello', got %v", rowData["data"])
	}
}

func TestSetFieldValue_BasicTypes(t *testing.T) {
	var i int
	field := reflect.ValueOf(&i).Elem()
	err := setFieldValue(field, reflect.TypeOf(i), 42)
	if err != nil {
		t.Fatalf("setFieldValue failed: %v", err)
	}
	if i != 42 {
		t.Errorf("Expected 42, got %d", i)
	}

	var s string
	field = reflect.ValueOf(&s).Elem()
	err = setFieldValue(field, reflect.TypeOf(s), "hello")
	if err != nil {
		t.Fatalf("setFieldValue failed: %v", err)
	}
	if s != "hello" {
		t.Errorf("Expected 'hello', got '%s'", s)
	}
}

func TestSetFieldValue_Int64ToInt(t *testing.T) {
	var i int
	field := reflect.ValueOf(&i).Elem()
	err := setFieldValue(field, reflect.TypeOf(i), int64(123))
	if err != nil {
		t.Fatalf("setFieldValue failed: %v", err)
	}
	if i != 123 {
		t.Errorf("Expected 123, got %d", i)
	}
}

func TestSetFieldValue_PointerField(t *testing.T) {
	var pi *int
	field := reflect.ValueOf(&pi).Elem()
	err := setFieldValue(field, reflect.TypeOf(pi), 55)
	if err != nil {
		t.Fatalf("setFieldValue failed: %v", err)
	}
	if pi == nil || *pi != 55 {
		t.Errorf("Expected pointer to 55, got %v", pi)
	}
}

func TestSetFieldValue_PointerToPointerField(t *testing.T) {
	val := 77
	var pi *int
	field := reflect.ValueOf(&pi).Elem()
	err := setFieldValue(field, reflect.TypeOf(pi), &val)
	if err != nil {
		t.Fatalf("setFieldValue failed: %v", err)
	}
	if pi == nil || *pi != 77 {
		t.Errorf("Expected pointer to 77, got %v", pi)
	}
}

func TestSetFieldValue_TimeTime(t *testing.T) {
	var tm time.Time
	now := time.Now().Truncate(time.Second)
	field := reflect.ValueOf(&tm).Elem()
	err := setFieldValue(field, reflect.TypeOf(tm), now)
	if err != nil {
		t.Fatalf("setFieldValue failed: %v", err)
	}
	if !tm.Equal(now) {
		t.Errorf("Expected %v, got %v", now, tm)
	}
}

func TestSetFieldValue_TimeTimeFromString(t *testing.T) {
	var tm time.Time
	str := "2023-01-02T15:04:05Z"
	field := reflect.ValueOf(&tm).Elem()
	err := setFieldValue(field, reflect.TypeOf(tm), str)
	if err != nil {
		t.Fatalf("setFieldValue failed: %v", err)
	}
	expected, _ := time.Parse(time.RFC3339, str)
	if !tm.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, tm)
	}
}

func TestSetFieldValue_InvalidConversion(t *testing.T) {
	var i int
	field := reflect.ValueOf(&i).Elem()
	err := setFieldValue(field, reflect.TypeOf(i), "not-an-int")
	if err == nil {
		t.Error("Expected error for invalid conversion, got nil")
	}
}

func TestSetFieldValue_NilValue(t *testing.T) {
	var i int
	field := reflect.ValueOf(&i).Elem()
	err := setFieldValue(field, reflect.TypeOf(i), nil)
	if err != nil {
		t.Fatalf("setFieldValue failed for nil: %v", err)
	}
	// Should remain zero value
	if i != 0 {
		t.Errorf("Expected 0, got %d", i)
	}
}

func TestSetFieldValue_PointerFieldWithNil(t *testing.T) {
	var pi *int
	field := reflect.ValueOf(&pi).Elem()
	err := setFieldValue(field, reflect.TypeOf(pi), nil)
	if err != nil {
		t.Fatalf("setFieldValue failed for nil pointer: %v", err)
	}
	if pi != nil {
		t.Errorf("Expected nil pointer, got %v", pi)
	}
}
func TestScanRowToMap_SimpleRow(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock DB: %v", err)
	}
	defer sqlDB.Close()
	db := sqlx.NewDb(sqlDB, "sqlmock")

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "Alice")
	mock.ExpectQuery("SELECT id, name FROM users").WillReturnRows(rows)

	sqlxRows, err := db.Queryx("SELECT id, name FROM users")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	defer sqlxRows.Close()

	if !sqlxRows.Next() {
		t.Fatal("No rows returned")
	}
	rowData, err := scanRowToMap(sqlxRows)
	if err != nil {
		t.Fatalf("scanRowToMap failed: %v", err)
	}
	if rowData["id"] != int64(1) {
		t.Errorf("Expected id=1, got %v", rowData["id"])
	}
	if rowData["name"] != "Alice" {
		t.Errorf("Expected name='Alice', got %v", rowData["name"])
	}
}

func TestScanRowToMap_BytesToString(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock DB: %v", err)
	}
	defer sqlDB.Close()
	db := sqlx.NewDb(sqlDB, "sqlmock")

	rows := sqlmock.NewRows([]string{"data"}).
		AddRow([]byte("hello"))
	mock.ExpectQuery("SELECT data FROM test").WillReturnRows(rows)

	sqlxRows, err := db.Queryx("SELECT data FROM test")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	defer sqlxRows.Close()

	if !sqlxRows.Next() {
		t.Fatal("No rows returned")
	}
	rowData, err := scanRowToMap(sqlxRows)
	if err != nil {
		t.Fatalf("scanRowToMap failed: %v", err)
	}
	if rowData["data"] != "hello" {
		t.Errorf("Expected 'hello', got %v", rowData["data"])
	}
}

func TestScanRowToMap_NullValue(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock DB: %v", err)
	}
	defer sqlDB.Close()
	db := sqlx.NewDb(sqlDB, "sqlmock")

	rows := sqlmock.NewRows([]string{"id", "value"}).
		AddRow(nil, nil)
	mock.ExpectQuery("SELECT id, value FROM test").WillReturnRows(rows)

	sqlxRows, err := db.Queryx("SELECT id, value FROM test")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	defer sqlxRows.Close()

	if !sqlxRows.Next() {
		t.Fatal("No rows returned")
	}
	rowData, err := scanRowToMap(sqlxRows)
	if err != nil {
		t.Fatalf("scanRowToMap failed: %v", err)
	}
	if rowData["id"] != nil {
		t.Errorf("Expected id=nil, got %v", rowData["id"])
	}
	if rowData["value"] != nil {
		t.Errorf("Expected value=nil, got %v", rowData["value"])
	}
}

func TestScanRowToMap_ColumnsError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock DB: %v", err)
	}
	defer sqlDB.Close()
	db := sqlx.NewDb(sqlDB, "sqlmock")

	rows := sqlmock.NewRows([]string{"id"}).
		AddRow(1)
	mock.ExpectQuery("SELECT id FROM test").WillReturnRows(rows)

	sqlxRows, err := db.Queryx("SELECT id FROM test")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	sqlxRows.Close() // force error

	_, err = scanRowToMap(sqlxRows)
	if err == nil {
		t.Error("Expected error from scanRowToMap on closed rows, got nil")
	}
}

func TestScanRowToMap_ScanError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock DB: %v", err)
	}
	defer sqlDB.Close()
	db := sqlx.NewDb(sqlDB, "sqlmock")

	// Create a row with a value that cannot be scanned into interface{}
	rows := sqlmock.NewRows([]string{"id"}).
		AddRow("not-an-int")
	mock.ExpectQuery("SELECT id FROM test").WillReturnRows(rows)

	sqlxRows, err := db.Queryx("SELECT id FROM test")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	defer sqlxRows.Close()

	// Move to the row, but forcibly close to simulate scan error
	if !sqlxRows.Next() {
		t.Fatal("No rows returned")
	}
	// Simulate scan error by closing underlying rows
	sqlxRows.Rows.Close()
	_, err = scanRowToMap(sqlxRows)
	if err == nil {
		t.Error("Expected error from scanRowToMap due to scan error, got nil")
	}
}
