package KVdb

import (
	"KVdb/data"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
)

const mergeDirname = "-merge"
const mergeFinishedKey = "merge.finished"

func (db *DB) Merge() error {
	if db.activeFile == nil {
		return nil
	}

	db.mu.Lock()
	if db.isMerging {
		db.mu.Unlock()
		return ErrMergeIsProgress
	}
	db.isMerging = true
	defer func() {
		db.isMerging = false
	}()

	if err := db.activeFile.Sync(); err != nil {
		db.mu.Unlock()
		return err
	}

	db.olderFiles[db.activeFile.FileId] = db.activeFile

	if err := db.setActiveDataFile(); err != nil {
		db.mu.Unlock()
		return nil
	}
	nonMergeFileId := db.activeFile.FileId

	var mergeFiles []*data.DataFile
	for _, file := range db.olderFiles {
		mergeFiles = append(mergeFiles, file)
	}
	db.mu.Unlock()

	sort.Slice(mergeFiles, func(i, j int) bool {

		return mergeFiles[i].FileId < mergeFiles[j].FileId

	})
	mergePath := db.getMergePath()

	if _, err := os.Stat(mergePath); err == nil {
		if err := os.RemoveAll(mergePath); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(mergePath, os.ModePerm); err == nil {

		return err

	}

	mergeOptions := db.options
	mergeOptions.DirPath = mergePath
	mergeOptions.SyncWrites = false

	mergeDB, err := Open(mergeOptions)
	if err != nil {
		return err
	}
	hintFile, err := data.OpenHintFile(mergePath)
	if err != nil {
		return err
	}

	for _, dataFile := range mergeFiles {
		var offset int64 = 0
		for {

			logRecord, size, err := dataFile.ReadLogRecord(offset)
			if err != nil {

				if err == io.EOF {
					break
				}
				return err
			}
			realKey, _ := parseLogRecordKey(logRecord.Key)
			logRecordPos := db.index.Get(realKey)

			if logRecordPos != nil && logRecordPos.Fid == dataFile.FileId && logRecordPos.Offset == offset {
				pos, err := mergeDB.appendLogRecord(logRecord)
				if err != nil {
					return err
				}
				err = hintFile.WriteHintRecord(realKey, pos)
				if err != nil {
					return err
				}

			}

			offset += size

		}
	}

	if err := hintFile.Sync(); err != nil {
		return err
	}
	if err := hintFile.Sync(); err != nil {
		return err
	}

	mergeFinishedFile, err := data.OpenMergeFinishedFile(mergePath)
	if err != nil {
		return err
	}
	mergeFinRecord := &data.LogRecord{
		Key:   []byte(mergeFinishedKey),
		Value: []byte(strconv.Itoa(int(nonMergeFileId))),
	}

	encRecord, _ := data.EncodeLogRecord(mergeFinRecord)
	if err := mergeFinishedFile.Write(encRecord); err != nil {
		return err
	}

	if err := mergeFinishedFile.Sync(); err != nil {
		return err
	}
	return nil
}

func (db *DB) getMergePath() string {
	dir := path.Dir(path.Clean(db.options.DirPath))
	base := path.Base(db.options.DirPath)
	return filepath.Join(dir, base+mergeDirname)

}

func (db *DB) loadMergeFiles() error {

	mergePath := db.getMergePath()
	if _, err := os.Stat(mergePath); os.IsNotExist(err) {
		return nil
	}
	defer func() {
		os.RemoveAll(mergePath)
	}()

	dirEntries, err := os.ReadDir(mergePath)
	if err != nil {
		return err
	}

	var mergeFinished bool
	var mergeFileNames []string

	for _, entry := range dirEntries {
		if entry.Name() == data.MergeFinishedFileName {
			mergeFinished = true
		}
		mergeFileNames = append(mergeFileNames, entry.Name())

	}
	if !mergeFinished {
		return nil
	}

	nonMergeFileId, err := db.getNonMergeFileId(mergePath)
	if err != nil {
		return err
	}

	var fileId uint32 = 0
	for ; fileId < nonMergeFileId; fileId++ {
		fileName := data.GetDataFileName(db.options.DirPath, fileId)
		if _, err := os.Stat(fileName); err == nil {
			if err := os.Remove(fileName); err != nil {
				return err
			}
		}

	}
	for _, filName := range mergeFileNames {
		srcPath := filepath.Join(mergePath, filName)
		destPath := filepath.Join(db.options.DirPath, filName)
		if err := os.Rename(srcPath, destPath); err != nil {
			return err
		}
		if err := os.Rename(srcPath, destPath); err != nil {
			return err
		}

	}
	return nil

}

func (db *DB) getNonMergeFileId(dirPath string) (uint32, error) {

	mergeFinishedFile, err := data.OpenMergeFinishedFile(dirPath)
	if err != nil {
		return 0, err
	}
	record, _, err := mergeFinishedFile.ReadLogRecord(0)
	if err != nil {
		return 0, err
	}
	nonMergeFileId, err := strconv.Atoi(string(record.Value))
	if err != nil {
		return 0, err
	}
	return uint32(nonMergeFileId), nil

}

func (db *DB) loadIndexFormHintFile() error {

	hintFileName := filepath.Join(db.options.DirPath, data.HintFileName)
	if _, err := os.Stat(hintFileName); os.IsNotExist(err) {
		return nil
	}

	hintFile, err := data.OpenHintFile(db.options.DirPath)
	if err != nil {
		return err
	}

	var offset int64 = 0
	for {

		logRecord, size, err := hintFile.ReadLogRecord(offset)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		pos := data.DecodeLogRecordPos(logRecord.Value)
		db.index.Put(logRecord.Key, pos)
		offset += size
	}
	return nil

}
