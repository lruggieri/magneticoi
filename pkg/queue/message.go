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
func (sts *SimpleTorrentSummary) GetTotalSize() (oTotalSize uint64){
	for _, file := range sts.Files {
		oTotalSize += uint64(file.Size)
	}
	return
}
type ExpandedTorrentSummary struct{
	*SimpleTorrentSummary
	TotalSize uint64 `json:"totalSize"`
	LastDiscovered int64 `json:"lastDiscovered"`
}