package model

type Book struct {
	BookName       string `json:"bookName"`
	Author         string `json:"author"`
	Intro          string `json:"intro"`
	Category       string `json:"category"`
	CoverUrl       string `json:"coverUrl"`
	LatestChapter  string `json:"latestChapter"`
	LastUpdateTime string `json:"lastUpdateTime"`
	Status         string `json:"status"`
	WordCount      string `json:"wordCount"`
	URL            string `json:"url"`
	SourceId       int    `json:"sourceId"`
}
