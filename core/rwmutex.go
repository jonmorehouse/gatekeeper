package core

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// IDEA ... keep a global counter that corresponds to the MutexID
type RWMutex struct {
	sync.RWMutex
}

func (rw *RWMutex) RUnlock() {
	rw.log("RUnlock", rw.RWMutex.RUnlock)
}

func (rw *RWMutex) RLock() {
	rw.log("RLock", rw.RWMutex.RLock)
}

func (rw *RWMutex) Lock() {
	rw.log("Lock", rw.RWMutex.Lock)
}

func (rw *RWMutex) Unlock() {
	rw.log("UnLock", rw.RWMutex.Unlock)
}

func (rw *RWMutex) log(action string, cb func()) {
	startTS := time.Now()
	cb()
	latency := time.Now().Sub(startTS)
	return
	log.Println(fmt.Sprintf("Mutex Operation: action=%s latency=%s", action, latency))
}
