package set

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func reading(s *ThreadSafeSet, wg *sync.WaitGroup) {

	// mark this go routine done
	defer wg.Done()

	for toCheck := 0; toCheck < 10; {
		if s.Contains(toCheck) {
			fmt.Println("Found - ", toCheck)
			toCheck++
		}
		time.Sleep(time.Millisecond * 5)
	}

	fmt.Println("found all 10")

}

func writing(s *ThreadSafeSet, wg *sync.WaitGroup) {

	// mark this go routine done
	defer wg.Done()

	// start adding first 3
	for toAdd := 0; toAdd < 3; toAdd++ {
		s.Add(toAdd)
		time.Sleep(time.Millisecond * 2)
	}

	fmt.Println("added first 3")

	// clear all
	s.Clear()

	fmt.Println("cleared the set")

	// start adding first 7
	for toAdd := 0; toAdd < 7; toAdd++ {
		s.Add(toAdd)
		time.Sleep(time.Millisecond * 3)
	}

	fmt.Println("added first 7")

	// remove some
	s.Remove(0, 3, 4, 7)

	fmt.Println("removed some elements")

	// start adding first 10
	for toAdd := 0; toAdd < 10; toAdd++ {
		s.Add(toAdd)
		time.Sleep(time.Millisecond * 4)
	}

	fmt.Println("added first 10")

}

func readTimes(s *ThreadSafeSet, wg *sync.WaitGroup, times int) {
	for i := 0; i < times; i++ {
		wg.Add(1)
		go reading(s, wg)
	}
}

func writeTimes(s *ThreadSafeSet, wg *sync.WaitGroup, times int) {
	for i := 0; i < times; i++ {
		wg.Add(1)
		go writing(s, wg)
	}
}

// should not face any deadlocks
func TestSet(t *testing.T) {
	s := NewThreadSafeSet()

	// gorutine waitgroup
	var wg sync.WaitGroup

	// spawn 10 read goroutines
	readTimes(s, &wg, 10)

	// spawn 10 write goroutines
	writeTimes(s, &wg, 10)

	// wait for all goroutines other than main to finish
	wg.Wait()
}
