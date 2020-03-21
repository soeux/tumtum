package semaphore

type Semaphore struct {
	waiters chan struct{}
}

func newSemaphore(capcacity int) *Semaphore {
	if capcacity <= 0 {
		panic("invalid capcacity")
	}

	return &Semaphore{
		waiters: make(chan struct{}, capcacity),
	}
}

func (s Semaphore) Close() {
	close(s.waiters)
}

func (s Semaphore) Acquire() {
	s.waiters <- struct{}{}
}

func (s Semaphore) release() {
	<-s.waiters
}
