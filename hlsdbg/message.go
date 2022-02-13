package main

import (
	"fmt"
	"log"
	"time"

	"github.com/asticode/go-astichartjs"
	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
)

var index int

func update() {
	time.Sleep(5 * time.Second)
	data := []int{60, 50, 5, 2, 30, 10, 25}
	if err := bootstrap.SendMessage(w, "update", data); err != nil {
		log.Println(fmt.Errorf("sending update event failed: %w", err))
	}
}

// handleMessages handles messages
func handleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	switch m.Name {
	case "disp":
		return data(), nil
	}
	return
}

type Data struct {
	Chart *astichartjs.Chart `json:"chart,omitempty"`
}

func data() *Data {
	return &Data{
		Chart: &astichartjs.Chart{
			Data: &astichartjs.Data{
				Datasets: []astichartjs.Dataset{{
					BackgroundColor: []string{
						astichartjs.ChartBackgroundColorYellow,
						astichartjs.ChartBackgroundColorGreen,
						astichartjs.ChartBackgroundColorRed,
						astichartjs.ChartBackgroundColorBlue,
						astichartjs.ChartBackgroundColorPurple,
					},
					BorderColor: []string{
						astichartjs.ChartBorderColorYellow,
						astichartjs.ChartBorderColorGreen,
						astichartjs.ChartBorderColorRed,
						astichartjs.ChartBorderColorBlue,
						astichartjs.ChartBorderColorPurple,
					},
					//Data:  []interface{}{0, 10, 5, 2, 20, 30, 45},
					Label: "表格",
				}},
				Labels: []string{"January", "February", "March", "April", "May", "June", "July"}},
			Type: astichartjs.ChartTypeLine,
		},
	}
}
