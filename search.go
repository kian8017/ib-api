package main

import (
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type SearchType int

const (
	ST_SPECIFIC SearchType = 1
	ST_FALLBACK SearchType = 2
	ST_EXTENDED SearchType = 3
)

func NewSearchType(s string) (SearchType, bool) {
	switch s {
	case "specific":
		return ST_SPECIFIC, true
	case "fallback":
		return ST_FALLBACK, true
	case "extended":
		return ST_EXTENDED, true
	default:
		return 0, false
	}
}

// NUM_RESULTS is the maximum number of search results to return.
const NUM_RESULTS int = 100

// SearchQueryRaw is a raw search query, straight from the client.
type SearchQueryRaw struct {
	Query    string `json:"query"`
	Location string `json:"location"`
	Type     string `json:"type"`
}

// SearchQuery is a validated search query, with actual Location and EntryType.
type SearchQuery struct {
	Query    string
	Location *Location
	Type     EntryType
}

// MarshalLogObject allows SearchQueries to be logged with Zap.
func (sq SearchQuery) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("query", sq.Query)
	enc.AddObject("location", sq.Location)
	enc.AddString("entryType", string(sq.Type))
	return nil
}

// NewSearchQuery creates a new search query from strings.
// It validates the type, location, and query before returning the new SearchQuery.
func (s *Server) NewSearchQuery(query, locationAbbr, entryType string) (SearchQuery, string) {
	et, ok := NewEntryType(entryType)
	if !ok {
		s.logger.Error("invalid entryType", zap.String(ZAP_ENTRY_TYPE, entryType))
		return SearchQuery{}, RS_INVALID_TYPE
	}
	location, ok := s.LookupLocationByAbbr(locationAbbr)
	if !ok {
		s.logger.Error("invalid locationAbbr", zap.String(ZAP_LOCATION_ABBR, locationAbbr))
		return SearchQuery{}, RS_INVALID_LOCATION
	}

	if query == "" {
		s.logger.Error("invalid query (blank)")
		log.Print("SV(server.go): Invalid query (blank)")
		return SearchQuery{}, RS_BLANK_QUERY
	}

	b := SearchQuery{Query: query, Location: location, Type: et}
	return b, ""
}

func (s *Server) FormatSearch(query string) string {
	// Replacements
	current := s.DoReplacements(query)
	return current
}

// IndividualSearch runs a specific (1 location) search.
func (s *Server) IndividualSearch(query string, loc *Location, typ EntryType, num int) ([]Entry, bool) {
	fileName := FileName(loc, typ)
	folder := filepath.Join(NAME_FOLDER, loc.Folder(), fileName)

	// Check for existence of file first
	if !FileExists(folder) {
		s.logger.Info("file doesn't exist", zap.String(ZAP_PATH, folder))
		return []Entry{}, true // No results, but not a user error
	}

	cmd := exec.Command("rg", "--crlf", "-i", "-m", strconv.Itoa(num), query, folder)

	out, err := cmd.Output()
	if err != nil {
		// Error with running rg
		if err.Error() == "exit status 1" {
			// No results
			return []Entry{}, true
		} else if err.Error() == "exit status 2" {
			// Invalid query (like just a single "[")
			s.logger.Warn("invalid query", zap.String("query", query))
			return []Entry{}, false
		} else {
			s.logger.Error("error running rg", zap.Error(err))
			return []Entry{}, false
		}
	} else {
		entries := []Entry{}
		trimmed := strings.TrimSuffix(string(out), "\n")
		lines := strings.Split(trimmed, "\n")
		for _, l := range lines {
			entries = append(entries, Entry{
				Name:     l,
				Type:     typ,
				Location: loc,
			})
		}
		return entries, true
	}
}

// ExtendedSearch runs a broader (all locations, but same EntryType) search.
func (s *Server) ExtendedSearch(query string, loc *Location, typ EntryType, numResults int) ([]Entry, bool) {
	excludeLocations := append(loc.RelatedIds, loc.ID) // don't return results for that specific country or related

	typeGlob := "**/*" + string(typ) + ".txt" // N, O, P
	cmd := exec.Command("rg", "--crlf", "-i", "--glob", typeGlob, "-m", strconv.Itoa(numResults), query, NAME_FOLDER)

	out, err := cmd.Output()
	if err != nil {
		// Error with running rg
		if err.Error() == "exit status 1" {
			// No results
			return []Entry{}, true
		} else if err.Error() == "exit status 2" {
			// Invalid query (like just a single "[")
			s.logger.Warn("invalid query", zap.String("query", query))
			return []Entry{}, false
		} else {
			log.Print(err)
			return []Entry{}, false
		}
	} else {

		trimmed := strings.TrimSuffix(string(out), "\n")
		lines := strings.Split(trimmed, "\n")
		entries := []Entry{}

		for _, l := range lines {
			if len(entries) == numResults { // Is this the most efficient way?
				break
			}
			parts := strings.Split(l, ":") // [0] is path
			locationFolder := filepath.Base(filepath.Dir(parts[0]))
			locationAbbr := strings.Split(locationFolder, " ")[0]
			location, ok := s.LookupLocationByAbbr(locationAbbr)
			if !ok {
				// Invalid location
				s.logger.Warn("invalid location ", zap.String(ZAP_LOCATION_ABBR, locationAbbr))
				continue
			}

			ok = true

			for _, curID := range excludeLocations {
				if location.ID == curID {
					// current entry ID matches an exclusion
					ok = false
					break
				}
			}

			if !ok {
				continue
			}

			name := strings.Join(parts[1:], ":")

			entries = append(entries, Entry{
				Name:     name,
				Type:     typ,
				Location: location,
			})
		}
		s.logger.Debug("extended search returning results",
			zap.Int(ZAP_NUM_RESULTS, len(entries)),
			zap.String("query", query))
		return entries, true
	}
}
