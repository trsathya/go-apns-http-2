package apns

import (
	"github.com/trsathya/uuid"
	"testing"
)

func TestApns(t *testing.T) {
	// Notification payload & header
	uuid, _ := uuid.NewUUID()
	n := Notification{
		Payload: Payload{
			Alert:             "Simple message",
			Badge:             0,
			Sound:             Default,
			Content_available: false,
			Title:             "This is the title",
		},
		Headers: Headers{
			Apns_id:         uuid,
			Apns_expiration: "0",
		},
	}

	// server cert and keys
	// s := NewServer(DevelopmentHost, certFromApple, keyFromApple, "") // Apple
	// s := NewServer(TestHost, certFromSelf, keyFromSelf, "") // Self
	s := NewServer(TestHost, CertFromCA, KeyFromCA, CaChainCert) // Test

	d := []string{"eb0c1132b01c777d36f2c3e1bbacbad0761f6c0cb4a50caa5fb873459fd42748"}
	SendPush(n, s, d)
}
