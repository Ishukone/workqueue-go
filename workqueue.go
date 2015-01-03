package workqueue

import (
	"runtime"
	"sync"
	"time"
)

type Work struct {
	Data   interface{}
	Action func(*Work) int
	Ch     interface{}
}

type WorkQueue struct {
	mainCond     *sync.Cond
	routinesCond *sync.Cond

	routinesLock  sync.RWMutex
	totalRoutines int
	freeRoutines  int

	listRWLock  sync.RWMutex
	workList    []*Work
	workNumbers int
}

func (wq *WorkQueue) wakeupMain() {
	wq.mainCond.Signal()
}

func (wq *WorkQueue) ScheduleWork(work *Work) {
	wq.listRWLock.Lock()
	wq.workList = append(wq.workList, work)
	wq.workNumbers++
	wq.listRWLock.Unlock()

	wq.wakeupMain()
}

func CreateWorkQueue(routinesNumber int) *WorkQueue {
	runtime.GOMAXPROCS(runtime.NumCPU())

	wq := new(WorkQueue)

	done := make(chan int)
	go func() {
		locker := new(sync.Mutex)
		wq.mainCond = sync.NewCond(locker)

		locker = new(sync.Mutex)
		wq.routinesCond = sync.NewCond(locker)

		done <- 0
		for {
			wq.mainCond.L.Lock()
			wq.mainCond.Wait()
			wq.mainCond.L.Unlock()

			wq.listRWLock.Lock()
			workNumbers := wq.workNumbers
			wq.listRWLock.Unlock()

			wq.routinesLock.Lock()
			freeRoutines := wq.freeRoutines
			wq.routinesLock.Unlock()

			var wakeupRoutines int
			var needSchedule bool
			if freeRoutines > workNumbers {
				wakeupRoutines = workNumbers
				needSchedule = false
			} else {
				wakeupRoutines = workNumbers - freeRoutines
				needSchedule = true
			}

			for ; wakeupRoutines > 0; wakeupRoutines-- {
				wq.routinesCond.L.Lock()
				wq.routinesCond.Signal()
				wq.routinesCond.L.Unlock()
			}

			if needSchedule {
				time.Sleep(5 * time.Second)
			}
		}
	}()

	<-done
	for i := 0; i < routinesNumber; i++ {
		go func(i int) {
			for {
				wq.routinesCond.L.Lock()

				wq.routinesLock.Lock()
				wq.freeRoutines++
				wq.routinesLock.Unlock()

				wq.routinesCond.Wait()

				wq.routinesLock.Lock()
				wq.freeRoutines--
				wq.routinesLock.Unlock()

				wq.routinesCond.L.Unlock()

				wq.listRWLock.Lock()
				work := wq.workList[0]
				wq.workList = wq.workList[1:]
				wq.workNumbers--
				wq.listRWLock.Unlock()

				work.Action(work)
			}
		}(i)
	}

	return wq
}

func (wq *WorkQueue) DestroyWorkqueue() {
}
