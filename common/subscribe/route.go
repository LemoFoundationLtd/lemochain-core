package subscribe

import (
	"errors"
	"reflect"
	"sync"
)

const (
	AddNewPeer = "addNewPeer"
	DeletePeer = "deletePeer"
)

var (
	ErrChType       = errors.New("error of channel type")
	ErrNotExistName = errors.New("not exist special subscription")
)

type CaseListItem struct {
	caseList caseList
	eType    reflect.Type
	lock     chan struct{}
}

var centralRoute = NewCentralRouteSub()

type CentralRouteSub struct {
	names map[string]*CaseListItem

	mux sync.RWMutex
}

func NewCentralRouteSub() *CentralRouteSub {
	return &CentralRouteSub{
		names: make(map[string]*CaseListItem),
	}
}

func (r *CentralRouteSub) sub(name string, ch interface{}) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	chValue := reflect.ValueOf(ch)
	chType := chValue.Type()
	if chType.Kind() != reflect.Chan || chType.ChanDir()&reflect.RecvDir == 0 {
		return ErrChType
	}

	if item, ok := r.names[name]; ok {
		if item.eType != chType {
			return ErrChType
		}
	} else {
		r.names[name] = &CaseListItem{caseList: caseList{}, eType: chType.Elem(), lock: make(chan struct{}, 1)}
	}
	cas := reflect.SelectCase{Dir: reflect.SelectSend, Chan: chValue}
	r.names[name].caseList = append(r.names[name].caseList, cas)
	return nil
}

func (r *CentralRouteSub) send(name string, value interface{}) error {
	r.mux.RLock()
	defer r.mux.RUnlock()

	if _, ok := r.names[name]; !ok {
		return ErrNotExistName
	}

	r.names[name].lock <- struct{}{}
	defer func() { <-r.names[name].lock }()

	item := r.names[name]
	cases := item.caseList
	vValue := reflect.ValueOf(value)
	vType := vValue.Type()

	if vType != item.eType {
		return ErrChType
	}

	for i := 0; i < len(cases); i++ {
		cases[i].Send = vValue
	}

	for {
		var i int
		for i = 0; i < len(cases); {
			cas := cases[i]
			if cas.Chan.TrySend(vValue) {
				cases = append(cases[:i], cases[i+1:]...)
			} else {
				i++
			}
		}
		if i == 0 {
			break
		}
	}
	return nil
}

func (r *CentralRouteSub) unSub(name string, ch interface{}) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	chValue := reflect.ValueOf(ch)
	chType := reflect.TypeOf(ch)
	if chType.Kind() != reflect.Chan || chType.ChanDir()&reflect.RecvDir == 0 {
		return ErrChType
	}
	if _, ok := r.names[name]; !ok {
		return ErrNotExistName
	}

	cases := r.names[name].caseList
	if chType != reflect.TypeOf(cases[0]) {
		return ErrChType
	}

	for i := 0; i < len(cases); i++ {
		if chValue == reflect.ValueOf(cases[i]) {
			r.names[name].caseList = append(cases[:i], cases[i+1:]...)
			break
		}
	}
	return nil
}

func Sub(name string, ch interface{}) {
	centralRoute.sub(name, ch)
}

func UnSub(name string, ch interface{}) {
	centralRoute.unSub(name, ch)
}

func Send(name string, value interface{}) {
	centralRoute.send(name, value)
}
