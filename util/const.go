package util

const (
	Standard              = "STANDARD"
	StandardIA            = "STANDARD_IA"
	IntelligentTiering    = "INTELLIGENT_TIERING"
	Archive               = "ARCHIVE"
	DeepArchive           = "DEEP_ARCHIVE"
	MAZStandard           = "MAZ_STANDARD"
	MAZStandardIA         = "MAZ_STANDARD_IA"
	MAZIntelligentTiering = "MAZ_INTELLIGENT_TIERING"
	MAZArchive            = "MAZ_ARCHIVE"
	Cold                  = "COLD"
	MAZCold               = "MAZ_COLD"

	StorageTierArchive     = "ARCHIVE_ACCESS"
	StorageTierDeepArchive = "DEEP_ARCHIEVE_ACCESS"
)

const (
	CommandCP      = "cp"
	CommandSync    = "sync"
	CommandLs      = "ls"
	CommandRm      = "rm"
	CommandRestore = "restore"
)

const (
	TypeSnapshotPath   = "snapshotPath"
	TypeFailOutputPath = "failOutputPath"
	TypeProcessLogPath = "processLogPath"
)

const (
	Version             string = "v1.0.9"
	Package             string = "coscli"
	SchemePrefix        string = "cos://"
	CosSeparator        string = "/"
	IncludePrompt              = "--include"
	ExcludePrompt              = "--exclude"
	ChannelSize         int    = 10000
	MaxSyncNumbers             = 5000000
	MaxDeleteBatchCount int    = 1000
	SnapshotConnector          = "==>"
	OfsMaxRenderNum     int    = 100
	CosServiceDomain    string = "service.cos.myqcloud.com"
)

const (
	CpTypeUpload CpType = iota
	CpTypeDownload
	CpTypeCopy
)

const (
	DU_TYPE_TOTAL          = 1
	DU_TYPE_CATEGORIZATION = 2
)

// 版本控制状态
const (
	VersionStatusSuspended = "Suspended"
	VersionStatusEnabled   = "Enabled"
)

const (
	BucketTypeCos = "COS"
	BucketTypeOfs = "OFS"
)

const (
	TypeSrc  = "src"
	TypeDest = "dest"
)

const (
	ContentTypePolicy    = "policy"
	ContentTypeInventory = "inventory"
)

const (
	SyncTypeUnknown        = "unknown"
	SyncTypeIgnoreExisting = "ignoreExisting"
	SyncTypeUpdate         = "update"
	SyncTypeCrc64          = "crc64"
	SyncTypeSnapshot       = "snapshot"
)
