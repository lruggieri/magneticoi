package queue

import (
	"fmt"
	"testing"
)

func TestBeanstalk_Read(t *testing.T) {
	bean := NewBeanstalkQueue()

	q, err := bean.Read()
	if err != nil{
		t.Error(err)
		t.FailNow()
	}

	fmt.Println("reading")
	for msg := range q{
		fmt.Println(msg)
	}
}
