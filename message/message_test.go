package message

import (
	"testing"
)

func TestSplitMessage(t *testing.T) {
	messages := SplitMessage("[1000-12-1510:00:00,1,V1.0.0,030600001,T3,1,E,113.252432,N,22.564152,50.6,270.5,1][2000-12-1510:00:00,1,V1.0.0,030600001,T6,DOMAIN,1][3000-12-15 10:00:00, 1,V1.0.0,030600001,T8,1]")
	if len(messages) != 3 {
		t.Errorf("Should get 3 messages, get %d however.", len(messages))
	}
}
