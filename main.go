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

func node(id int, initialSum float64, in chan Message, neighbors []chan Message, wg *sync.WaitGroup, n int, avg float64) {
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
		randIndex := rand.Intn(len(neighbors))
		target := neighbors[randIndex]

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

	fmt.Println("\nSelect Topology: 1=Ring, 2=Line, 3=Star, 4=Complete")
	var topology int
	fmt.Scan(&topology)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)

		var myNeighbors []chan Message

		switch topology {
		case 1: // Ring
			left := (n - 1 + i) % n
			right := (i + 1) % n
			myNeighbors = append(myNeighbors, channels[left], channels[right])

		case 2: // Line
			if i > 0 { // connect to left
				myNeighbors = append(myNeighbors, channels[i-1])
			}
			if i < n-1 { // connect to right
				myNeighbors = append(myNeighbors, channels[i+1])
			}

		case 3: // Star - node 0 is the hub
			if i == 0 {
				for j := 1; j < n; j++ {
					myNeighbors = append(myNeighbors, channels[j])
				}
			} else {
				myNeighbors = append(myNeighbors, channels[0])
			}

		case 4: // Complete
			for j := 0; j < n; j++ {
				if i != j {
					myNeighbors = append(myNeighbors, channels[j])
				}
			}
		}

		// Pass the constructed neighbor list to the node
		go node(i, sums[i], channels[i], myNeighbors, &wg, n, avg)
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
