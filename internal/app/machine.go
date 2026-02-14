package app

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/Kenji-Uema/guestEmulator/internal/app/state"
	"github.com/Kenji-Uema/guestEmulator/internal/app/utils"
	"github.com/Kenji-Uema/guestEmulator/internal/domain"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
)

type Machine struct {
	zeroState                 state.ZeroState
	initState                 state.State
	stateMap                  map[state.State][]domain.WeightedTuple[state.State]
	timeBetweenStepsInSeconds int
}

func (m *Machine) Start(ctx context.Context) error {
	baseCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	machineCtx, err := m.zeroState.Execute(baseCtx)
	if err != nil {
		return err
	}
	var input any = domain.IgnoredField{}
	s := m.initState

	ticker := time.NewTicker(time.Duration(m.timeBetweenStepsInSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-machineCtx.Done():
			return nil
		case <-ticker.C:
			nextInput, err := s.Execute(machineCtx, input)
			if err != nil {
				return err
			}

			nextState := m.stateMap[s]
			if nextState == nil {
				cancel()
			} else {
				input = nextInput
				s = utils.PickRandomWeighted(nextState)
			}
		}
	}
}

func readGraph(graphFile string, states map[string]state.State) (map[state.State][]domain.WeightedTuple[state.State], error) {
	stateMap := make(map[state.State][]domain.WeightedTuple[state.State])

	graph, err := graphviz.ParseFile(graphFile)
	if err != nil {
		return nil, err
	}
	defer func(graph *cgraph.Graph) {
		err := graph.Close()
		if err != nil {
			slog.Error("did not close the graph", "err", err)
		}
	}(graph)

	if err := forEachNode(graph, func(node *cgraph.Node) error {
		return forEachOutEdge(graph, node, func(edge *cgraph.Edge) error {
			headName, tailName, weight, err := edgeProperties(edge)
			if err != nil {
				return err
			}

			stateMap[states[headName]] = append(
				stateMap[states[headName]],
				domain.WeightedTuple[state.State]{Value: states[tailName], Weight: weight},
			)

			return nil
		})
	}); err != nil {
		return nil, err
	}

	return stateMap, nil
}

func forEachNode(graph *cgraph.Graph, fn func(*cgraph.Node) error) error {
	node, err := graph.FirstNode()
	if err != nil {
		return err
	}

	for node != nil {
		if err := fn(node); err != nil {
			return err
		}

		node, err = graph.NextNode(node)
		if err != nil {
			return err
		}
	}

	return nil
}

func forEachOutEdge(graph *cgraph.Graph, node *cgraph.Node, fn func(*cgraph.Edge) error) error {
	edge, err := graph.FirstOut(node)
	if err != nil {
		return err
	}

	for edge != nil {
		if err := fn(edge); err != nil {
			return err
		}

		edge, err = graph.NextOut(edge)
		if err != nil {
			return err
		}
	}

	return nil
}

func edgeProperties(edge *cgraph.Edge) (headName string, tailName string, weight float64, err error) {
	tail, err := edge.Tail()
	if err != nil {
		return
	}
	head, err := edge.Head()
	if err != nil {
		return
	}
	tailName, err = tail.Name()
	if err != nil {
		return
	}
	headName, err = head.Name()
	if err != nil {
		return
	}

	wStr := edge.GetStr("weight")
	if wStr != "" {
		weight, err = strconv.ParseFloat(wStr, 64)
		if err != nil {
			return
		}
	}

	return tailName, headName, weight, nil
}
