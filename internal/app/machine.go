package app

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/utils"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/tooling/telemetry"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
)

type Machine struct {
	zeroStep                  steps.Step
	firstStep                 steps.Step
	stateMap                  map[steps.Step][]domain.WeightedTuple[steps.Step]
	timeBetweenStepsInSeconds int
}

func (m *Machine) Start(ctx context.Context) error {
	machineCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	machineCtx, span := telemetry.Tracer.Start(machineCtx, "Machine")
	defer span.End()

	if err := m.zeroStep.Execute(machineCtx); err != nil {
		return err
	}
	step := m.firstStep

	ticker := time.NewTicker(time.Duration(m.timeBetweenStepsInSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-machineCtx.Done():
			return nil
		case <-ticker.C:
			slog.InfoContext(machineCtx, "executing step", "step", step.Name())

			if err := step.Validate(); err != nil {
				return err
			}
			if err := step.Execute(machineCtx); err != nil {
				return err
			}

			nextStep := m.stateMap[step]
			if nextStep == nil {
				cancel()
			} else {
				nextStep := utils.PickRandomWeighted(nextStep)
				slog.InfoContext(machineCtx, "transitioning to state", "oldStep", step.Name(), "newStep", nextStep.Name())
				step = nextStep
			}
		}
	}
}

func readGraph(graphFile string, states map[string]steps.Step) (map[steps.Step][]domain.WeightedTuple[steps.Step], error) {
	stateMap := make(map[steps.Step][]domain.WeightedTuple[steps.Step])

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
				domain.WeightedTuple[steps.Step]{Value: states[tailName], Weight: weight},
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
