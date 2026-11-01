// Package aoc models Advent of Code as a provider.
//
// The year list is pinned rather than derived from the clock: the current
// year's puzzles only exist after December starts. Puzzle text is not fetched
// yet; courses carry placeholder stages so the UI can be built and tested
// without scraping the site.
package aoc

import (
	"fmt"

	"github.com/aadhii-yz/quest/internal/provider"
)

// Years are the pinned Advent of Code years, newest first.
var Years = []int{2024, 2023, 2022, 2021, 2020, 2019, 2018, 2017, 2016, 2015}

// Provider implements provider.Provider for Advent of Code.
type Provider struct{}

// New returns the Advent of Code provider.
func New() *Provider { return &Provider{} }

func (p *Provider) ID() string   { return "aoc" }
func (p *Provider) Name() string { return "Advent of Code" }

func (p *Provider) Note() string {
	return "Annual puzzles. Quest fetches puzzle text politely, one request at a time, and caches it locally."
}

func (p *Provider) Planned() bool { return false }

func (p *Provider) Courses() []provider.Course {
	courses := make([]provider.Course, 0, len(Years))
	for _, year := range Years {
		courses = append(courses, course(year))
	}
	return courses
}

func course(year int) provider.Course {
	c := provider.Course{
		ID:           fmt.Sprintf("aoc-%d", year),
		ProviderID:   "aoc",
		ProviderName: "Advent of Code",
		Name:         fmt.Sprintf("%d", year),
		Summary:      fmt.Sprintf("25 two-part puzzles from the %d Advent of Code.", year),
		Languages:    provider.DefaultLanguages,
		Note:         "puzzle fetching is not implemented yet",
	}
	for day := 1; day <= 25; day++ {
		c.Stages = append(c.Stages, provider.Stage{
			ID:     fmt.Sprintf("day-%02d", day),
			Number: day,
			Name:   fmt.Sprintf("Day %d", day),
			Description: fmt.Sprintf(
				"Puzzle text for %d day %d will appear here once polite fetching and local caching are implemented.",
				year, day),
		})
	}
	return c
}
