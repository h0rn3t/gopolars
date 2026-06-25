package polars

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/h0rn3t/gopolars/pkg/exec"
	"github.com/h0rn3t/gopolars/pkg/frame"
	icsv "github.com/h0rn3t/gopolars/pkg/io/csv"
	iipc "github.com/h0rn3t/gopolars/pkg/io/ipc"
	ijson "github.com/h0rn3t/gopolars/pkg/io/json"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

type IO interface {
	ScanCSV(input ScanCSVInput) (LazyFrame, error)
	ScanParquet(input ScanParquetInput) (LazyFrame, error)
	ScanIPC(input ScanIPCInput) (LazyFrame, error)
	ScanJSON(input ScanJSONInput) (LazyFrame, error)
	ReadCSV(input ReadCSVInput) (DataFrame, error)
	ReadParquet(input ReadParquetInput) (DataFrame, error)
	ReadIPC(input ReadIPCInput) (DataFrame, error)
	ReadJSON(input ReadJSONInput) (DataFrame, error)
}

type ioFacade struct{}

func NewIO() IO {
	return ioFacade{}
}

func (f ioFacade) ScanCSV(input ScanCSVInput) (LazyFrame, error) {
	return &lf{
		source: frame.DataFrame{},
		engine: exec.New(),
		nodes:  []logical.Node{},
		scan: &scanSource{
			format: "csv",
			path:   input.Path,
			csv:    input,
		},
	}, nil
}

func (f ioFacade) ScanParquet(input ScanParquetInput) (LazyFrame, error) {
	return &lf{
		source: frame.DataFrame{},
		engine: exec.New(),
		nodes:  []logical.Node{},
		scan: &scanSource{
			format: "parquet",
			path:   input.Path,
			parq:   input,
		},
	}, nil
}

func (f ioFacade) ScanIPC(input ScanIPCInput) (LazyFrame, error) {
	return &lf{
		source: frame.DataFrame{},
		engine: exec.New(),
		nodes:  []logical.Node{},
		scan: &scanSource{
			format: "ipc",
			path:   input.Path,
			ips:    input,
		},
	}, nil
}

func (f ioFacade) ScanJSON(input ScanJSONInput) (LazyFrame, error) {
	return &lf{
		source: frame.DataFrame{},
		engine: exec.New(),
		nodes:  []logical.Node{},
		scan: &scanSource{
			format: "json",
			path:   input.Path,
			json:   input,
		},
	}, nil
}

func (f ioFacade) ReadCSV(input ReadCSVInput) (DataFrame, error) {
	path, err := resolveObjectStorePath(input.Path)
	if err != nil {
		return nil, err
	}
	return fromFrame(icsv.Read(icsv.ReadInput{
		Path:      path,
		HasHeader: input.HasHeader,
		Separator: input.Separator,
		Schema:    input.Schema,
	}))
}

func (f ioFacade) ReadParquet(input ReadParquetInput) (DataFrame, error) {
	path, err := resolveObjectStorePath(input.Path)
	if err != nil {
		return nil, err
	}
	df, err := readParquetSource(path, input.Columns, nil)
	return fromFrame(df, err)
}

func (f ioFacade) ReadIPC(input ReadIPCInput) (DataFrame, error) {
	path, err := resolveObjectStorePath(input.Path)
	if err != nil {
		return nil, err
	}
	return fromFrame(iipc.Read(iipc.ReadInput{Path: path}))
}

func (f ioFacade) ReadJSON(input ReadJSONInput) (DataFrame, error) {
	path, err := resolveObjectStorePath(input.Path)
	if err != nil {
		return nil, err
	}
	return fromFrame(ijson.Read(ijson.ReadInput{
		Path:   path,
		NDJSON: input.NDJSON,
		Schema: input.Schema,
	}))
}

func resolveObjectStorePath(path string) (string, error) {
	switch {
	case strings.HasPrefix(path, "s3://"):
		return mapObjectStorePath(path, "s3://", "GOPOLARS_S3_ROOT")
	case strings.HasPrefix(path, "gcs://"):
		return mapObjectStorePath(path, "gcs://", "GOPOLARS_GCS_ROOT")
	case strings.HasPrefix(path, "az://"):
		return mapObjectStorePath(path, "az://", "GOPOLARS_AZURE_ROOT")
	default:
		return path, nil
	}
}

func mapObjectStorePath(path string, prefix string, envKey string) (string, error) {
	root := os.Getenv(envKey)
	if root == "" {
		return "", fmt.Errorf("%s is not configured", envKey)
	}
	trimmed := strings.TrimPrefix(path, prefix)
	trimmed = strings.TrimPrefix(trimmed, "/")
	return filepath.Join(root, trimmed), nil
}
