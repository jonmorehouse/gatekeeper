package gatekeeper

import (
	"encoding/gob"
	"math/rand"
	"time"
)

func init() {
	gob.Register(make(map[string]interface{}))
	rand.Seed(time.Now().Unix())
}
