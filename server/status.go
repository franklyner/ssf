package server

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	SnapshotKey = "--snapshot--"
)

// StatusInformation struct holding all status metrics. Meant to be instantiated once per server
type StatusInformation struct {
	Stats map[string]int
	mutex sync.Mutex
	start time.Time
}

// Metric enum of all available metrics
type Metric string

// CreateStatusInfo factory to create a new status info and start thread to process incoming metrics
func CreateStatusInfo() *StatusInformation {
	s := StatusInformation{
		Stats: make(map[string]int),
		start: time.Now(),
	}
	return &s
}

// IncrementMetric increments given metric by one. Threadsafe
func (s *StatusInformation) IncrementMetric(m string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	curVal, exists := s.Stats[m]
	if exists {
		s.Stats[m] = curVal + 1
	} else {
		s.Stats[m] = 1
	}
}

// SetMetric Sets the given metric to the provided value. Threadsafe
func (s *StatusInformation) SetMetric(metric string, value int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Stats[metric] = value
}

func (s *StatusInformation) snapshot() map[string]int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	ss := make(map[string]int)
	for k, v := range s.Stats {
		ss[k] = v
	}
	return ss
}

// StatusController shows status page
var StatusController Controller = Controller{
	Name:      "StatusController",
	Metric:    "status",
	Path:      "/status",
	Methods:   []string{"GET"},
	IsSecured: false,
	ControllerFunc: func(ctx *Context) {
		stats := ctx.StatusInformation.snapshot()
		html := strings.Builder{}
		html.WriteString("<html><h1>Status</h1><br/>")
		html.WriteString(fmt.Sprintf("Running since: %s<br/><br/>", ctx.StatusInformation.start.Format("2006-01-02 15:04:05")))
		html.WriteString(
			`<table>
				<tr align="left">
					<th>Controller Name</th>
					<th>Methods</th>
					<th>Path</th>
					<th>Invokation Count</th>
					<th>Description</th>
				</tr>`)
		for _, ctr := range ctx.Server.GetControllers() {
			html.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%+v</td><td>%s</td><td align='center'>%d</td><td>%s</td></tr>", ctr.Name, ctr.Methods, ctr.Path, stats[ctr.Metric], ctr.Description))
			delete(stats, ctr.Metric)
		}
		html.WriteString("</table>\n")
		html.WriteString("<p><h2>Non Controller Metrics</h2></p>\n")
		html.WriteString(
			`<table>
				<tr align="left">
					<th>Metric</th>
					<th>Value</th>
				</tr>`)
		for metric, value := range stats {
			html.WriteString(fmt.Sprintf("<tr><td>%s</td><td align='center'>%d</td></tr>", metric, value))
		}
		html.WriteString("</table>\n")

		ctx.SendHTMLResponse(http.StatusOK, []byte(html.String()))
	},
}
