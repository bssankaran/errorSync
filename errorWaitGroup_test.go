package errorSync_test

import (
	"fmt"
	"github.com/bssankaran/errorSync"
	"github.com/matryer/is"
	"math/rand"
	"os"
	"testing"
	"time"
)

//Initial setup common for all test cases
func setup() {
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

const CHAN_MAX = 5

func TestErrorWaitGroup_WithAllGoRoutinesDoneWithoutError_ShouldWaitAndReturnNilError(t *testing.T) {
	var channels []chan int
	ewg := errorSync.New()
	for i := 0; i < CHAN_MAX; i++ {
		ch := make(chan int)
		channels = append(channels, ch)
		admin := ewg.Add()
		go funcWithoutError(ch, admin, true)
	}

	var total int
	admin := ewg.Add()
	go funcAdd(channels, &total, admin, true)

	notif, done := ewg.Wait()

	is := is.New(t)
	is.True(done)
	is.Equal(notif, errorSync.Notif{})
	is.Equal(total, CHAN_MAX*ITERATIONS*ITERATIONS_SUB)
	t.Logf("%v\ntotal= %d", notif, total)
}

func TestErrorWaitGroup_WithoutCallingAddMethod_ShouldNotWaitAndReturnNilError(t *testing.T) {
	var channels []chan int
	ewg := errorSync.New()
	for i := 0; i < CHAN_MAX; i++ {
		ch := make(chan int)
		channels = append(channels, ch)
		go funcWithoutError(ch, nil, false)
	}

	var total int
	go funcAdd(channels, &total, nil, false)

	notif, done := ewg.Wait()

	is := is.New(t)
	is.True(done)
	is.Equal(notif, errorSync.Notif{})
	is.True(total != CHAN_MAX*ITERATIONS*ITERATIONS_SUB)
	t.Logf("%v\ntotal= %d", notif, total)
}

func TestErrorWaitGroup_WithAllGoRoutinesDoneWithoutError_WithNoEWGForSumRoutine_ShouldNotWaitForRoutineToComplete(t *testing.T) {
	var channels []chan int
	ewg := errorSync.New()
	for i := 0; i < CHAN_MAX; i++ {
		ch := make(chan int)
		channels = append(channels, ch)
		admin := ewg.Add()
		go funcWithoutError(ch, admin, true)
	}

	var total int
	go funcAdd(channels, &total, nil, false)

	notif, done := ewg.Wait()

	is := is.New(t)
	is.True(done)
	is.Equal(notif, errorSync.Notif{})
	is.True(total != CHAN_MAX*ITERATIONS*ITERATIONS_SUB)
	t.Logf("%v\ntotal= %d", notif, total)
}

func TestErrorWaitGroup_WithSomeGoRoutinesErroring_ShouldWaitAndReturnTheError(t *testing.T) {
	var channels []chan int
	ewg := errorSync.New()
	for i := 0; i < CHAN_MAX; i++ {
		ch := make(chan int)
		channels = append(channels, ch)
		admin := ewg.Add()
		go funcWithError(i, ch, admin)
	}

	var total int
	admin := ewg.Add()
	go funcAdd(channels, &total, admin, true)

	notif, done := ewg.Wait()

	is := is.New(t)
	is.True(!done)
	is.Equal(notif.Code(), errorSync.NotifCode_Error)
	is.True(notif.Err() != nil)
	is.True(total != CHAN_MAX*ITERATIONS*ITERATIONS_SUB)
	t.Logf("%v\ntotal= %d", notif.Err(), total)
}

const ITERATIONS_SUB = 1000

func doSomething(n int) (string, int) {
	i := 0
	var str string
	for true {
		i++
		n += i
		if i >= ITERATIONS_SUB {
			break
		}
		str = fmt.Sprintf("%s%d ", str, n)
	}
	return str, i
}

const ITERATIONS = 1000

func funcWithoutError(ch chan int, admin *errorSync.Admin, withEwg bool) {
	defer close(ch)
	i := 0
	for ; i < ITERATIONS; i++ {
		_, n := doSomething(i)
		ch <- n
	}

	if withEwg {
		admin.NotifyDone()
	}
}

func funcWithError(id int, ch chan int, admin *errorSync.Admin) {
	defer close(ch)
	errorAt := random(0, ITERATIONS)
	for i := 0; i < ITERATIONS; i++ {
		if i == errorAt {
			admin.NotifyError(fmt.Errorf("Error in func %d at iteration %d", id, i))
		}
		_, n := doSomething(i)
		ch <- n
	}
	admin.NotifyDone()
}

func funcAdd(channels []chan int, sum *int, admin *errorSync.Admin, withEwg bool) {
	var total int
	if len(channels) > 0 {
		for n := range channels[0] {
			total += n
			for i := 1; i < len(channels); i++ {
				total += <-channels[i]
			}
		}
	}

	doSomething(10)

	*sum = total

	if withEwg {
		admin.NotifyDone()
	}
}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
