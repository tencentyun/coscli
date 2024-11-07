package util

import (
	"github.com/olekukonko/tablewriter"
	"github.com/syndtr/goleveldb/leveldb"
	"net/http"
	"os"
)

type Config struct {
	Base    BaseCfg  `yaml:"base"`
	Buckets []Bucket `yaml:"buckets"`
}

type BaseCfg struct {
	SecretID            string `yaml:"secretid"`
	SecretKey           string `yaml:"secretkey"`
	SessionToken        string `yaml:"sessiontoken"`
	Protocol            string `yaml:"protocol"`
	Mode                string `yaml:"mode"`
	CvmRoleName         string `yaml:"cvmrolename"`
	CloseAutoSwitchHost string `yaml:"closeautoswitchhost"`
}

type Bucket struct {
	Name     string `yaml:"name"`
	Alias    string `yaml:"alias"`
	Region   string `yaml:"region"`
	Endpoint string `yaml:"endpoint"`
	Ofs      bool   `yaml:"ofs"`
}
type Param struct {
	SecretID     string
	SecretKey    string
	SessionToken string
	Endpoint     string
	Customized   bool
	Protocol     string
}

type UploadInfo struct {
	Key       string `xml:"Key,omitempty"`
	UploadID  string `xml:"UploadId,omitempty"`
	Initiated string `xml:"Initiated,omitempty"`
}

type fileInfoType struct {
	filePath string
	dir      string
}

type objectInfoType struct {
	prefix       string
	relativeKey  string
	size         int64
	lastModified string
}

type CpType int

type FileOperations struct {
	Operation   Operation
	Monitor     *FileProcessMonitor
	ErrOutput   *ErrOutput
	Config      *Config
	Param       *Param
	SnapshotDb  *leveldb.DB
	CpType      CpType
	Command     string
	DeleteCount int
	BucketType  string
}

type Operation struct {
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
	ErrRetryNum       int
	ErrRetryInterval  int
	OnlyCurrentDir    bool
	DisableAllSymlink bool
	EnableSymlinkDir  bool
	DisableCrc64      bool
	DisableChecksum   bool
	DisableLongLinks  bool
	SnapshotPath      string
	Delete            bool
	BackupDir         string
	Force             bool
}

type ErrOutput struct {
	Path       string
	outputFile *os.File
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

type LsCounter struct {
	TotalLimit int
	RenderNum  int
	Table      *tablewriter.Table
}
