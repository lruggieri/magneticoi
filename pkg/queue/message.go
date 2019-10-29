package queue

type File struct {
	Size int64  `json:"size"`
	Path string `json:"path"`
}

type SimpleTorrentSummary struct {
	InfoHash string `json:"infoHash"`
	Name     string `json:"name"`
	Files    []File `json:"files"`
}
