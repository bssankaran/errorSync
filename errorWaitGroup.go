package errorSync

import (
	"sync"
)

type NotifCode int

const (
	NotifCode_Done NotifCode = iota
	NotifCode_Error
)

type Notif struct {
	adminId AdminId
	code    NotifCode
	info    interface{}
	err     error
}

func (n Notif) AdminId() AdminId {
	return n.adminId
}

func (n Notif) Code() NotifCode {
	return n.code
}

func (n Notif) Info() interface{} {
	return n.info
}

func (n Notif) Err() error {
	return n.err
}

type InterruptCode int

const (
	InterruptCode_Stop InterruptCode = iota
	InterruptCode_Kill
)

type Interrupt struct {
	code InterruptCode
	info interface{}
}

func (i Interrupt) Code() InterruptCode {
	return i.code
}

func (i Interrupt) Info() interface{} {
	return i.info
}

type Admin struct {
	id          AdminId
	interruptCh chan Interrupt
	ewg         *ErrorWaitGroup
}

func (a Admin) PollInterrupt() (Interrupt, bool) {
	select {
	case i := <-a.interruptCh:
		return i, true
	default:
		return Interrupt{}, false
	}
}

// Sends an interrupt in its interrupt channel
func (a *Admin) interrupt(inter Interrupt) {
	a.interruptCh <- inter
}

func (a *Admin) Notify(code NotifCode, info interface{}, err error) {
	a.ewg.notifCh <- Notif{a.id, code, info, err}
}

func (a *Admin) NotifyDone() {
	a.ewg.safe.Lock()
	delete(a.ewg.admins, a.id)
	a.ewg.safe.Unlock()
	a.Notify(NotifCode_Done, nil, nil)
}

func (a *Admin) NotifyError(err error) {
	a.Notify(NotifCode_Error, nil, err)
}

type AdminId struct {
	id int
}

type adminIdGen struct {
	top AdminId
}

func (aig *adminIdGen) next() AdminId {
	aig.top.id++
	return aig.top
}

type ErrorWaitGroup struct {
	notifCh        chan Notif
	idGen          adminIdGen
	info           interface{}
	safe           *sync.Mutex
	admins         map[AdminId]*Admin
	autoInterrupts map[NotifCode]InterruptCode
}

func New() ErrorWaitGroup {
	return ErrorWaitGroup{make(chan Notif), adminIdGen{AdminId{-1}}, nil, &sync.Mutex{}, make(map[AdminId]*Admin), make(map[NotifCode]InterruptCode)}
}

func (ewg *ErrorWaitGroup) Add() *Admin {
	ewg.safe.Lock()
	adminId := ewg.idGen.next()
	admin := Admin{adminId, make(chan Interrupt), ewg}
	ewg.admins[adminId] = &admin
	ewg.safe.Unlock()
	return &admin
}

func (ewg *ErrorWaitGroup) SetAutoInterrupt(n NotifCode, i InterruptCode) {
	ewg.safe.Lock()
	ewg.autoInterrupts[n] = i
	ewg.safe.Unlock()
}

func (ewg *ErrorWaitGroup) RemoveAutoInterrupt(n NotifCode) {
	ewg.safe.Lock()
	delete(ewg.autoInterrupts, n)
	ewg.safe.Unlock()
}

func (ewg *ErrorWaitGroup) Wait() (Notif, bool) {

	isDone := true
	_n := Notif{}

	if len(ewg.admins) == 0 {
		return _n, isDone
	}

	interruptAll := false
	var iCode InterruptCode

	for true {
		// Wait for someone to call the NotifyDone() or NotifyError() methods.
		n := <-ewg.notifCh

		// If NotifCode is NotifCode_Done, it indicates that one of the processes has called the Done() method.
		// If there are few more admins in alive in the wait group, the method waits, else returns a nil Notif.
		// If NotifCode is NotifCode_Error, it indicates that one of the processes has called the NotifyError()
		// method. The method waits no more and returns the Notification.
		ewg.safe.Lock()
		if n.code != NotifCode_Done {
			iCode, interruptAll = ewg.autoInterrupts[n.code]
			_n = n
			isDone = false
			break
		} else {
			if len(ewg.admins) == 0 {
				break
			}
		}
		ewg.safe.Unlock()
	}
	if interruptAll {
		ewg.InterruptAll(Interrupt{iCode, nil})
	}
	return _n, isDone
}

// Broadcasts an interrupt to all the admins
func (ewg *ErrorWaitGroup) InterruptAll(inter Interrupt) {
	ewg.safe.Lock()
	for id, _ := range ewg.admins {
		ewg.admins[id].interrupt(inter)
	}
	ewg.safe.Unlock()
}

// Interrupts a single admin
func (ewg *ErrorWaitGroup) Interrupt(id AdminId, inter Interrupt) {
	ewg.safe.Lock()
	admin := ewg.admins[id]
	if admin != nil {
		admin.interrupt(inter)
	}
	ewg.safe.Unlock()
}
