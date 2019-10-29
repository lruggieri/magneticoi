package queue

import (
	"errors"
	"github.com/beanstalkd/go-beanstalk"
	"github.com/lruggieri/magneticoi/pkg/util"
	"strings"
	"time"
)

type Beanstalk struct{
	c *beanstalk.Conn
	magneticoTube *beanstalk.TubeSet
	q chan Message
}
func (btk *Beanstalk) Read () (oMessage chan Message, oError error){
	c, err := beanstalk.Dial("tcp", "127.0.0.1:11300")
	if err != nil{
		return nil, err
	}
	btk.c = c

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
	return btk.q,nil
}

func (btk *Beanstalk) fetch(){
	util.Logger.Info("Fetch loop started")
	defer func() {util.Logger.Error("Fetch loop returned")}()
	for{
		id, body, err := btk.magneticoTube.Reserve(5 * time.Second)
		if err != nil{
			if err == beanstalk.ErrTimeout{continue}
			btk.q <- Message{
				Error:err,
			}
		}
	}
}