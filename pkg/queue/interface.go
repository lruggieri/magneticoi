package queue

type Message struct{
	Error error
	Body ExpandedTorrentSummary
}

type Queue interface{
	Read() (chan Message, error)
}
