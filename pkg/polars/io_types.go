package polars

import "github.com/h0rn3t/gopolars/pkg/dtypes"

type JoinType string

const (
	JoinTypeInner JoinType = "inner"
	JoinTypeLeft  JoinType = "left"
	JoinTypeRight JoinType = "right"
	JoinTypeFull  JoinType = "full"
	JoinTypeSemi  JoinType = "semi"
	JoinTypeAnti  JoinType = "anti"
	JoinTypeCross JoinType = "cross"
	JoinTypeAsof  JoinType = "asof"
)

type JoinInput struct {
	Other         DataFrame
	LeftOn        []string
	RightOn       []string
	How           JoinType
	Suffix        string
	AsofDirection string
	AsofTolerance int64
}

type SortInput struct {
	By            []string
	Descending    []bool
	NullsLast     bool
	MaintainOrder bool
}

type ConcatInput struct {
	Others []DataFrame
	How    string
}

type ScanCSVInput struct {
	Path              string
	HasHeader         bool
	Separator         rune
	Schema            dtypes.Schema
	InferSchemaLength int
	IgnoreErrors      bool
}

type ScanParquetInput struct {
	Path    string
	Columns []string
}

type ScanIPCInput struct {
	Path string
}

type ScanJSONInput struct {
	Path   string
	NDJSON bool
	Schema dtypes.Schema
}

type ReadCSVInput struct {
	Path      string
	HasHeader bool
	Separator rune
	Schema    dtypes.Schema
}

type ReadParquetInput struct {
	Path    string
	Columns []string
}

type ReadIPCInput struct {
	Path string
}

type ReadJSONInput struct {
	Path   string
	NDJSON bool
	Schema dtypes.Schema
}

type WriteCSVInput struct {
	Path          string
	IncludeHeader bool
	Separator     rune
}

type WriteJSONInput struct {
	Path   string
	Pretty bool
	NDJSON bool
}

type WriteParquetInput struct {
	Path         string
	Compression  string
	RowGroupSize int
}

type WriteIPCInput struct {
	Path string
}

// IfTableExists керує поведінкою write_database, коли цільова таблиця вже існує.
// Відповідає полю if_table_exists у Python polars (fail/append/replace).
type IfTableExists string

const (
	// IfTableExistsFail — типове значення: помилка, якщо таблиця вже існує.
	IfTableExistsFail IfTableExists = "fail"
	// IfTableExistsAppend — створити таблицю за відсутності, інакше дописати рядки.
	IfTableExistsAppend IfTableExists = "append"
	// IfTableExistsReplace — перестворити таблицю під DataFrame.
	IfTableExistsReplace IfTableExists = "replace"
)

// WriteDatabaseInput описує запис DataFrame у зовнішню SQL-базу через ADBC-двигун
// (pkg/io/database). Дзеркалить polars.DataFrame.write_database. Conn — це вже
// відкрите adbc.Connection (carried as any); інакше з'єднання відкривається за
// DriverName + DriverOptions через ADBC driver manager (CGO).
type WriteDatabaseInput struct {
	TableName     string
	IfTableExists IfTableExists
	Conn          any
	DriverName    string
	DriverOptions map[string]string
	BatchSize     int
}

// ReadDatabaseInput описує виконання SQL-запиту проти ADBC-з'єднання та побудову
// DataFrame з результату. Дзеркалить polars.read_database.
type ReadDatabaseInput struct {
	Query         string
	Conn          any
	DriverName    string
	DriverOptions map[string]string
	// BatchSize is advisory on read (the ADBC driver controls result batching);
	// reserved for a future iter-batches form.
	BatchSize int
}

type ToArrowInput struct {
	ChunkSize int
}
