package booking_state

import (
	"context"
	"encoding/json"
	"fmt"
	"guestEmulator/internal/app/utils"
	"guestEmulator/internal/domain"
	clockEmuProto "guestEmulator/internal/transport/grpc/pb/clockEmu"
	"log"

	"github.com/go-resty/resty/v2"
	"google.golang.org/protobuf/types/known/emptypb"
)

var numberOfNights = []int{3, 5, 7, 10, 14}
var daysAhead = []int{5, 7, 14, 30, 45, 60, 90, 120}
var window = 30

type SelectPeriodState struct {
	clock  clockEmuProto.ClockServiceClient
	client *resty.Client
}

func NewSelectPeriodState(clock clockEmuProto.ClockServiceClient, client *resty.Client) SelectPeriodState {
	return SelectPeriodState{clock: clock, client: client}
}

func (s SelectPeriodState) Execute(ctx context.Context, cottageName string) (domain.Period, error) {
	log.Println("User selects a period of time")

	nights := utils.PickRandom(numberOfNights)
	searchPeriod := utils.PickRandom(daysAhead)

	nowResp, _ := s.clock.Now(context.Background(), &emptypb.Empty{})
	now := nowResp.Time.AsTime()
	from := now.AddDate(0, 0, searchPeriod)
	to := from.AddDate(0, 0, searchPeriod+window)

	resp, err := s.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{"to": to.Format("2006-01-02"), "from": from.Format("2006-01-02")}).
		Get(fmt.Sprintf("/cottage/%s/available-dates", cottageName))

	if err != nil {
		return domain.Period{}, err
	}

	if resp.IsError() {
		return domain.Period{}, fmt.Errorf("error: %s", resp.Status())
	}

	var availablePeriods domain.CottageAvailablePeriod
	if err := json.Unmarshal(resp.Body(), &availablePeriods); err != nil {
		return domain.Period{}, err
	}

	for _, period := range availablePeriods.Periods {
		if period.End.Sub(period.Start).Hours()-float64(24*nights) >= 0 {
			return period, nil
		}
	}

	log.Println("No suitable period found")
	return domain.Period{}, nil
}
