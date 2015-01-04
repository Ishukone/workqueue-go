package workqueue

import (
	"runtime"
	"sync"
)

type Work struct {
	Data   interface{}
	Action func(*Work)
	Ch     interface{}
}

type WorkQueue struct {
	mainCond     *sync.Cond
	routinesCond *sync.Cond

	routinesLock sync.RWMutex
	freeRoutines int

	listRWLock  sync.RWMutex
	workList    []*Work
	workNumbers int
}

func (wq *WorkQueue) waitScheduleWork() {
	wq.mainCond.L.Lock()
	wq.mainCond.Wait()
	wq.mainCond.L.Unlock()
}

func (wq *WorkQueue) wakeupMainRoutine() {
	wq.mainCond.L.Lock()
	wq.mainCond.Signal()
	wq.mainCond.L.Unlock()
}

func (wq *WorkQueue) waitMainSchedule() {
	wq.routinesCond.L.Lock()

	wq.routinesLock.Lock()
	wq.freeRoutines++
	wq.routinesLock.Unlock()

	wq.routinesCond.Wait()

	wq.routinesLock.Lock()
	wq.freeRoutines--
	wq.routinesLock.Unlock()

	wq.routinesCond.L.Unlock()
}

func (wq *WorkQueue) wakeupRoutines() {
	wq.routinesCond.L.Lock()
	wq.routinesCond.Signal()
	wq.routinesCond.L.Unlock()
}

func (wq *WorkQueue) fetchWork() *Work {
	var work *Work = nil

	wq.listRWLock.Lock()
	if wq.workNumbers > 0 {
		work = wq.workList[0]
		if wq.workNumbers == 1 {
			wq.workList = []*Work{}
		} else {
			wq.workList = wq.workList[1:]
		}
		wq.workNumbers--
	}

	wq.listRWLock.Unlock()

	return work
}

func (wq *WorkQueue) schedulableWorks(works *int, schedule *bool) {
	wq.listRWLock.Lock()
	workNumbers := wq.workNumbers
	wq.listRWLock.Unlock()

	wq.routinesLock.Lock()
	freeRoutines := wq.freeRoutines
	wq.routinesLock.Unlock()

	if freeRoutines >= workNumbers {
		*works = workNumbers
		*schedule = false
	} else {
		*works = workNumbers - freeRoutines
		*schedule = true
	}
}

func (wq *WorkQueue) ScheduleWork(work *Work) {
	wq.listRWLock.Lock()
	wq.workList = append(wq.workList, work)
	wq.workNumbers++
	wq.listRWLock.Unlock()

	wq.wakeupMainRoutine()
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

		var needSchedule bool
		var wakeupRoutines int
		done <- 0
		for {
			if !needSchedule {
				wq.waitScheduleWork()
			}

			wq.schedulableWorks(&wakeupRoutines, &needSchedule)

			for ; wakeupRoutines > 0; wakeupRoutines-- {
				wq.wakeupRoutines()
			}

			if needSchedule {
				runtime.Gosched()
			}
		}
	}()

	<-done
	for i := 0; i < routinesNumber; i++ {
		go func() {
			for {
				wq.waitMainSchedule()

				work := wq.fetchWork()

				if work != nil {
					work.Action(work)
				}
			}
		}()
	}

	return wq
}

func (wq *WorkQueue) DestroyWorkqueue() {
}
