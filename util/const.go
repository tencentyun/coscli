package util

const (
	Standard           = "STANDARD"
	StandardIA         = "STANDARD_IA"
	IntelligentTiering = "INTELLIGENT_TIERING"
	Archive            = "ARCHIVE"
	DeepArchive        = "DEEP_ARCHIVE"

	MAZStandard           = "MAZ_STANDARD"
	MAZStandardIA         = "MAZ_STANDARD_IA"
	MAZIntelligentTiering = "MAZ_INTELLIGENT_TIERING"
	MAZArchive            = "MAZ_ARCHIVE"
)

const (
	Package       string = "coscli"
	ChannelSize   int    = 1000
	IncludePrompt        = "--include"
	ExcludePrompt        = "--exclude"
	CheckpointDir        = ".coscli_checkpoint"
)

const (
	CpTypeUpload CpType = iota
	CpTypeDownload
	CpTypeCopy
)
