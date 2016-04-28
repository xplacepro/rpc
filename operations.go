package rpc

import (
	"github.com/gorilla/mux"
	"github.com/pborman/uuid"
	"log"
	"net/http"
	"sync"
	"time"
)

var OperationQueue chan *Operation
var WorkersPool chan *Worker

type Worker struct {
	Id             int
	OperationQueue chan *Operation
	Quit           chan int
}

func (w *Worker) Run() {
	go func() {
		log.Printf("Initialized worker %d", w.Id)
		for {
			WorkersPool <- w
			select {
			case op := <-w.OperationQueue:
				op.Run()
			case <-w.Quit:
				return
			}
		}
	}()
}

func (w *Worker) Stop() {
	go func() {
		w.Quit <- 1
	}()
}

func RunDispatcher(num int) {
	OperationQueue = make(chan *Operation)
	WorkersPool = make(chan *Worker)
	for i := 0; i < num; i++ {
		w := &Worker{OperationQueue: make(chan *Operation), Quit: make(chan int), Id: i}
		w.Run()
	}

	go func() {
		for {
			select {
			case op := <-OperationQueue:
				w := <-WorkersPool
				w.OperationQueue <- op
			}
		}
	}()
}

type Operation struct {
	id         string
	name       string
	status     int
	createdAt  int64
	finishedAt int64
	chanDone   chan error
	lock       sync.Mutex
	run        func(*Operation) (interface{}, error)
	err        error
	metadata   interface{}
}

func (op *Operation) Run() {
	op.lock.Lock()
	defer op.lock.Unlock()

	//go func(op *Operation) {
	metadata, err := op.run(op)
	if err != nil {
		op.status = 400
		op.err = err
	} else {
		op.metadata = metadata
		op.status = 200
	}
	op.finishedAt = time.Now().Unix()
	//log.Printf("Finished operation %s, %s, took %d seconds", op.name, op.id, op.finishedAt-op.createdAt)
	//}(op)

}

func (op *Operation) Meta() map[string]interface{} {
	body := map[string]interface{}{
		"id":         op.id,
		"status":     op.status,
		"createdAt":  op.createdAt,
		"finishedAt": op.finishedAt,
		"metadata":   op.metadata,
	}
	if op.err != nil {
		body["error"] = op.err.Error()
	}
	return body
}

var operationsLock sync.Mutex
var operations map[string]*Operation = make(map[string]*Operation)

func OperationCreate(run func(op *Operation) (interface{}, error), name string) string {
	operationsLock.Lock()
	defer operationsLock.Unlock()

	op := &Operation{}
	op.id = uuid.New()

	op.status = 100
	op.name = name
	op.createdAt = time.Now().Unix()
	op.run = run
	operations[op.id] = op

	OperationQueue <- op

	return op.id
}

func OperationGet(id string) (*Operation, bool) {
	operationsLock.Lock()
	defer operationsLock.Unlock()

	op, ok := operations[id]
	return op, ok
}

func OperationDelete(id string) {
	operationsLock.Lock()
	defer operationsLock.Unlock()

	delete(operations, id)
}

func GetOperationHandler(env *Env, w http.ResponseWriter, r *http.Request) Response {
	vars := mux.Vars(r)
	id := vars["uuid"]

	op, ok := OperationGet(id)
	if !ok {
		op = &Operation{}
		op.status = 404
		op.id = id
	}

	if op.status == 200 {
		OperationDelete(id)
	}

	return SyncResponse(op.Meta())
}
