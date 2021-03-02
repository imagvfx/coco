package sqlite

import "fmt"

type Where struct {
	keys []string
	vals []interface{}
}

func NewWhere() *Where {
	return &Where{
		keys: make([]string, 0),
		vals: make([]interface{}, 0),
	}
}

func (w *Where) Add(k string, v interface{}) {
	w.keys = append(w.keys, k)
	w.vals = append(w.vals, v)
}

func (w *Where) Stmt() string {
	if len(w.keys) == 0 {
		return ""
	}
	stmt := " WHERE"
	for i, k := range w.keys {
		if i != 0 {
			stmt += " AND"
		}
		stmt += fmt.Sprintf(" %v = ?", k)
	}
	return stmt
}

func (w *Where) Vals() []interface{} {
	return w.vals
}
