package main

import (
	"encoding/json"
)

// EntryType represents the type of Entry (Name, Place, or Other).
type EntryType string

// NewEntryType creates a new EntryType from a string.
func NewEntryType(et string) (EntryType, bool) {
	switch et {
	case "name":
		return EntryType("N"), true
	case "place":
		return EntryType("P"), true
	case "other":
		return EntryType("O"), true
	default:
		return EntryType(""), false
	}
}

// Entry represents a search result.
type Entry struct {
	Name     string    `json:"name"`
	Type     EntryType `json:"type"`
	Location *Location `json:"location"`
}

// MarshalEntries takes a list of entries and encodes them into JSON.
func MarshalEntries(e []Entry) []byte {
	enc, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}
	return enc
}
