package core

import (
	"sync"
	"time"
)

func InStrList(value string, list []string) bool {
	for _, item := range list {
		if value == item {
			return true
		}
	}

	return false
}

func Retry(retries uint, f func() error) error {
	err := f()
	if err == nil {
		return nil
	}

	if retries == 0 {
		return err
	}
	return Retry(retries-1, f)
}

func CallWithTimeout(dur time.Duration, f func() error) (error, bool) {
	var err error
	doneCh := make(chan struct{})

	go func() {
		err = f()
		doneCh <- struct{}{}
	}()

	select {
	case <-doneCh:
		break
	case <-time.After(dur):
		return nil, false
	}

	return err, true
}

func pluginManagersToInterfaces(items []PluginManager) []interface{} {
	if items == nil {
		return []interface{}(nil)
	}

	output := make([]interface{}, len(items))
	for idx, item := range items {
		output[idx] = item
	}

	return output
}

func CallWith(items []interface{}, f func(i interface{}) error) error {
	if items == nil {
		return nil
	}

	errs := NewMultiError()
	var wg sync.WaitGroup

	for _, item := range items {
		wg.Add(1)
		go func(item interface{}) {
			defer wg.Done()
			if err := f(item); err != nil {
				errs.Add(err)
			}
		}(item)
	}

	wg.Wait()
	return errs.ToErr()

}
