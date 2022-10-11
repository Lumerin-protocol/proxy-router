package blockchain

type clientSubscription struct {
	ErrCh         chan error
	UnsubscribeFn func()
}

func NewClientSubscription(errCh chan error, unsubscribeFn func()) *clientSubscription {
	return &clientSubscription{
		ErrCh:         errCh,
		UnsubscribeFn: unsubscribeFn,
	}
}

func (s *clientSubscription) Err() <-chan error {
	return s.ErrCh
}

func (s *clientSubscription) Unsubscribe() {
	s.UnsubscribeFn()
}
