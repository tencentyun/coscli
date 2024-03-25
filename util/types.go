package util

import (
	"net/http"
	"os"
)

type CpType int

type CopyCommand struct {
	CpParams  CpParams
	Monitor   *CpProcessMonitor
	ErrOutput *ErrOutput
	Config    *Config
	Param     *Param
}

type CpParams struct {
	Recursive         bool
	Filters           []FilterOptionType
	StorageClass      string
	RateLimiting      float32
	PartSize          int64
	ThreadNum         int
	Routines          int
	FailOutput        bool
	FailOutputPath    string
	Meta              Meta
	RetryNum          int
	OnlyCurrentDir    bool
	DisableAllSymlink bool
	EnableSymlinkDir  bool
	CheckpointDir     string
	DisableCrc64      bool
}

type ErrOutput struct {
	Path         string
	ListOutput   *os.File
	UploadOutput *os.File
}

type FilterOptionType struct {
	name    string
	pattern string
}

type Meta struct {
	CacheControl       string
	ContentDisposition string
	ContentEncoding    string
	ContentType        string
	ContentMD5         string
	ContentLength      int64
	ContentLanguage    string
	Expires            string
	// 自定义的 x-cos-meta-* header
	XCosMetaXXX *http.Header
	MetaChange  bool
}
