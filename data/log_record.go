package data

type LogRecordPos struct {
	//文件id,文件里面的偏移量
	Fid    uint32
	Offset int64
}

type LogRecord struct {
	Key   []byte
	Value []byte
	Type  LogRecordType
}

type LogRecordType = byte

const (
	LogRecordNormal LogRecordType = iota
	LogRecordDeleted
)

func EncodeLogRecord(logRecord *LogRecord) ([]byte, int64) {

	return nil, 0

}
