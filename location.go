package main

import (
	"strings"

	"go.uber.org/zap/zapcore"
)

// Location contains the abbreviation and name of a location.
type Location struct {
	ID         int    `json:"-"`
	Abbr       string `json:"abbr"`
	Name       string `json:"name"`
	IsLanguage bool   `json:"-"`
	RelatedIds []int  `json:"-"`
}

/*
type RelatedLocation struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	LocationID int  `gorm:"index:idx,unique"`
	RelatedID  int  `gorm:"index:idx"`
	Sort       int
}
*/

// Folder returns the folder of a location.
func (c Location) Folder() string {
	return c.Abbr + " " + c.Name
}

// NewLocation creates a location from a folder name
func NewLocation(dir string) (Location, bool) {
	parts := strings.SplitN(dir, " ", 2) // ABBR NAME
	if len(parts) != 2 {
		return Location{}, false
	}
	return Location{Abbr: parts[0], Name: parts[1]}, true
}

// MarshalLogObject allows Locations to be logged with Zap.
func (c Location) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("abbr", c.Abbr)
	enc.AddString("name", c.Name)
	enc.AddInt("id", c.ID)
	return nil
}
