package queue

import (
	"encoding/json"
	"errors"
	"github.com/beanstalkd/go-beanstalk"
	"github.com/lruggieri/magneticoi/pkg/util"
	"strings"
	"time"
)

func NewBeanstalkQueue() (oQueue *beanstalkQueue){
	return &beanstalkQueue{
		queueChannel:make(chan Message),
	}
}

type beanstalkQueue struct{
	connection    *beanstalk.Conn
	magneticoTube *beanstalk.TubeSet
	queueChannel  chan Message //receive messages from queue
}
func (btk *beanstalkQueue) Read () (oMessage chan Message, oError error){
	c, err := beanstalk.Dial("tcp", "127.0.0.1:11300")
	if err != nil{
		return nil, err
	}
	btk.connection = c

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

func (btk *beanstalkQueue) fetch(){
	util.Logger.Info("Beanstalk fetch loop started")
	defer func() {util.Logger.Error("Beanstalk fetch loop returned")}()
	for{
		beanId, body, err := btk.magneticoTube.Reserve(10 * time.Second)
		if err != nil{
			if err == beanstalk.ErrTimeout{continue}
			btk.queueChannel <- Message{
				Error:err,
			}
			continue
		}
		var torrentSummary ExpandedTorrentSummary
		err = json.Unmarshal(body, &torrentSummary)
		if err != nil{
			btk.queueChannel <- Message{
				Error:err,
			}
			continue
		}
		util.Logger.Debug("sending message")
		btk.queueChannel <- Message{
			Body:torrentSummary,
		}
		err = btk.connection.Delete(beanId)
		if err != nil{
			btk.queueChannel <- Message{
				Error:err,
			}
			continue
		}
	}
}