package dtypes

type DataType string

type DecimalValue string

const (
	Int64       DataType = "int64"
	Float64     DataType = "float64"
	String      DataType = "string"
	Boolean     DataType = "bool"
	Datetime    DataType = "datetime"
	Decimal     DataType = "decimal"
	Categorical DataType = "categorical"
	Enum        DataType = "enum"
	List        DataType = "list"
	Struct      DataType = "struct"
	Binary      DataType = "binary"
	Duration    DataType = "duration"
)
