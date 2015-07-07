package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type State struct {
	activeGets map[string]struct{}

	provideWorker map[int]string
	taskWorkers   map[int]string
	rebroadcast   string
	provConnector string
}

func NewState() *State {
	return &State{
		activeGets:    make(map[string]struct{}),
		provideWorker: make(map[int]string),
		taskWorkers:   make(map[int]string),
	}
}

func (s *State) Print() {
	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println("Bitswap State")
	fmt.Printf("Active Requests: (%d)\n", len(s.activeGets))
	for k, _ := range s.activeGets {
		fmt.Printf("\t%s\n", k)
	}
	fmt.Println("Provides Workers:")
	for i := 0; i < 4; i++ {
		fmt.Printf("\tworker %d: %s\n", i, s.provideWorker[i])
	}
	fmt.Println("TaskWorkers:")
	for i := 0; i < len(s.taskWorkers); i++ {
		fmt.Printf("\tworker %d: %s\n", i, s.taskWorkers[i])
	}

	fmt.Printf("Rebroadcast Worker: %s\n", s.rebroadcast)
	fmt.Printf("Provider Connector: %s\n", s.provConnector)

	fmt.Println()
}

func eventGrabber(api string) (<-chan map[string]interface{}, error) {
	resp, err := http.Get(api)
	if err != nil {
		return nil, err
	}

	out := make(chan map[string]interface{}, 16)
	go func() {
		defer close(out)
		dec := json.NewDecoder(resp.Body)
		for {
			var d map[string]interface{}
			err := dec.Decode(&d)
			if err != nil {
				panic(err)
			}

			if d["system"].(string) == "bitswap" {
				out <- d
			}
		}
	}()
	return out, nil
}

func main() {
	evs, err := eventGrabber("http://localhost:5001/logs")
	if err != nil {
		panic(err)
	}

	s := NewState()
	for e := range evs {
		event := e["event"].(string)
		parts := strings.Split(event, ".")
		if parts[0] != "Bitswap" {
			fmt.Println("incorrectly formatted event: ", event)
			continue
		}
		switch parts[1] {
		case "ProvideWorker":
			id := int(e["ID"].(float64))
			switch parts[2] {
			case "Loop":
				s.provideWorker[id] = "idle"
			case "Work":
				s.provideWorker[id] = e["key"].(string)
			}
		case "Rebroadcast":
			s.rebroadcast = parts[2]
		case "ProviderConnector":
			switch parts[2] {
			case "Loop":
				s.provConnector = "idle"
			case "Work":
				s.provConnector = "active"
			}
		case "GetBlockRequest":
			k := e["key"].(string)

			switch parts[2] {
			case "Start":
				s.activeGets[k] = struct{}{}
			case "End":
				delete(s.activeGets, k)
			}
		case "TaskWorker":
			id := int(e["ID"].(float64))
			switch parts[2] {
			case "Loop":
				s.taskWorkers[id] = "idle"
			case "Work":
				target := e["Target"].(string)
				block := e["Block"].(string)
				s.taskWorkers[id] = fmt.Sprintf("%s to %s", block, target)
			}
		default:
			fmt.Println("UNRECOGNIZED EVENT: ", parts[1])
		}
		s.Print()
	}
}
