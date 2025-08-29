package model

type Rule struct {
	ID       int         `json:"id"`
	URL      string      `json:"url"`
	Name     string      `json:"name"`
	Comment  string      `json:"comment"`
	Language string      `json:"language"`
	Search   SearchRule  `json:"search"`
	Book     BookRule    `json:"book"`
	Toc      TocRule     `json:"toc"`
	Chapter  ChapterRule `json:"chapter"`
	Crawl    CrawlRule   `json:"crawl"`
}

type SearchRule struct {
	Disabled       bool   `json:"disabled"`
	URL            string `json:"url"`
	Method         string `json:"method"`
	Data           string `json:"data"`
	Cookies        string `json:"cookies"`
	Result         string `json:"result"`
	BookName       string `json:"bookName"`
	Author         string `json:"author"`
	Category       string `json:"category"`
	WordCount      string `json:"wordCount"`
	Status         string `json:"status"`
	LatestChapter  string `json:"latestChapter"`
	LastUpdateTime string `json:"lastUpdateTime"`
	Pagination     bool   `json:"pagination"`
	NextPage       string `json:"nextPage"`
}

type BookRule struct {
	URL            string `json:"url"`
	BookName       string `json:"bookName"`
	Author         string `json:"author"`
	Intro          string `json:"intro"`
	Category       string `json:"category"`
	CoverUrl       string `json:"coverUrl"`
	LatestChapter  string `json:"latestChapter"`
	LastUpdateTime string `json:"lastUpdateTime"`
	Status         string `json:"status"`
	WordCount      string `json:"wordCount"`
}

type TocRule struct {
	BaseUri    string `json:"baseUri"`
	URL        string `json:"url"`
	Item       string `json:"item"`
	IsDesc     bool   `json:"isDesc"`
	Pagination bool   `json:"pagination"`
	NextPage   string `json:"nextPage"`
}

type ChapterRule struct {
	Title              string `json:"title"`
	Content            string `json:"content"`
	ParagraphTagClosed bool   `json:"paragraphTagClosed"`
	ParagraphTag       string `json:"paragraphTag"`
	FilterTxt          string `json:"filterTxt"`
	FilterTag          string `json:"filterTag"`
	Pagination         bool   `json:"pagination"`
	NextPage           string `json:"nextPage"`
}

type CrawlRule struct {
	Threads          int `json:"threads"`
	MinInterval      int `json:"minInterval"`
	MaxInterval      int `json:"maxInterval"`
	MaxAttempts      int `json:"maxAttempts"`
	RetryMinInterval int `json:"retryMinInterval"`
	RetryMaxInterval int `json:"retryMaxInterval"`
}
