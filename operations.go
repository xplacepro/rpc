package rpc

import (
	"github.com/gorilla/mux"
	"github.com/pborman/uuid"
	"log"
	"net/http"
	"sync"
	"time"
)

type Operation struct {
	id         string
	status     int
	createdAt  int64
	finishedAt int64
	chanDone   chan error
	lock       sync.Mutex
	run        func(*Operation) (interface{}, error)
	err        error
	metadata   interface{}
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

func OperationCreate(run func(op *Operation) (interface{}, error), name string) (string, chan error) {
	operationsLock.Lock()
	defer operationsLock.Unlock()

	op := Operation{}
	op.id = uuid.New()

	op.status = 100
	op.createdAt = time.Now().Unix()
	operations[op.id] = &op

	log.Printf("Created operation, %s", op.id)

	op.lock.Lock()
	defer op.lock.Unlock()

	chanRun := make(chan error, 1)

	go func(op *Operation, chanRun chan error) {
		metadata, err := run(op)
		chanRun <- err
		close(chanRun)
		if err != nil {
			op.status = 400
			op.err = err
		} else {
			op.metadata = metadata
			op.status = 200
		}
		op.finishedAt = time.Now().Unix()
		log.Printf("Finished operation %s, %s, took %d seconds", name, op.id, op.finishedAt-op.createdAt)
	}(&op, chanRun)

	return op.id, chanRun
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
		return NotFound
	}

	if op.status == 200 {
		OperationDelete(id)
	}

	return SyncResponse(op.Meta())
}
