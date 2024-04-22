package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	defer close(ch)
	for i := 1; i > 0; i++ {
		select {
		case ch <- int64(i):
			fn(int64(i))
		case <-ctx.Done():
			return
		}
	}
}

func Worker(in <-chan int64, out chan<- int64) {
	for {
		v, ok := <-in
		if !ok {
			close(out)
			return
		}

		out <- v
		time.Sleep(1 * time.Millisecond)
	}
}

func main() {
	chIn := make(chan int64)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var inputSum int64
	var inputCount int64

	go Generator(ctx, chIn, func(i int64) {
		atomic.AddInt64(&inputSum, i)
		atomic.AddInt64(&inputCount, 1)
	})

	const NumOut = 5

	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {

		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	amounts := make([]int64, NumOut)

	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup

	for i, out := range outs {
		wg.Add(1)
		go func(in <-chan int64, i int) {
			defer wg.Done()
			for n := range in {
				amounts[i]++
				chOut <- n
			}
		}(out, i)
	}

	go func() {

		wg.Wait()

		close(chOut)
	}()

	var count int64
	var sum int64

	for n := range chOut {
		count++
		sum += n
	}

	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)
	fmt.Println("Разбивка по каналам", amounts)

	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
	for _, v := range amounts {
		inputCount -= v
	}
	if inputCount != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}
