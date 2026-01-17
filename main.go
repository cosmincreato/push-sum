package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

type Message struct {
	Sum    float64
	Weight float64
}

// global state for monitoring
var (
	currentEstimates []float64
	currentErrors    []float64
	mu               sync.Mutex // to prevent race conditions during prints
)

func node(id int, initialSum float64, in chan Message, left chan Message, right chan Message, wg *sync.WaitGroup, n int, avg float64) {
	defer wg.Done()

	sum := initialSum
	weight := 1.0

	for {
		// extract half the nodes' values
		sumHalf := sum / 2
		weightHalf := weight / 2
		sum = sumHalf
		weight = weightHalf

		// select a random peer
		target := left
		if rand.Intn(2) == 0 {
			target = right
		}

		// send half the nodes' values
		target <- Message{Sum: sumHalf, Weight: weightHalf}

		// latency to make the process observable
		time.Sleep(time.Millisecond * 100)

		// receive messages from other nodes
		stopReceiving := false
		for !stopReceiving {
			select {
			case msg := <-in:
				sum += msg.Sum
				weight += msg.Weight
			default:
				stopReceiving = true
			}
		}

		// update global state safely
		mu.Lock()
		currentEstimates[id] = sum / weight
		currentErrors[id] = math.Abs((sum / weight) - avg)
		mu.Unlock()
	}
}

func main() {
	// random peer selection
	rand.Seed(time.Now().UnixNano())

	// input for number of nodes
	var n int
	for {
		fmt.Print("Number of nodes (n): ")
		_, err := fmt.Scan(&n)
		if err != nil {
			return
		}
		if n >= 2 {
			break
		}
		fmt.Println("The algorithm needs at least 2 nodes.")
	}

	// input for initial sum
	sums := make([]float64, n)
	var totalSum float64
	for i := 0; i < n; i++ {
		fmt.Printf("Sum of node %d: ", i+1)
		_, err := fmt.Scan(&sums[i])
		if err != nil {
			return
		}
		totalSum += sums[i]
	}

	avg := totalSum / float64(n)
	fmt.Printf("\nTarget Average: %.4f\n\n", avg)

	currentEstimates = make([]float64, n)
	currentErrors = make([]float64, n)

	// node buffers for receiving asynchronous messages
	channels := make([]chan Message, n)
	for i := range channels {
		channels[i] = make(chan Message, n*2)
	}

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		left := (n - 1 + i) % n
		right := (i + 1) % n
		go node(i, sums[i], channels[i], channels[left], channels[right], &wg, n, avg)
	}

	counter := 0
	for {
		time.Sleep(time.Second * 2)
		counter++

		mu.Lock()
		maxErr := 0.0
		for _, e := range currentErrors {
			if e > maxErr {
				maxErr = e
			}
		}

		// Print a compact block of info
		fmt.Printf("Maximum Error: %.6f\n", maxErr)
		fmt.Print("Estimates: ")
		for i, est := range currentEstimates {
			fmt.Printf("[Node %d] %.4f, ", i+1, est)
		}
		fmt.Println("\n------------------------------------------------------------------")

		// stop if converged
		if maxErr < 0.000001 {
			fmt.Println("Convergence criteria met.")
			break
		}
		mu.Unlock()
	}
}
