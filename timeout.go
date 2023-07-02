package dtls_tunnel

import "time"

type ActiveRecorder struct {
	lastRead  time.Time
	lastWrite time.Time
}

func NewActiveRecorder(lastRead, lastWrite time.Time) *ActiveRecorder {
	return &ActiveRecorder{
		lastRead:  lastRead,
		lastWrite: lastWrite,
	}
}

func (ar *ActiveRecorder) SetLastRead(lastRead time.Time) {
	ar.lastRead = lastRead
}

func (ar *ActiveRecorder) SetLastWrite(lastWrite time.Time) {
	ar.lastWrite = lastWrite
}

func (ar *ActiveRecorder) RefreshLastRead() {
	ar.SetLastRead(time.Now())
}

func (ar *ActiveRecorder) RefreshLastWrite() {
	ar.SetLastWrite(time.Now())
}

func (ar *ActiveRecorder) IsTimeout(timeout time.Duration) bool {
	return ar.lastRead.Add(timeout).Before(time.Now()) || ar.lastWrite.Add(timeout).Before(time.Now())
}
