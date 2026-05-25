package polars

import (
	"fmt"

	iarrow "github.com/eugeneshershen/gopolars/pkg/io/arrow"
)

// ErrPySetItemNotSupported відповідає відсутності Python __setitem__ у статичному Go API.
var ErrPySetItemNotSupported = fmt.Errorf("gopolars: Python __setitem__ is not supported; use ReplaceColumn, WithColumns, Hstack, or Set on Series")

// PyArrayDataFrame відповідає Python DataFrame.__array__ (NumPy-совместимый снимок).
func PyArrayDataFrame(df DataFrame) [][]any {
	if df == nil {
		return nil
	}
	return df.ToNumpy()
}

// PyArraySeries відповідає Python Series.__array__.
func PyArraySeries(s Series) []any {
	if s == nil {
		return nil
	}
	return s.ToNumpy()
}

// PyArrowTableDataFrame відповідає DataFrame.to_arrow() / __arrow_c_stream__ (табличний знімок).
func PyArrowTableDataFrame(df DataFrame) (iarrow.Table, error) {
	if df == nil {
		return iarrow.Table{}, fmt.Errorf("nil DataFrame")
	}
	return df.ToArrow(ToArrowInput{})
}

// PyArrowTableSeries відповідає Series.to_arrow().
func PyArrowTableSeries(s Series) (iarrow.Table, error) {
	if s == nil {
		return iarrow.Table{}, fmt.Errorf("nil Series")
	}
	return s.ToArrow()
}

// PyDataFrameInterchange повертає той самий DataFrame як interchange-compatible view (Python __dataframe__ 0.19+).
func PyDataFrameInterchange(df DataFrame) DataFrame {
	return df
}
