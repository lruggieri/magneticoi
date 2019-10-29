package queue

import (
	"fmt"
	"github.com/beanstalkd/go-beanstalk"
	"time"
)

type Beanstalk struct{
	c *beanstalk.Conn
	q chan Message
}
func (btk *Beanstalk) Read () (oMessage chan Message, oError error){
	c, err := beanstalk.Dial("tcp", "127.0.0.1:11300")
	if err != nil{
		return nil, err
	}
	btk.c = c
	go btk.fetch()
	return btk.q,nil
}

func (btk *Beanstalk) fetch(){
	for{
		id, body, err := btk.c.Reserve(5 * time.Second)
		if err != nil{
			if err == beanstalk.ErrTimeout{continue}
			btk.q <- Message{
				Error:err,
			}
		}
		fmt.Println("received ",id,body)
	}
}