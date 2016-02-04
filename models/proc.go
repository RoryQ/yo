package models

import "database/sql"

// Proc represents an Proc.
type Proc struct {
	Name           string
	ParameterTypes string
	ReturnType     string

	// extra
	Schema           string
	FuncName         string
	GoParameterTypes []string
	GoReturnType     string
	GoNilReturnType  string
}

// ProcsBySchema returns enums from the database for the specified schema.
func ProcsBySchema(db *sql.DB, schema string) ([]*Proc, error) {
	// sql query
	const sqlstr = `SELECT p.proname, oidvectortypes(p.proargtypes), pg_get_function_result(p.oid) ` +
		`FROM pg_proc p ` +
		`INNER JOIN pg_namespace n ON (p.pronamespace = n.oid) ` +
		`WHERE n.nspname = $1 `

	// run query
	q, err := db.Query(sqlstr, schema)
	if err != nil {
		return nil, err
	}
	defer q.Close()

	// load results
	res := []*Proc{}
	for q.Next() {
		p := Proc{}

		// scan
		err = q.Scan(
			&p.Name, &p.ParameterTypes, &p.ReturnType,
		)

		// check err
		if err != nil {
			return nil, err
		}

		res = append(res, &p)
	}

	return res, nil
}