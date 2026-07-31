package util

import (
	"github.com/olekukonko/tablewriter"
	"github.com/syndtr/goleveldb/leveldb"
	"net/http"
	"os"
)

// Config coscli配置文件
type Config struct {
	Base    BaseCfg  `yaml:"base"`
	Buckets []Bucket `yaml:"buckets"`
}

// BaseCfg 基础配置
type BaseCfg struct {
	SecretID                   string `yaml:"secretid"`
	SecretKey                  string `yaml:"secretkey"`
	SessionToken               string `yaml:"sessiontoken"`
	Protocol                   string `yaml:"protocol"`
	ServiceDomain              string `yaml:"servicedomain"`
	Mode                       string `yaml:"mode"`
	CvmRoleName                string `yaml:"cvmrolename"`
	CloseAutoSwitchHost        string `yaml:"closeautoswitchhost"`
	DisableEncryption          string `yaml:"disableencryption"`
	DisableAutoFetchBucketType string `yaml:"disableautofetchbuckettype"`
	Proxy                      string `yaml:"proxy"`
}

// Bucket 桶信息
type Bucket struct {
	Name       string `yaml:"name"`
	Alias      string `yaml:"alias"`
	Region     string `yaml:"region"`
	Endpoint   string `yaml:"endpoint"`
	Ofs        bool   `yaml:"ofs"`
	Customized bool   `yaml:"customized"`
}

// Param 命令传入参数
type Param struct {
	SecretID            string
	SecretKey           string
	SessionToken        string
	Endpoint            string
	Customized          bool
	Protocol            string
	CloseAutoSwitchHost string
	BucketType          string
	Proxy               string
}

// UploadInfo 上传文件信息
type UploadInfo struct {
	Key       string `xml:"Key,omitempty"`
	UploadID  string `xml:"UploadId,omitempty"`
	Initiated string `xml:"Initiated,omitempty"`
}

// commonInfoType
type commonInfoType struct {
	key              string
	dir              string
	size             int64
	lastModifiedUnix int64
	lastModified     string
	isDir            bool
}

// fileInfoType 文件信息
type fileInfoType struct {
	filePath     string
	dir          string
	size         int64
	lastModified int64
	isDir        bool
	skip         bool
}

// objectInfoType cos对象信息
type objectInfoType struct {
	prefix       string
	relativeKey  string
	size         int64
	lastModified string
	skip         bool
}

type CpType int

// FileOperations 文件操作配置
type FileOperations struct {
	Operation            Operation
	Monitor              *FileProcessMonitor
	ErrOutput            *ErrOutput
	ProcessLogger        *ProcessLogger
	Config               *Config
	Param                *Param
	SnapshotDb           *leveldb.DB
	CpType               CpType
	Command              string
	DeleteCount          int
	SyncDeleteObjectInfo SyncDeleteObjectInfo
	BucketType           string
	DstBucketType        string
	OutPutDirName        string
}

// Operation 文件操作参数
type Operation struct {
	Recursive            bool
	Filters              []FilterOptionType
	StorageClass         string
	RateLimiting         float32
	PartSize             int64
	CheckPoint           bool
	ThreadNum            int
	MaxThreadNum         int
	Routines             int
	FailOutput           bool
	FailOutputPath       string
	ProcessLog           bool
	ProcessLogPath       string
	Meta                 Meta
	RetryNum             int
	ErrRetryNum          int
	ErrRetryInterval     int
	OnlyCurrentDir       bool
	DisableAllSymlink    bool
	EnableSymlinkDir     bool
	DisableCrc64         bool
	DisableChecksum      bool
	DisableLongLinks     bool
	LongLinksNums        int
	VersionId            string
	AllVersions          bool
	SnapshotPath         string
	Delete               bool
	BackupDir            string
	Force                bool
	Days                 int
	RestoreMode          string
	Move                 bool
	SkipDir              bool
	Update               bool
	IgnoreExisting       bool
	Acl                  string
	GrantRead            string
	GrantWrite           string
	GrantReadAcp         string
	GrantWriteAcp        string
	GrantFullControl     string
	Tags                 string
	ForbidOverWrite      bool
	ServerSideEncryption string
	SSECustomerAlgo      string
	SSECustomerKey       string
	SSECustomerKeyMD5    string
	IgnoreEmptyFile      bool
}

// ErrOutput 错误输出信息
type ErrOutput struct {
	Path       string
	outputFile *os.File
}

// ProcessLogger 进程日志信息
type ProcessLogger struct {
	Path    string
	logFile *os.File
}

// FilterOptionType 正则规则信息
type FilterOptionType struct {
	name    string
	pattern string
}

// Meta cos元数据信息
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

// LsCounter ls统计信息
type LsCounter struct {
	TotalLimit int
	RenderNum  int
	Table      *tablewriter.Table
}

// SyncDeleteObjectInfo todo
type SyncDeleteObjectInfo struct {
	srcCount  int
	destCount int
}

// ACLSettings todo
type ACLSettings struct {
	ACL              string
	GrantRead        string
	GrantWrite       string
	GrantReadACP     string
	GrantWriteACP    string
	GrantFullControl string
}

// BucketEncryptionSettings todo
type BucketEncryptionSettings struct {
	SSEAlgorithm   string
	KMSMasterKeyID string
	KMSAlgorithm   string
}

// BucketInventorySettings todo
type BucketInventorySettings struct {
	ID             string
	KMSMasterKeyID string
	KMSAlgorithm   string
}
