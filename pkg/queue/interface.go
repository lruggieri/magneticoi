package queue

type Message struct{
	Error error
	Body SimpleTorrentSummary
}

type Queue interface{
	Read() (chan Message, error)
}
