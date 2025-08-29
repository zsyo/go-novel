package model

type Chapter struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Order   int    `json:"order"`
	URL     string `json:"url"`
}
