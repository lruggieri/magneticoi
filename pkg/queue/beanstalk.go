package queue

import (
	"encoding/json"
	"errors"
	"github.com/beanstalkd/go-beanstalk"
	"github.com/lruggieri/magneticoi/pkg/util"
	"strings"
	"time"
)

type Beanstalk struct{
	connection    *beanstalk.Conn
	magneticoTube *beanstalk.TubeSet
	queueChannel  chan Message //receive messages from queue
	dbChannel chan SimpleTorrentSummary //send messages to the database
}
func (btk *Beanstalk) Read (iDbChannel chan SimpleTorrentSummary) (oMessage chan Message, oError error){
	c, err := beanstalk.Dial("tcp", "127.0.0.1:11300")
	if err != nil{
		return nil, err
	}
	btk.connection = c
	btk.dbChannel = iDbChannel

	availableTubes,err := c.ListTubes()
	if err != nil{
		return nil,err
	}
	util.Logger.Info("Available tubes: "+strings.Join(availableTubes,", "))
	tubeFound := false
	for _,availableTube := range availableTubes{
		if strings.Contains(strings.ToLower(availableTube),"magnetico"){
			tubeFound = true
			btk.magneticoTube = beanstalk.NewTubeSet(c,availableTube)
		}
	}
	if !tubeFound{return nil, errors.New("tube for magnetico not found")}

	go btk.fetch()
	return btk.queueChannel,nil
}

func (btk *Beanstalk) fetch(){
	util.Logger.Info("Fetch loop started")
	defer func() {util.Logger.Error("Fetch loop returned")}()
	for{
		_, body, err := btk.magneticoTube.Reserve(5 * time.Second)
		if err != nil{
			if err == beanstalk.ErrTimeout{continue}
			btk.queueChannel <- Message{
				Error:err,
			}
			continue
		}
		var simpleTorrentSummary SimpleTorrentSummary
		err = json.Unmarshal(body, &simpleTorrentSummary)
		if err != nil{
			btk.queueChannel <- Message{
				Error:err,
			}
			continue
		}
		btk.dbChannel <- simpleTorrentSummary

		//TODO what do we do with the ID from the beanstalk? We should delete it from the queue
	}
}