package query

import (
	"fmt"
	"reflect"

	"github.com/yaoapp/xun/dbal"
	"github.com/yaoapp/xun/utils"
)

// Where Add a basic where clause to the query.
func (builder *Builder) Where(column interface{}, args ...interface{}) Query {

	columnKind := reflect.TypeOf(column).Kind()

	// Here we will make some assumptions about the operator. If only 2 values are
	// passed to the method, we will assume that the operator is an equals sign
	// and keep going. Otherwise, we'll require the operator to be passed in.
	operator, value, boolean, offset := builder.prepareArgs(args...)

	// Where([][]interface{}{ {"score", ">", 64.56},{"vote", 10}})
	// If the column is an array, we will assume it is an array of key-value pairs
	// and can add them each as a where clause. We will maintain the boolean we
	// received when the method was called and pass it into the Wheres attribute.
	if columnKind == reflect.Array || columnKind == reflect.Slice {
		builder.addArrayOfWheres(column, boolean)
		return builder
	}

	// Where( func(qb Query){  qb.Where("name", "Ken")... })
	// Where( func(qb Query){  qb.Where("name", "Ken")... }, "and")
	// Where( func(qb Query){  qb.Where("name", "Ken")... }, "or")
	// If the columns is actually a Closure instance, we will assume the developer
	// wants to begin a nested where statement which is wrapped in parenthesis.
	// We'll add that Closure to the query then return back out immediately.
	if builder.isClosure(column) && (len(args) == 0 || (len(args) >= 1 && builder.isOperator(args[0]))) {
		callback := column.(func(Query))
		boolean := "and"
		if len(args) == 1 && reflect.TypeOf(args[0]).Kind() == reflect.String {
			boolean = args[0].(string)
		}
		builder.whereNested(callback, boolean)
		return builder
	}

	// Where( func(qb Query){ qb.Where("name", "Ken")... }, 5)
	// Where( func(qb Query){ qb.Where("name", "Ken")... }, 5, "or")
	// Where( func(qb Query){ qb.Where("name", "Ken")... }, ">", 5)
	// Where( func(qb Query){ qb.Where("name", "Ken")... }, ">", 5, "or")
	// If the column is a Closure instance and there is an operator value, we will
	// assume the developer wants to run a subquery and then compare the result
	// of that subquery with the given value that was provided to the method.
	if builder.isQueryable(column) && len(args) >= 1 {
		sub, bindings, whereOffset := builder.createSub(column)
		builder.Query.AddBinding("where", bindings)
		builder.Where(dbal.Raw(fmt.Sprintf("(%s)", sub)), operator, value, boolean, whereOffset)
		return builder
	}

	// Where("vote", '>', func(sub Query) {
	// 		sub.From("table_test_where").
	// 			Where("score", ">", 5).
	// 	 		Sum()
	// })
	// If the value is a Closure, it means the developer is performing an entire
	// sub-select within the query and we will need to compile the sub-select
	// within the where clause to get the appropriate query record results.
	if columnKind == reflect.String && builder.isClosure(value) {
		callback := value.(func(Query))
		return builder.whereSub(column.(string), operator, callback, boolean)
	}

	// If the value is "null", we will just assume the developer wants to add a
	// where null clause to the query. So, we will allow a short-cut here to
	// that method for convenience so the developer doesn't have to check.
	if utils.IsNil(value) {
		return builder.WhereNull(column, boolean, operator != "=")
	}

	queryType := "basic"

	// If the column is making a JSON reference we'll check to see if the value
	// is a boolean. If it is, we'll add the raw boolean string as an actual
	// value to the query to ensure this is properly handled by the query.
	//  if (Str::contains($column, '->') && is_bool($value)) {

	// Where("email", "like", "%@yao.run")
	// Now that we are working with just a simple query we can put the elements
	// in our array and add the query binding to our array of bindings that
	// will be bound to each SQL statements when it is finally executed.
	builder.Query.Wheres = append(builder.Query.Wheres, dbal.Where{
		Type:     queryType,
		Column:   column,
		Operator: operator,
		Boolean:  boolean,
		Value:    value,
		Offset:   offset,
	})
	if !builder.isExpression(value) {
		builder.Query.AddBinding("where", builder.flattenValue(value))
	}
	return builder
}

// WhereColumn Add a "where" clause comparing two columns to the query.
func (builder *Builder) WhereColumn(first interface{}, args ...interface{}) Query {

	columnKind := reflect.TypeOf(first).Kind()

	// Here we will make some assumptions about the operator. If only 2 values are
	// passed to the method, we will assume that the operator is an equals sign
	// and keep going. Otherwise, we'll require the operator to be passed in.
	operator, second, boolean, offset := builder.prepareArgs(args...)

	// Where([][]interface{}{ {"score", ">", 64.56},{"vote", 10}})
	// If the column is an array, we will assume it is an array of key-value pairs
	// and can add them each as a where clause. We will maintain the boolean we
	// received when the method was called and pass it into the Wheres attribute.
	if columnKind == reflect.Array || columnKind == reflect.Slice {
		builder.addArrayOfWheres(first, boolean)
		return builder
	}

	// Finally, we will add this where clause into this array of clauses that we
	// are building for the query. All of them will be compiled via a grammar
	// once the query is about to be executed and run against the database.
	builder.Query.Wheres = append(builder.Query.Wheres, dbal.Where{
		Type:     "column",
		Second:   second,
		First:    first,
		Operator: operator,
		Boolean:  boolean,
		Offset:   offset,
	})

	return builder
}

// OrWhereColumn Add an "or where" clause comparing two columns to the query.
func (builder *Builder) OrWhereColumn() {
}

// Where([][]interface{}{ {"score", ">", 64.56},{"vote", 10},})
// addArrayOfWheres Add an array of where clauses to the query.
func (builder *Builder) addArrayOfWheres(inputColumns interface{}, boolean string) *Builder {

	switch inputColumns.(type) {
	case [][]interface{}:
		columns := inputColumns.([][]interface{})
		return builder.whereNested(func(qb Query) {
			for _, args := range columns {
				if len(args) > 1 && reflect.TypeOf(args[0]).Kind() == reflect.String {
					column := args[0].(string)
					operator, value, boolean, offset := builder.prepareArgs(args[1:]...)
					qb.Where(column, operator, value, boolean, offset)
				}
			}
		}, boolean)
	}

	return builder
}

// createSub Creates a subquery and parse it.
func (builder *Builder) createSub(subquery interface{}) (string, []interface{}, int) {
	if builder.isClosure(subquery) {
		callback := subquery.(func(qb Query))
		qb := builder.forSubQuery()
		callback(qb)
		subquery = qb
	}
	return builder.parseSub(subquery)
}

// Parse the subquery into SQL and bindings.
func (builder *Builder) parseSub(subquery interface{}) (string, []interface{}, int) {

	switch subquery.(type) {
	case *Builder:
		qb := builder.prependDatabaseNameIfCrossDatabaseQuery(subquery.(*Builder))
		offset := len(builder.GetBindings())
		bindings := qb.GetBindings()
		whereOffset := offset + len(utils.Flatten(bindings))
		return builder.Grammar.CompileSelectOffset(qb.Query, &offset), bindings, whereOffset
	case dbal.Expression:
		return subquery.(dbal.Expression).GetValue(), []interface{}{}, 0
	case string:
		return subquery.(string), []interface{}{}, 0
	}

	panic(fmt.Errorf("a subquery must be a query builder instance, a Closure, or a string"))
}

// prependDatabaseNameIfCrossDatabaseQuery Prepend the database name if the given query is on another database.
func (builder *Builder) prependDatabaseNameIfCrossDatabaseQuery(subquery *Builder) *Builder {
	// if (strpos($query->from, $databaseName) !== 0 && strpos($query->from, '.') === false) {
	// 	$query->from($databaseName.'.'.$query->from);
	// }
	return subquery
}

// whereSub Add a full sub-select to the query.
func (builder *Builder) whereSub(column string, operator string, callback func(qb Query), boolean string) *Builder {
	new := builder.forSubQuery()
	callback(new)
	builder.Query.Wheres = append(builder.Query.Wheres, dbal.Where{
		Type:     "sub",
		Column:   column,
		Operator: operator,
		Query:    new.Query,
		Boolean:  boolean,
	})
	builder.Query.AddBinding("where", new.Query.Bindings["where"])
	return builder
}

// forSubQuery Create a new query instance for a sub-query.
func (builder *Builder) forSubQuery() *Builder {
	new := builder.new()
	return new
}

// whereNested  Add a nested where statement to the query.
func (builder *Builder) whereNested(callback func(qb Query), boolean string) *Builder {
	new := builder.forNestedWhere()
	callback(new)
	return builder.addNestedWhereQuery(new.Query, boolean)
}

// Add another query builder as a nested where to the query builder.
func (builder *Builder) addNestedWhereQuery(query *dbal.Query, boolean string) *Builder {

	if len(query.Wheres) > 0 {
		builder.Query.Wheres = append(builder.Query.Wheres, dbal.Where{
			Type:    "nested",
			Query:   query,
			Boolean: boolean,
		})
		builder.Query.AddBinding("where", query.Bindings["where"])
	}

	return builder
}

// forNestedWhere Create a new query instance for nested where condition.
func (builder *Builder) forNestedWhere() *Builder {
	new := builder.new()
	new.Query.From = builder.Query.From
	return new
}

// OrWhere Add an "or where" clause to the query.
func (builder *Builder) OrWhere() {
}

// WhereJSONContains Add a "where JSON contains" clause to the query.
func (builder *Builder) WhereJSONContains() {
}

// OrWhereJSONContains Add an "or where JSON contains" clause to the query.
func (builder *Builder) OrWhereJSONContains() {
}

// WhereJSONDoesntContain Add a "where JSON not contains" clause to the query.
func (builder *Builder) WhereJSONDoesntContain() {
}

// OrWhereJSONDoesntContain Add an "or where JSON not contains" clause to the query.
func (builder *Builder) OrWhereJSONDoesntContain() {
}

// WhereJSONLength Add a "where JSON length" clause to the query.
func (builder *Builder) WhereJSONLength() {
}

// OrWhereJSONLength Add an "or where JSON length" clause to the query.
func (builder *Builder) OrWhereJSONLength() {
}

// WhereBetween Add a where between statement to the query.
func (builder *Builder) WhereBetween() {
}

// OrWhereBetween Add an or where between statement to the query.
func (builder *Builder) OrWhereBetween() {
}

// WhereNotBetween Add a where not between statement to the query.
func (builder *Builder) WhereNotBetween() {
}

// OrWhereNotBetween Add an or where not between statement using columns to the query.
func (builder *Builder) OrWhereNotBetween() {
}

// WhereIn Add a "where in" clause to the query.
func (builder *Builder) WhereIn() {
}

// OrWhereIn Add an "or where in" clause to the query.
func (builder *Builder) OrWhereIn() {
}

// WhereNotIn Add a "where not in" clause to the query.
func (builder *Builder) WhereNotIn() {
}

// OrWhereNotIn Add an "or where not in" clause to the query.
func (builder *Builder) OrWhereNotIn() {
}

// WhereNull Add a "where null" clause to the query.
func (builder *Builder) WhereNull(column interface{}, args ...interface{}) Query {
	_, not, boolean, _ := builder.prepareArgs(args...)
	typ := "null"
	if !utils.IsNil(not) && reflect.TypeOf(not).Kind() == reflect.Bool {
		if reflect.ValueOf(not).Bool() {
			typ = "notnull"
		}
	}

	reflectColumn := reflect.ValueOf(column)
	columnKind := reflectColumn.Kind()

	columns := []interface{}{}
	if columnKind == reflect.Array || columnKind == reflect.Slice {
		reflectColumn = reflect.Indirect(reflectColumn)
		for i := 0; i < reflectColumn.Len(); i++ {
			if reflectColumn.Index(i).Kind() == reflect.String {
				columns = append(columns, reflectColumn.Index(i).String())
			}
		}
	} else if columnKind == reflect.String {
		columns = append(columns, reflectColumn.String())
	}

	for _, col := range columns {
		builder.Query.Wheres = append(builder.Query.Wheres, dbal.Where{
			Column:  col,
			Type:    typ,
			Boolean: boolean,
		})
	}
	return builder
}

// OrWhereNull Add an "or where null" clause to the query.
func (builder *Builder) OrWhereNull(column interface{}) Query {
	return builder.WhereNull(column, "or")
}

// WhereNotNull Add a "where not null" clause to the query.
func (builder *Builder) WhereNotNull(column interface{}, args ...interface{}) Query {
	boolean, _, _, _ := builder.prepareArgs(args...)
	return builder.WhereNull(column, boolean, true)
}

// OrWhereNotNull Add an "or where not null" clause to the query.
func (builder *Builder) OrWhereNotNull(column interface{}) Query {
	return builder.WhereNull(column, "or", true)
}

// WhereDate Add a "where date" statement to the query.
func (builder *Builder) WhereDate() {
}

// OrWhereDate Add an "or where date" statement to the query.
func (builder *Builder) OrWhereDate() {
}

// WhereYear Add a "where year" statement to the query.
func (builder *Builder) WhereYear() {
}

// OrWhereYear Add an "or where year" statement to the query.
func (builder *Builder) OrWhereYear() {
}

// WhereMonth Add a "where month" statement to the query.
func (builder *Builder) WhereMonth() {
}

// OrWhereMonth Add an "or where month" statement to the query.
func (builder *Builder) OrWhereMonth() {
}

// WhereDay Add a "where day" statement to the query.
func (builder *Builder) WhereDay() {
}

// OrWhereDay Add an "or where day" statement to the query.
func (builder *Builder) OrWhereDay() {
}

// WhereTime Add a "where time" statement to the query.
func (builder *Builder) WhereTime() {
}

// OrWhereTime Add an "or where time" statement to the query.
func (builder *Builder) OrWhereTime() {
}

// WhereExists Add an exists clause to the query.
func (builder *Builder) WhereExists() {
}

// OrWhereExists Add an or exists clause to the query.
func (builder *Builder) OrWhereExists() {
}

// WhereNotExists  Add a where not exists clause to the query.
func (builder *Builder) WhereNotExists() {
}

// OrWhereNotExists Add a where not exists clause to the query.
func (builder *Builder) OrWhereNotExists() {
}

// WhereRaw Add a basic where clause to the query.
func (builder *Builder) WhereRaw(sql string, bindings ...interface{}) Query {
	return builder.whereRaw(sql, bindings, "and")
}

// OrWhereRaw Add an "or where" clause to the query.
func (builder *Builder) OrWhereRaw(sql string, bindings ...interface{}) Query {
	return builder.whereRaw(sql, bindings, "or")
}

func (builder *Builder) whereRaw(sql string, bindings []interface{}, boolean string) Query {
	builder.Query.Wheres = append(builder.Query.Wheres, dbal.Where{
		Type:    "raw",
		SQL:     sql,
		Boolean: boolean,
	})
	builder.Query.AddBinding("where", bindings)
	return builder
}
