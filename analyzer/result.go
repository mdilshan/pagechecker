package analyzer

// Result holds everything we report about an analyzed page.
type Result struct {
	URL          string
	HTMLVersion  string
	Title        string
	Headings     HeadingCounts
	Links        LinkStats
	HasLoginForm bool
}

// HeadingCounts stores the number of headings per level: index 0 is h1, index 5 is h6.
type HeadingCounts [6]int

// Total returns the number of headings across all levels.
func (h HeadingCounts) Total() int {
	total := 0
	for _, n := range h {
		total += n
	}
	return total
}

// LinkStats summarizes the links found in the document.
type LinkStats struct {
	Internal     int
	External     int
	Inaccessible int
}

// Total returns the number of internal and external links.
func (l LinkStats) Total() int {
	return l.Internal + l.External
}
