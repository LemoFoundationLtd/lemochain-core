package subscribe

import (
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"reflect"
	"sync"
)

const (
	AddNewPeer      = "addNewPeer"
	DeletePeer      = "deletePeer" // protocol_manager delete peer
	SrvDeletePeer   = "delPeer"    // server delete peer
	NewMinedBlock   = "newMinedBlock"
	NewCurrentBlock = "newCurrentBlock"
	NewStableBlock  = "newStableBlock"
	NewTx           = "newTx"
	NewConfirm      = "newConfirm"
	FetchConfirms   = "fetchConfirm"
)

var (
	ErrChType       = errors.New("error of channel type")
	ErrDataType     = errors.New("the type of data can not match with channel")
	ErrNotExistName = errors.New("subscriptions not exist")
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
		log.Error("ch must be a writable channel")
		return ErrChType
	}

	if item, ok := r.names[name]; ok {
		if item.eType != chType.Elem() {
			log.Errorf("it's not allowed to setup [%s] channel for name: %s, cause the last channel type is [%s]", chType, name, item.eType)
			return ErrChType
		}
	} else {
		r.names[name] = &CaseListItem{caseList: caseList{}, eType: chType.Elem(), lock: make(chan struct{}, 1)}
	}

	// case ch <- data:
	cas := reflect.SelectCase{Dir: reflect.SelectSend, Chan: chValue}
	r.names[name].caseList = append(r.names[name].caseList, cas)
	return nil
}

func (r *CentralRouteSub) send(name string, value interface{}) error {
	r.mux.RLock()
	defer r.mux.RUnlock()

	if _, ok := r.names[name]; !ok {
		log.Debugf("[%s] event is not subscribed", name)
		return nil
	}

	r.names[name].lock <- struct{}{}
	defer func() { <-r.names[name].lock }()

	item := r.names[name]
	cases := item.caseList
	vValue := reflect.ValueOf(value)
	vType := vValue.Type()
	if vType != item.eType && !vType.Implements(item.eType) {
		return ErrDataType
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
	if chType != cases[0].Chan.Type() {
		return ErrChType
	}

	for i := 0; i < len(cases); i++ {
		if chValue == cases[i].Chan {
			r.names[name].caseList = append(cases[:i], cases[i+1:]...)
			break
		}
	}
	return nil
}

func (r *CentralRouteSub) clearSub() {
	r.names = make(map[string]*CaseListItem)
}

func Sub(name string, ch interface{}) {
	if err := centralRoute.sub(name, ch); err != nil {
		log.Error(err.Error())
	}
}

func UnSub(name string, ch interface{}) {
	if err := centralRoute.unSub(name, ch); err != nil {
		log.Error(err.Error())
	}
}

func ClearSub() {
	centralRoute.clearSub()
}

func Send(name string, value interface{}) {
	if err := centralRoute.send(name, value); err != nil {
		log.Error(err.Error())
	}
}
