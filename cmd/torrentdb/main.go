package main

import (
	"flag"
	"fmt"
	"github.com/lruggieri/magneticoi/pkg/elastic"
	"github.com/lruggieri/magneticoi/pkg/queue"
	"github.com/lruggieri/magneticoi/pkg/util"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"os"
)

var elasticConnection *elastic.Connection
func main(){

	debugMode := flag.Bool("debug",false,"Activate debug mode")
	flag.Parse()

	fmt.Println(*debugMode)
	if *debugMode {
		util.Logger.Warn("DEBUG MODE ACTIVATED")
		util.Logger.Level = logrus.DebugLevel
	}

	esHost := os.Getenv("ES_SERVER_HOST")
	esPort := os.Getenv("ES_SERVER_PORT")
	if len(esHost) == 0{
		esHost = "http://localhost"
	}
	if len(esPort) == 0{
		esPort = "9200"
	}

	esCon,err := elastic.New(esHost,esPort)
	//esCon,err := elastic.New("http://localhost","9200")
	if err != nil{
		log.Fatal(err)
	}
	elasticConnection = esCon

	torrentQueue := queue.NewBeanstalkQueue()
	queueChannel, err := torrentQueue.Read()
	if err != nil{
		log.Panic(err)
	}

	go startTorrentInsertion(queueChannel)

	mux := http.NewServeMux()

	port := os.Getenv("MAGNETICOI_PORT")
	if port == "" {
		port = "8080"
	}
	util.Logger.Info("running API on port "+port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), mux))

}

func startTorrentInsertion(iQueueChannel chan queue.Message){

	insertionChannel := make(chan queue.ExpandedTorrentSummary)

	go func(){
		for{
			util.Logger.Info("spawning insertion torrent function")
			err :=  elastic.InsertTorrentSummary(elasticConnection.Connection,elastic.IndexNameTorrents,insertionChannel)
			if err != nil{
				util.Logger.Error(err)
			}
		}
	}()

	for msg := range iQueueChannel {
		util.Logger.Debug("read message")
		if msg.Error != nil{
			util.Logger.Error(msg.Error)
			continue
		}else{
			insertionChannel <- msg.Body
		}
	}

	util.Logger.Error("startTorrentInsertion returned")

}