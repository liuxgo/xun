package schema

import (
	"github.com/yaoapp/xun/grammar"
)

// NewTable create a new blueprint intance
func NewTable(name string, builder *Builder) *Table {
	table := &Table{
		Table:     grammar.NewTable(name, builder.Conn.WriteConfig.DBName()),
		Builder:   builder,
		ColumnMap: map[string]*Column{},
		IndexMap:  map[string]*Index{},
	}
	table.onChange("NewTable", name, builder)
	return table
}

// Column get the column instance of the table, if the column does not exist create
func (table *Table) Column(name string) *Column {
	column, has := table.ColumnMap[name]
	if has {
		return column
	}
	return table.NewColumn(name)
}

// NewIndex Create a new index instance
func (table *Table) NewIndex(name string, columns ...*Column) *Index {
	cols := []*grammar.Column{}
	for _, column := range columns {
		cols = append(cols, &column.Column)
	}
	index := &Index{
		Index: table.Table.NewIndex(name, cols...),
		Table: table,
	}

	// mapping index
	for _, column := range columns {
		column.Indexes = append(column.Indexes, &index.Index)
	}
	return index
}

// NewColumn Create a new column instance
func (table *Table) NewColumn(name string) *Column {
	column := &Column{
		Column: table.Table.NewColumn(name),
		Table:  table,
	}
	return column
}

// GetIndex get the index instance for the given name, create if not exists.
func (table *Table) GetIndex(name string) *Index {
	index, has := table.IndexMap[name]
	if !has {
		index = table.NewIndex(name)
	}
	return index
}

// GetColumn get the column instance for the given name, create if not exists.
func (table *Table) GetColumn(name string) *Column {
	column, has := table.ColumnMap[name]
	if !has {
		column = table.NewColumn(name)
	}
	return column
}

// PushColumn add a column to the table
func (table *Table) PushColumn(column *Column) *Column {
	table.Table.PushColumn(&column.Column)
	table.ColumnMap[column.Name] = column
	table.onChange("PushColumn", column)
	return column
}

// PushIndex add an index to the table
func (table *Table) PushIndex(index *Index) *Index {
	table.Table.PushIndex(&index.Index)
	table.IndexMap[index.Name] = index
	table.onChange("PushIndex", index)
	return index
}

// AddCommand Add a new command to the table.
func (table *Table) AddCommand(name string, params ...interface{}) {
	table.Table.AddCommand(name, params...)
}

// AddColumnCommand add a new command that adding a column
func (table *Table) AddColumnCommand(column *grammar.Column) {
	table.AddCommand("AddColumn", column)
}

// ModifyColumnCommand add a new command that modifing a column
func (table *Table) ModifyColumnCommand(column *grammar.Column) {
	table.AddCommand("ModifyColumn", column)
}

// RenameColumnCommand add a new command that renaming a column
func (table *Table) RenameColumnCommand(old string, new string) {
	table.AddCommand("RenameColumn", old, new)
}

// DropColumnCommand add a new command that dropping a column
func (table *Table) DropColumnCommand(name string) {
	table.AddCommand("DropColumn", name)
}

// CreateIndexCommand add a new command that creating a index
func (table *Table) CreateIndexCommand(index *grammar.Index) {
	table.AddCommand("CreateIndex", index)
}

// DropIndexCommand add a new command that dropping a index
func (table *Table) DropIndexCommand(name string) {
	table.AddCommand("DropIndex", name)
}

// RenameIndexCommand add a new command that renaming a index
func (table *Table) RenameIndexCommand(old string, new string) {
	table.AddCommand("RenameColumn", old, new)
}

// onChange call this when the table changed
func (table *Table) onChange(event string, args ...interface{}) {
}

// GrammarTable translate type to the grammar table.
// func (table *Table) GrammarTable() *grammar.Table {
// 	gtable := &table.Table
// 	return gtable
// db := table.Builder.Conn.WriteConfig.DBName()
// gtable := &grammar.Table{
// 	DBName:    db,
// 	Name:      table.Name,
// 	Comment:   table.Comment,
// 	Columns:   []*grammar.Column{},
// 	Indexes:   []*grammar.Index{},
// 	Collation: table.Builder.Conn.Config.Collation,
// 	Charset:   table.Builder.Conn.Config.Charset,
// }

// // translate columns
// ColumnMap := sync.Map{}
// for _, column := range table.Columns {
// 	Column := &grammar.Column{
// 		DBName:            db,
// 		TableName:         table.Name,
// 		Name:              column.Name,
// 		Type:              column.Type,
// 		Length:            column.Length,
// 		Precision:         column.Precision(),
// 		Scale:             column.Scale(),
// 		DatetimePrecision: column.DatetimePrecision(),
// 		Comment:           column.Comment,
// 		Collation:         table.Builder.Conn.Config.Collation,
// 		Charset:           table.Builder.Conn.Config.Charset,
// 		Indexes:           []*grammar.Index{},
// 	}
// 	gtable.Columns = append(gtable.Columns, Column)
// 	ColumnMap.Store(column.Name, Column)
// }

// // translate indexes
// for _, index := range table.Indexes {
// 	gindex := &grammar.Index{
// 		DBName:    db,
// 		TableName: table.Name,
// 		Name:      index.Name,
// 		Type:      index.Type,
// 		Comment:   index.Comment,
// 		Columns:   []*grammar.Column{},
// 	}
// 	// bind columns and indexes
// 	for _, column := range index.Columns {
// 		Column, has := ColumnMap.Load(column.Name)
// 		if has {
// 			gindex.Columns = append(gindex.Columns, Column.(*grammar.Column))
// 			Column.(*grammar.Column).Indexes = append(Column.(*grammar.Column).Indexes, gindex)
// 		}
// 	}
// 	gtable.Indexes = append(gtable.Indexes, gindex)
// }

// gtable.SetDefaultEngine("InnoDB")
// gtable.SetDefaultCollation("utf8mb4_unicode_ci")
// gtable.SetDefaultCharset("utf8mb4")
// return gtable
// }

// func (table *Table) sqlColumns() string {
// 	// SELECT *
// 	// FROM INFORMATION_SCHEMA.COLUMNS
// 	// WHERE TABLE_SCHEMA = 'test' AND TABLE_NAME ='products';
// 	Columns := []string{
// 		"TABLE_SCHEMA AS `db_name`",
// 		"TABLE_NAME AS `table_name`",
// 		"COLUMN_NAME AS `Column`",
// 		"ORDINAL_POSITION AS `position`",
// 		"COLUMN_DEFAULT AS `default`",
// 		"IS_NULLABLE as `nullable`",
// 		"DATA_TYPE as `type`",
// 		"CHARACTER_MAXIMUM_LENGTH as `length`",
// 		"CHARACTER_OCTET_LENGTH as `octet_length`",
// 		"NUMERIC_PRECISION as `precision`",
// 		"NUMERIC_SCALE as `scale`",
// 		"DATETIME_PRECISION as `datetime_precision`",
// 		"CHARACTER_SET_NAME as `character`",
// 		"COLLATION_NAME as `collation`",
// 		"COLUMN_KEY as `key`",
// 		"EXTRA as `extra`",
// 		"COLUMN_COMMENT as `comment`",
// 	}
// 	sql := `
// 		SELECT %s
// 		FROM INFORMATION_SCHEMA.COLUMNS
// 		WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME ='%s';
// 	`
// 	cfg := table.Builder.Conn.WriteConfig
// 	fmt.Printf("sqlColumns: %#v\n", cfg.Sqlite3DBName())
// 	sql = fmt.Sprintf(sql, strings.Join(Columns, ","), "xiang", table.Name)
// 	fmt.Printf("%s", sql)
// 	return sql
// }