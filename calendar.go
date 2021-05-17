package mintid

import (
	"fmt"
	"time"

	ics "github.com/arran4/golang-ical"
)

func CreateCalendar(shifts []Shift, method ics.Method, skipLabels ...string) (string, error) {
	cal := ics.NewCalendar()

	cal.SetMethod(method)

	filter := make(map[string]bool)
	for _, label := range skipLabels {
		filter[label] = true
	}

	for n, shift := range shifts {
		if _, ok := filter[shift.Label]; ok {
			continue
		}

		event := cal.AddEvent(fmt.Sprintf("%d-%s", n, shift.Label))
		event.SetSummary(shift.Label)
		event.SetStartAt(shift.Start)
		event.SetEndAt(shift.End)
		event.SetDtStampTime(time.Now())
	}

	return cal.Serialize(), nil
}
