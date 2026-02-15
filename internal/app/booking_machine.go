package app

import (
	"log/slog"

	"github.com/Kenji-Uema/guestEmulator/internal/app/state"
	"github.com/Kenji-Uema/guestEmulator/internal/app/state/booking_state"
	"github.com/Kenji-Uema/guestEmulator/internal/config"
	"github.com/Kenji-Uema/guestEmulator/internal/domain"
	"github.com/Kenji-Uema/guestEmulator/internal/transport/grpc"
	"github.com/Kenji-Uema/guestEmulator/internal/transport/http"
)

func NewBookingMachine(machineConfig config.BookingMachineConfig, serviceConfig config.ServicesConfig) (*Machine, error) {
	cottageClient := http.NewRestyClient(serviceConfig.CottageManagerUrl, serviceConfig.CottageManagerPort)
	guestClient := http.NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)
	clockEmu, err := grpc.NewClockEmu(serviceConfig)
	defer func(clockEmu *grpc.Emu) {
		err := clockEmu.Close()
		if err != nil {
			slog.Error("failed to close clockEmu", "err", err)
		}
	}(clockEmu)
	if err != nil {
		return nil, err
	}

	zeroState := state.NewInitState()
	bookingMachineStates := map[string]state.State{
		"End":                        state.Adapter[domain.IgnoredField, domain.IgnoredField]{State: state.NewEndState()},
		"SelectCottage":              state.Adapter[[]string, string]{State: booking_state.NewSelectCottageState()},
		"ListCottages":               state.Adapter[domain.IgnoredField, []string]{State: booking_state.NewListCottagesState(cottageClient)},
		"SelectPeriod":               state.Adapter[string, domain.Period]{State: booking_state.NewSelectPeriodState(clockEmu, guestClient)},
		"SearchBy_TypeAndPeriod":     state.Adapter[domain.IgnoredField, []domain.CottageAvailable]{State: booking_state.NewSearchByTypeAndPeriodState(cottageClient)},
		"SelectCottage_PeriodPreSet": state.Adapter[[]domain.CottageAvailable, string]{State: booking_state.NewSelectCottagePeriodPreSetState()},
		"BookCottage":                state.Adapter[domain.Cottage, domain.BookingConfirmation]{State: booking_state.NewBookCottageState(guestClient)},
	}

	stateMap, err := readGraph(machineConfig.GraphFile, bookingMachineStates)
	if err != nil {
		return nil, err
	}

	return &Machine{
		zeroState:                 zeroState,
		initState:                 bookingMachineStates["ListCottages"],
		stateMap:                  stateMap,
		timeBetweenStepsInSeconds: machineConfig.TimeBetweenStepsInSeconds,
	}, nil
}
