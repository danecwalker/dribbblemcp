package dribbble

// Shot is a Dribbble shot summary suitable for design inspiration.
type Shot struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	URL      string   `json:"url"`
	ImageURL string   `json:"image_url"`
	Designer string   `json:"designer,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

// ShotDetail is a full shot with higher-res assets and metadata.
type ShotDetail struct {
	Shot
	Description string   `json:"description,omitempty"`
	Images      []string `json:"images,omitempty"`
	OGImage     string   `json:"og_image,omitempty"`
}

// SearchResult is returned by search tools.
type SearchResult struct {
	Query  string `json:"query"`
	Source string `json:"source"`
	Count  int    `json:"count"`
	Shots  []Shot `json:"shots"`
}
