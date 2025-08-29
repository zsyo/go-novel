package model

type SearchResult struct {
	BookName       string `json:"bookName"`
	Author         string `json:"author"`
	Category       string `json:"category"`
	WordCount      string `json:"wordCount"`
	Status         string `json:"status"`
	LatestChapter  string `json:"latestChapter"`
	LastUpdateTime string `json:"lastUpdateTime"`
	URL            string `json:"url"`
	SourceId       int    `json:"sourceId"`
}
