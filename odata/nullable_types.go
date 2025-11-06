package odata

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"
)

// Int64 representa um int64 que pode ser null
type Int64 struct {
	Val   int64
	Valid bool
}

// NewInt64 cria um novo Int64 válido
func NewInt64(value int64) Int64 {
	return Int64{Val: value, Valid: true}
}

// NullInt64 cria um Int64 null
func NullInt64() Int64 {
	return Int64{Valid: false}
}

// Scan implementa sql.Scanner
func (n *Int64) Scan(value interface{}) error {
	if value == nil {
		n.Valid = false
		return nil
	}
	n.Valid = true
	switch v := value.(type) {
	case int64:
		n.Val = v
	case int:
		n.Val = int64(v)
	case int32:
		n.Val = int64(v)
	default:
		n.Valid = false
	}
	return nil
}

// Value implementa driver.Valuer
func (n Int64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Val, nil
}

// MarshalJSON implementa json.Marshaler
func (n Int64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Val)
}

// UnmarshalJSON implementa json.Unmarshaler
func (n *Int64) UnmarshalJSON(data []byte) error {
	if strings.EqualFold(string(data), "null") {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Val)
}

// String representa uma string que pode ser null
type String struct {
	Val   string
	Valid bool
}

// NewString cria uma nova String válida
func NewString(value string) String {
	return String{Val: value, Valid: true}
}

// NullString cria uma String null
func NullString() String {
	return String{Valid: false}
}

// Scan implementa sql.Scanner
func (n *String) Scan(value interface{}) error {
	if value == nil {
		n.Valid = false
		return nil
	}
	n.Valid = true
	if s, ok := value.(string); ok {
		n.Val = s
	} else if b, ok := value.([]byte); ok {
		n.Val = string(b)
	} else {
		n.Valid = false
	}
	return nil
}

// Value implementa driver.Valuer
func (n String) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Val, nil
}

// MarshalJSON implementa json.Marshaler
func (n String) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Val)
}

// UnmarshalJSON implementa json.Unmarshaler
func (n *String) UnmarshalJSON(data []byte) error {
	if strings.EqualFold(string(data), "null") {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Val)
}

// Bool representa um bool que pode ser null
type Bool struct {
	Val   bool
	Valid bool
}

// NewBool cria um novo Bool válido
func NewBool(value bool) Bool {
	return Bool{Val: value, Valid: true}
}

// NullBool cria um Bool null
func NullBool() Bool {
	return Bool{Valid: false}
}

// Scan implementa sql.Scanner
func (n *Bool) Scan(value interface{}) error {
	if value == nil {
		n.Valid = false
		return nil
	}
	n.Valid = true
	if b, ok := value.(bool); ok {
		n.Val = b
	} else {
		n.Valid = false
	}
	return nil
}

// Value implementa driver.Valuer
func (n Bool) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Val, nil
}

// MarshalJSON implementa json.Marshaler
func (n Bool) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Val)
}

// UnmarshalJSON implementa json.Unmarshaler
func (n *Bool) UnmarshalJSON(data []byte) error {
	if strings.EqualFold(string(data), "null") {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Val)
}

// Time representa um time.Time que pode ser null
type Time struct {
	Val   time.Time
	Valid bool
}

// NewTime cria um novo Time válido
func NewTime(value time.Time) Time {
	return Time{Val: value, Valid: true}
}

// NullTime cria um Time null
func NullTime() Time {
	return Time{Valid: false}
}

// Scan implementa sql.Scanner
func (n *Time) Scan(value interface{}) error {
	if value == nil {
		n.Valid = false
		return nil
	}
	n.Valid = true
	if t, ok := value.(time.Time); ok {
		n.Val = t
	} else {
		n.Valid = false
	}
	return nil
}

// Value implementa driver.Valuer
func (n Time) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Val, nil
}

// MarshalJSON implementa json.Marshaler
func (n Time) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Val)
}

// UnmarshalJSON implementa json.Unmarshaler
func (n *Time) UnmarshalJSON(data []byte) error {
	if strings.EqualFold(string(data), "null") {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Val)
}

// Float64 representa um float64 que pode ser null
type Float64 struct {
	Val   float64
	Valid bool
}

// NewFloat64 cria um novo Float64 válido
func NewFloat64(value float64) Float64 {
	return Float64{Val: value, Valid: true}
}

// NullFloat64 cria um Float64 null
func NullFloat64() Float64 {
	return Float64{Valid: false}
}

// Scan implementa sql.Scanner
func (n *Float64) Scan(value interface{}) error {
	if value == nil {
		n.Valid = false
		return nil
	}
	n.Valid = true
	switch v := value.(type) {
	case float64:
		n.Val = v
	case float32:
		n.Val = float64(v)
	default:
		n.Valid = false
	}
	return nil
}

// Value implementa driver.Valuer
func (n Float64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Val, nil
}

// MarshalJSON implementa json.Marshaler
func (n Float64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Val)
}

// UnmarshalJSON implementa json.Unmarshaler
func (n *Float64) UnmarshalJSON(data []byte) error {
	if strings.EqualFold(string(data), "null") {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Val)
}
