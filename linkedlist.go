// Package linkedlist provides a linked list implementation.
package linkedlist

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// Node represents a single node in the linked list containing data.
type Node struct {
	Data map[string]interface{}
	next *Node
}

// LinkedList represents a linked list of data with scanning capabilities.
type LinkedList struct {
	head    *Node
	tail    *Node
	current *Node // for iteration
	len     int
}

// New creates a new empty linked list.
func New() *LinkedList {
	return &LinkedList{}
}

// StructScan scans the current node's data into the provided struct.
// The destination must be a pointer to a struct. Supports db and json struct tags.
func (n *Node) StructScan(dest interface{}) error {
	if n.Data == nil {
		return errors.New("node contains no data")
	}

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.IsNil() {
		return errors.New("destination must be a non-nil pointer")
	}

	destElem := destValue.Elem()
	if destElem.Kind() != reflect.Struct {
		return errors.New("destination must be a pointer to a struct")
	}

	destType := destElem.Type()

	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		fieldValue := destElem.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		// Get the field name, considering db and json tags
		fieldName := field.Name
		if tag := field.Tag.Get("db"); tag != "" {
			fieldName = tag
		} else if tag := field.Tag.Get("json"); tag != "" {
			if commaIdx := strings.Index(tag, ","); commaIdx != -1 {
				fieldName = tag[:commaIdx]
			} else {
				fieldName = tag
			}
		}

		// Try case-insensitive match if exact match not found
		var dataValue interface{}
		var found bool
		if dataValue, found = n.Data[fieldName]; !found {
			// Case-insensitive search
			for k, v := range n.Data {
				if strings.EqualFold(k, fieldName) {
					dataValue = v
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Handle NULL values
		if dataValue == nil {
			continue
		}

		// Convert the data value to the field type
		if err := setFieldValue(fieldValue, field.Type, dataValue); err != nil {
			return fmt.Errorf("error setting field %s: %w", fieldName, err)
		}
	}

	return nil
}

// setFieldValue handles the actual value conversion and assignment
func setFieldValue(field reflect.Value, fieldType reflect.Type, dataValue interface{}) error {
	// Special handling for time.Time
	if fieldType == reflect.TypeOf(time.Time{}) {
		if t, ok := dataValue.(time.Time); ok {
			field.Set(reflect.ValueOf(t))
			return nil
		}
		if s, ok := dataValue.(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				field.Set(reflect.ValueOf(t))
				return nil
			}
		}
	}

	dataVal := reflect.ValueOf(dataValue)
	if !dataVal.IsValid() {
		return nil
	}

	if dataVal.Type().ConvertibleTo(fieldType) {
		field.Set(dataVal.Convert(fieldType))
		return nil
	}

	if fieldType.Kind() == reflect.Ptr {
		// Handle pointer fields
		if dataVal.Kind() == reflect.Ptr {
			if dataVal.Elem().Type().ConvertibleTo(fieldType.Elem()) {
				newVal := reflect.New(fieldType.Elem())
				newVal.Elem().Set(dataVal.Elem().Convert(fieldType.Elem()))
				field.Set(newVal)
				return nil
			}
		} else {
			if dataVal.Type().ConvertibleTo(fieldType.Elem()) {
				newVal := reflect.New(fieldType.Elem())
				newVal.Elem().Set(dataVal.Convert(fieldType.Elem()))
				field.Set(newVal)
				return nil
			}
		}
	}

	return fmt.Errorf("cannot convert %T to %v", dataValue, fieldType)
}

// LoadFromSQLx loads data from sqlx rows into the linked list.
func (ll *LinkedList) LoadFromSQLx(rows *sqlx.Rows) error {
	for rows.Next() {
		rowData, err := scanRowToMap(rows)
		if err != nil {
			return err
		}
		ll.Append(rowData)
	}

	return rows.Err()
}

// scanRowToMap scans a single row into a map[string]interface{}
func scanRowToMap(rows *sqlx.Rows) (map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	values := make([]interface{}, len(cols))
	for i := range values {
		var v interface{}
		values[i] = &v
	}

	if err := rows.Scan(values...); err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	rowData := make(map[string]interface{})
	for i, col := range cols {
		val := reflect.Indirect(reflect.ValueOf(values[i])).Interface()
		if b, ok := val.([]byte); ok {
			rowData[col] = string(b)
		} else {
			rowData[col] = val
		}
	}

	return rowData, nil
}

// Append adds a new row to the end of the list.
func (ll *LinkedList) Append(data map[string]interface{}) {
	newNode := &Node{Data: data}

	if ll.head == nil {
		ll.head = newNode
		ll.tail = newNode
		ll.current = newNode
	} else {
		ll.tail.next = newNode
		ll.tail = newNode
	}
	ll.len++
}

// First returns the first node in the list.
func (ll *LinkedList) First() *Node {
	return ll.head
}

// Last returns the last node in the list.
func (ll *LinkedList) Last() *Node {
	return ll.tail
}

// Next returns the next node in iteration.
func (ll *LinkedList) Next() *Node {
	if ll.current == nil {
		return nil
	}
	current := ll.current
	ll.current = ll.current.next
	return current
}

// ResetIterator resets the iterator to the beginning.
func (ll *LinkedList) ResetIterator() {
	ll.current = ll.head
}

// Len returns the length of the list.
func (ll *LinkedList) Len() int {
	return ll.len
}

// ToSlice scans all nodes into a slice of the given struct type.
func (ll *LinkedList) ToSlice(destSlice interface{}) error {
	sliceVal := reflect.ValueOf(destSlice)
	if sliceVal.Kind() != reflect.Ptr || sliceVal.Elem().Kind() != reflect.Slice {
		return errors.New("destination must be a pointer to a slice")
	}

	sliceElem := sliceVal.Elem()
	elementType := sliceElem.Type().Elem()

	ll.ResetIterator()
	for node := ll.Next(); node != nil; node = ll.Next() {
		newElement := reflect.New(elementType)
		if err := node.StructScan(newElement.Interface()); err != nil {
			return err
		}
		sliceElem.Set(reflect.Append(sliceElem, newElement.Elem()))
	}

	return nil
}
