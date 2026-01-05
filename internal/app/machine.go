package app

import (
	"context"
	"guestEmulator/internal/app/state"
	"guestEmulator/internal/app/utils"
	"guestEmulator/internal/config"
	"guestEmulator/internal/domain"
	clockEmuProto "guestEmulator/internal/transport/grpc/pb/clockEmu"
	"log"
	"strconv"
	"time"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
)

type Machine struct {
	zeroState state.ZeroState
	initState state.State
	stateMap  map[state.State][]domain.WeightedTuple[state.State]
}

func NewBookingMachine(config config.BookingMachineConfig) (*Machine, error) {
	cottageClient := utils.NewRestyClient(config.CottageManagerUrl)
	guestClient := utils.NewRestyClient(config.GuestManagerUrl)
	grpcConn := utils.NewGrpcConnection(config.ClockEmuUrl)
	defer utils.CloseGrpcConnection(grpcConn)

	clock := clockEmuProto.NewClockServiceClient(grpcConn)

	zeroState := state.NewInitState()
	bookingMachineStates := map[string]state.State{
		"End":                        state.Adapter[domain.IgnoredField, domain.IgnoredField]{State: state.NewEndState()},
		"SelectCottage":              state.Adapter[[]string, string]{State: state.NewSelectCottageState()},
		"ListCottages":               state.Adapter[domain.IgnoredField, []string]{State: state.NewListCottagesState(cottageClient)},
		"SelectPeriod":               state.Adapter[string, domain.Period]{State: state.NewSelectPeriodState(clock, guestClient)},
		"SearchBy_TypeAndPeriod":     state.Adapter[domain.IgnoredField, []domain.CottageAvailable]{State: state.NewSearchByTypeAndPeriodState(cottageClient)},
		"SelectCottage_PeriodPreSet": state.Adapter[[]domain.CottageAvailable, string]{State: state.NewSelectCottagePeriodPreSetState()},
		"BookCottage":                state.Adapter[domain.Cottage, domain.BookingConfirmation]{State: state.NewBookCottageState(guestClient)},
	}

	stateMap, err := readGraph(config.GraphFile, bookingMachineStates)
	if err != nil {
		return nil, err
	}

	return &Machine{zeroState: zeroState, initState: bookingMachineStates["ListCottages"], stateMap: stateMap}, nil
}

func (m *Machine) Start(ctx context.Context) error {
	machineCtx, err := m.zeroState.Execute(ctx)
	if err != nil {
		return err
	}
	var input any = domain.IgnoredField{}
	s := m.initState

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			nextInput, err := s.Execute(machineCtx, input)
			if err != nil {
				return err
			}
			input = nextInput
			s = utils.PickRandomWeighted(m.stateMap[s])
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
			log.Fatalf("did not close the graph: %v", err)
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
