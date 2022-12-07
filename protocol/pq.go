package protocol

import (
	"errors"
	"sync"

	"github.com/cornelk/hashmap"
	"github.com/sunvim/dogesyncer/types"
)

var (
	// ErrDisposed is returned when an operation is performed on a disposed
	// queue.
	ErrDisposed = errors.New(`queue: disposed`)

	// ErrTimeout is returned when an applicable queue operation times out.
	ErrTimeout = errors.New(`queue: poll timed out`)

	// ErrEmptyQueue is returned when an non-applicable queue operation was called
	// due to the queue's empty item state
	ErrEmptyQueue = errors.New(`queue: empty queue`)
)

type sema struct {
	ready    chan bool
	response *sync.WaitGroup
}

func newSema() *sema {
	return &sema{
		ready:    make(chan bool, 1),
		response: &sync.WaitGroup{},
	}
}

type waiters []*sema

func (w *waiters) get() *sema {
	if len(*w) == 0 {
		return nil
	}

	sema := (*w)[0]
	copy((*w)[0:], (*w)[1:])
	(*w)[len(*w)-1] = nil // or the zero value of T
	*w = (*w)[:len(*w)-1]
	return sema
}

func (w *waiters) put(sema *sema) {
	*w = append(*w, sema)
}

func (w *waiters) remove(sema *sema) {
	if len(*w) == 0 {
		return
	}
	// build new slice, copy all except sema
	ws := *w
	newWs := make(waiters, 0, len(*w))
	for i := range ws {
		if ws[i] != sema {
			newWs = append(newWs, ws[i])
		}
	}
	*w = newWs
}

type priorityItems []*types.Block

func (items *priorityItems) swap(i, j int) {
	(*items)[i], (*items)[j] = (*items)[j], (*items)[i]
}

func (items *priorityItems) pop() *types.Block {
	size := len(*items)

	// Move last leaf to root, and 'pop' the last item.
	items.swap(size-1, 0)
	item := (*items)[size-1] // Item to return.
	(*items)[size-1], *items = nil, (*items)[:size-1]

	// 'Bubble down' to restore heap property.
	index := 0
	childL, childR := 2*index+1, 2*index+2
	for len(*items) > childL {
		child := childL
		if len(*items) > childR && (*items)[childR].Compare((*items)[childL]) < 0 {
			child = childR
		}

		if (*items)[child].Compare((*items)[index]) < 0 {
			items.swap(index, child)

			index = child
			childL, childR = 2*index+1, 2*index+2
		} else {
			break
		}
	}

	return item
}

func (items *priorityItems) get(number int) []*types.Block {
	returnItems := make([]*types.Block, 0, number)
	for i := 0; i < number; i++ {
		if len(*items) == 0 {
			break
		}

		returnItems = append(returnItems, items.pop())
	}

	return returnItems
}

func (items *priorityItems) push(item *types.Block) {
	// Stick the item as the end of the last level.
	*items = append(*items, item)

	// 'Bubble up' to restore heap property.
	index := len(*items) - 1
	parent := int((index - 1) / 2)
	for parent >= 0 && (*items)[parent].Compare(item) > 0 {
		items.swap(index, parent)

		index = parent
		parent = int((index - 1) / 2)
	}
}

// PriorityQueue is similar to queue except that it takes
// items that implement the Item interface and adds them
// to the queue i order.
type PriorityQueue struct {
	waiters         waiters
	items           priorityItems
	itemMap         *hashmap.Map[uint64, struct{}]
	lock            sync.Mutex
	disposeLock     sync.Mutex
	disposed        bool
	allowDuplicates bool
}

// Put adds items to the queue.
func (pq *PriorityQueue) Put(items ...*types.Block) error {
	if len(items) == 0 {
		return nil
	}

	pq.lock.Lock()
	defer pq.lock.Unlock()

	if pq.disposed {
		return ErrDisposed
	}

	for _, item := range items {
		if pq.allowDuplicates {
			pq.items.push(item)
		} else if _, ok := pq.itemMap.Get(item.Number()); !ok {
			pq.itemMap.Insert(item.Number(), struct{}{})
			pq.items.push(item)
		}
	}

	for {
		sema := pq.waiters.get()
		if sema == nil {
			break
		}

		sema.response.Add(1)
		sema.ready <- true
		sema.response.Wait()
		if len(pq.items) == 0 {
			break
		}
	}

	return nil
}

// Get retrieves items from the queue.  If the queue is empty,
// this call blocks until the next item is added to the queue.  This
// will attempt to retrieve number of items.
func (pq *PriorityQueue) Get(number int) ([]*types.Block, error) {
	if number < 1 {
		return nil, nil
	}

	pq.lock.Lock()

	if pq.disposed {
		pq.lock.Unlock()
		return nil, ErrDisposed
	}

	var items []*types.Block

	// Remove references to popped items.
	deleteItems := func(items []*types.Block) {
		for _, item := range items {
			pq.itemMap.Del(item.Number())
		}
	}

	if len(pq.items) == 0 {
		sema := newSema()
		pq.waiters.put(sema)
		pq.lock.Unlock()

		<-sema.ready

		if pq.Disposed() {
			return nil, ErrDisposed
		}

		items = pq.items.get(number)
		if !pq.allowDuplicates {
			deleteItems(items)
		}
		sema.response.Done()
		return items, nil
	}

	items = pq.items.get(number)
	deleteItems(items)
	pq.lock.Unlock()
	return items, nil
}

// Peek will look at the next item without removing it from the queue.
func (pq *PriorityQueue) Peek() *types.Block {
	pq.lock.Lock()
	defer pq.lock.Unlock()
	if len(pq.items) > 0 {
		return pq.items[0]
	}
	return nil
}

// Empty returns a bool indicating if there are any items left
// in the queue.
func (pq *PriorityQueue) Empty() bool {
	pq.lock.Lock()
	defer pq.lock.Unlock()

	return len(pq.items) == 0
}

// Len returns a number indicating how many items are in the queue.
func (pq *PriorityQueue) Len() int {
	pq.lock.Lock()
	defer pq.lock.Unlock()

	return len(pq.items)
}

// Disposed returns a bool indicating if this queue has been disposed.
func (pq *PriorityQueue) Disposed() bool {
	pq.disposeLock.Lock()
	defer pq.disposeLock.Unlock()

	return pq.disposed
}

// Dispose will prevent any further reads/writes to this queue
// and frees available resources.
func (pq *PriorityQueue) Dispose() {
	pq.lock.Lock()
	defer pq.lock.Unlock()

	pq.disposeLock.Lock()
	defer pq.disposeLock.Unlock()

	pq.disposed = true
	for _, waiter := range pq.waiters {
		waiter.response.Add(1)
		waiter.ready <- true
	}

	pq.items = nil
	pq.waiters = nil
}

// NewPriorityQueue is the constructor for a priority queue.
func NewPriorityQueue(hint int, allowDuplicates bool) *PriorityQueue {
	return &PriorityQueue{
		items:           make(priorityItems, 0, hint),
		itemMap:         hashmap.New[uint64, struct{}](),
		allowDuplicates: allowDuplicates,
	}
}
