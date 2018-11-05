package subscribe

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

type Subscription interface {
	Err() <-chan error
	Unsubscribe()
}

type Feed struct {
	once      sync.Once
	sendLock  chan struct{}
	removeSub chan interface{}
	sendCases caseList

	mu     sync.Mutex
	inbox  caseList
	eType  reflect.Type
	closed bool
}

const subSendCaseOffset = 1

type feedTypeError struct {
	got, want reflect.Type
	op        string
}

var errBadChannel = errors.New("subscribe channel can't be used to send")

func (e feedTypeError) Error() string {
	return fmt.Sprintf("event: wrong type in %s got %s, want %s", e.op, e.got.String(), e.want.String())
}

func (f *Feed) init() {
	f.removeSub = make(chan interface{})
	f.sendLock = make(chan struct{}, 1)
	f.sendLock <- struct{}{}
	f.sendCases = caseList{{Chan: reflect.ValueOf(f.removeSub), Dir: reflect.SelectRecv}}
}

func (f *Feed) Subscribe(channel interface{}) Subscription {
	f.once.Do(f.init)

	chanVal := reflect.ValueOf(channel)
	chanType := chanVal.Type()
	if chanType.Kind() != reflect.Chan || chanType.ChanDir()&reflect.SendDir == 0 {
		panic(errBadChannel)
	}
	sub := &feedSub{feed: f, channel: chanVal, err: make(chan error, 1)}

	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.checkType(chanType.Elem()) {
		panic(feedTypeError{op: "Subscribe", got: chanType, want: reflect.ChanOf(reflect.SendDir, f.eType)})
	}
	cas := reflect.SelectCase{Dir: reflect.SelectSend, Chan: chanVal}
	f.inbox = append(f.inbox, cas)
	return sub
}

func (f *Feed) checkType(t reflect.Type) bool {
	if f.eType == nil {
		f.eType = t
		return true
	}
	return f.eType == t
}

func (f *Feed) remove(sub *feedSub) {
	ch := sub.channel.Interface()
	f.mu.Lock()
	index := f.inbox.find(ch)
	if index != -1 {
		f.inbox = f.inbox.delete(index)
		f.mu.Unlock()
		return
	}
	f.mu.Unlock()

	select {
	case f.removeSub <- ch:
	case <-f.sendLock:
		f.sendCases = f.sendCases.delete(f.sendCases.find(ch))
		f.sendLock <- struct{}{}
	}
}

func (f *Feed) Send(value interface{}) (nsent int) {
	rValue := reflect.ValueOf(value)
	f.once.Do(f.init)
	<-f.sendLock

	f.mu.Lock()
	f.sendCases = append(f.sendCases, f.inbox...)
	f.inbox = nil

	if !f.checkType(rValue.Type()) {
		f.sendLock <- struct{}{}
		panic(feedTypeError{op: "Send", got: rValue.Type(), want: f.eType})
	}
	f.mu.Unlock()

	for i := subSendCaseOffset; i < len(f.sendCases); i++ {
		f.sendCases[i].Send = rValue
	}

	cases := f.sendCases
	for {
		for i := subSendCaseOffset; i < len(cases); i++ {
			if cases[i].Chan.TrySend(rValue) {
				nsent++
				cases = cases.deactivate(i)
				i--
			}
		}
		if len(cases) == subSendCaseOffset {
			break
		}

		chosen, recv, _ := reflect.Select(cases)
		if chosen == 0 {
			index := f.sendCases.find(recv.Interface())
			f.sendCases = f.sendCases.delete(index)
			if index >= 0 && index < len(cases) {
				cases = f.sendCases[:len(cases)-1]
			}
		} else {
			cases = cases.deactivate(chosen)
			nsent++
		}
	}

	for i := subSendCaseOffset; i < len(f.sendCases); i++ {
		f.sendCases[i].Send = reflect.Value{}
	}
	f.sendLock <- struct{}{}
	return nsent
}

type feedSub struct {
	feed    *Feed
	channel reflect.Value
	errOnce sync.Once
	err     chan error
}

func (sub *feedSub) Err() <-chan error {
	return sub.err
}

func (sub *feedSub) Unsubscribe() {
	sub.errOnce.Do(func() {
		sub.feed.remove(sub)
		close(sub.err)
	})
}

type caseList []reflect.SelectCase

// find find index of special channel
func (cs caseList) find(channel interface{}) int {
	for i, item := range cs {
		if item.Chan.Interface() == channel {
			return i
		}
	}
	return -1
}

// delete delete special index of caseList
func (cs caseList) delete(index int) caseList {
	return append(cs[:index], cs[index+1:]...)
}

// deactivate switch position of special element and last element
func (cs caseList) deactivate(index int) caseList {
	last := len(cs) - 1
	cs[index], cs[last] = cs[last], cs[index]
	return cs[:last]
}
