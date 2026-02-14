package client

import (
	"sync"

	pb "github.com/dgarson/claude-sidecar/gen/claude_sidecar/v1"
)

type eventQueue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	items  []*pb.ServerEvent
	closed bool
}

func newEventQueue() *eventQueue {
	queue := &eventQueue{}
	queue.cond = sync.NewCond(&queue.mu)
	return queue
}

func (q *eventQueue) Push(event *pb.ServerEvent) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return false
	}
	q.items = append(q.items, event)
	q.cond.Signal()
	return true
}

func (q *eventQueue) Pop() (*pb.ServerEvent, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) == 0 && !q.closed {
		q.cond.Wait()
	}
	if len(q.items) == 0 {
		return nil, false
	}
	event := q.items[0]
	q.items = q.items[1:]
	return event, true
}

func (q *eventQueue) Close() {
	q.mu.Lock()
	q.closed = true
	q.cond.Broadcast()
	q.mu.Unlock()
}

type subscription struct {
	queue *eventQueue
	out   chan *pb.ServerEvent
	once  sync.Once
}

func newSubscription(buffer int) *subscription {
	sub := &subscription{
		queue: newEventQueue(),
		out:   make(chan *pb.ServerEvent, buffer),
	}
	go sub.run()
	return sub
}

func (s *subscription) run() {
	for {
		event, ok := s.queue.Pop()
		if !ok {
			close(s.out)
			return
		}
		s.out <- event
	}
}

func (s *subscription) Enqueue(event *pb.ServerEvent) {
	_ = s.queue.Push(event)
}

func (s *subscription) Chan() <-chan *pb.ServerEvent {
	return s.out
}

func (s *subscription) Close() {
	s.once.Do(func() {
		s.queue.Close()
	})
}

type eventMux struct {
	queue     *eventQueue
	mu        sync.Mutex
	byRequest map[string][]*subscription
	global    []*subscription
	closed    bool
}

func newEventMux() *eventMux {
	mux := &eventMux{
		queue:     newEventQueue(),
		byRequest: make(map[string][]*subscription),
	}
	go mux.run()
	return mux
}

func (m *eventMux) Enqueue(event *pb.ServerEvent) {
	_ = m.queue.Push(event)
}

func (m *eventMux) Close() {
	m.queue.Close()
}

func (m *eventMux) SubscribeAll(buffer int) *subscription {
	sub := newSubscription(buffer)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		sub.Close()
		return sub
	}
	m.global = append(m.global, sub)
	return sub
}

func (m *eventMux) SubscribeRequest(requestID string, buffer int) *subscription {
	sub := newSubscription(buffer)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		sub.Close()
		return sub
	}
	m.byRequest[requestID] = append(m.byRequest[requestID], sub)
	return sub
}

func (m *eventMux) UnsubscribeRequest(requestID string, sub *subscription) {
	if sub == nil || requestID == "" {
		return
	}
	m.mu.Lock()
	subs := m.byRequest[requestID]
	for i, item := range subs {
		if item == sub {
			subs = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	if len(subs) == 0 {
		delete(m.byRequest, requestID)
	} else {
		m.byRequest[requestID] = subs
	}
	m.mu.Unlock()
	sub.Close()
}

func (m *eventMux) run() {
	for {
		event, ok := m.queue.Pop()
		if !ok {
			m.closeAll()
			return
		}
		for _, sub := range m.subscriptionsFor(event) {
			sub.Enqueue(event)
		}
		if isTurnEnd(event) {
			m.closeRequest(event.GetRequestId())
		}
	}
}

func (m *eventMux) subscriptionsFor(event *pb.ServerEvent) []*subscription {
	m.mu.Lock()
	defer m.mu.Unlock()
	subs := make([]*subscription, 0, len(m.global))
	subs = append(subs, m.global...)
	requestID := event.GetRequestId()
	if requestID == "" {
		return subs
	}
	return append(subs, m.byRequest[requestID]...)
}

func (m *eventMux) closeRequest(requestID string) {
	if requestID == "" {
		return
	}
	m.mu.Lock()
	subs := m.byRequest[requestID]
	delete(m.byRequest, requestID)
	m.mu.Unlock()
	for _, sub := range subs {
		sub.Close()
	}
}

func (m *eventMux) closeAll() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	subs := make([]*subscription, 0, len(m.global))
	subs = append(subs, m.global...)
	for _, list := range m.byRequest {
		subs = append(subs, list...)
	}
	m.global = nil
	m.byRequest = make(map[string][]*subscription)
	m.mu.Unlock()
	for _, sub := range subs {
		sub.Close()
	}
}

func isTurnEnd(event *pb.ServerEvent) bool {
	payload, ok := event.Payload.(*pb.ServerEvent_Turn)
	if !ok || payload.Turn == nil {
		return false
	}
	return payload.Turn.Kind == pb.TurnBoundary_TURN_END
}
