package main

import (
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

type Counts struct {
	Specific int64 `json:"specific"`
	Fallback int64 `json:"fallback"` // sum of all fallback countries
	Extended int64 `json:"extended"`
}

// InstallFileLengths sets up the arrays, grabs file lengths from the nameFolder, and populates the server's cache.
//
// Depends on InstallLocations.
func (s *Server) InstallFileLengths() {
	s.fileLengths = make(map[EntryType]map[int]int64)
	s.totalLengths = make(map[EntryType]int64)
	// Initialize main array
	entryTypes := [3]EntryType{EntryType("N"), EntryType("P"), EntryType("O")}

	for _, et := range entryTypes {
		curTotal := int64(0)
		s.fileLengths[et] = make(map[int]int64)
		for _, location := range s.locations {
			length := s.GetFileCharCount(location, et)
			s.fileLengths[et][location.ID] = length
			curTotal += length
		}
		s.totalLengths[et] = curTotal
	}
}

// FileName takes a Location and Entrytype and returns the filename.
func FileName(c *Location, et EntryType) string {
	return strings.Title(strings.ToLower(c.Abbr)) + string(et) + ".txt"
}

// FileExists returns whether or not a file exists.
func FileExists(fn string) bool {
	_, err := os.Stat(fn)
	return err == nil
}

// GetFileCharCount gets the length of a single file given the location and type.
func (s *Server) GetFileCharCount(c *Location, t EntryType) int64 {
	fileName := FileName(c, t)
	folder := filepath.Join(NAME_FOLDER, c.Folder(), fileName)
	if !FileExists(folder) {
		return 0
	}

	fi, err := os.Stat(folder)
	if err != nil {
		s.logger.Error("error stat'ing file", zap.Error(err), zap.String(ZAP_PATH, folder))
	}

	return fi.Size()
}
