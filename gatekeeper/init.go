package gatekeeper

import (
	"encoding/gob"
	"errors"
	"math/rand"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

func init() {
	//gob.Register(shared.NewError(errors.New("")))
	gob.RegisterName("error", shared.NewError(errors.New("")))
	gob.Register(make(map[string]interface{}))
	rand.Seed(time.Now().Unix())
}
