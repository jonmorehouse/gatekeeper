package core

import (
	"encoding/gob"
	"math/rand"
	"time"
)

func init() {
	gob.Register(make(map[string]interface{}))
	gob.Register(struct{}{})
	rand.Seed(time.Now().Unix())
}
