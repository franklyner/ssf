package server

import (
	"fmt"
	"net/http"
	"strings"
)

// StatusInformation struct holding all status metrics. Meant to be instantiated once per server
type StatusInformation struct {
	Stats map[string]int
	inc   chan string
}

// Metric enum of all available metrics
type Metric string

// CreateStatusInfo factory to create a new status info and start thread to process incoming metrics
func CreateStatusInfo() *StatusInformation {
	s := StatusInformation{
		Stats: make(map[string]int),
		inc:   make(chan string, 10),
	}
	go s.processIncoming()
	return &s
}

func (s *StatusInformation) processIncoming() {
	for m := range s.inc {
		curVal, exists := s.Stats[m]
		if exists {
			s.Stats[m] = curVal + 1
		} else {
			s.Stats[m] = 1
		}
	}
}

// IncrementMetric increments given metric by one. Threadsafe
func (s *StatusInformation) IncrementMetric(m string) {
	s.inc <- m
}

// StatusController shows status page
var StatusController Controller = Controller{
	Name:      "StatusController",
	Metric:    "status",
	Path:      "/status",
	Methods:   []string{"GET"},
	IsSecured: false,
	ControllerFunc: func(ctx *Context) {
		stats := &ctx.StatusInformation.Stats
		html := strings.Builder{}
		html.WriteString("<html>This is a status page. <br/><br/>")
		html.WriteString(
			`<table>
				<tr align="left">
					<th>Controller Name</th>
					<th>Methods</th>
					<th>Path</th>
					<th>Invokation Count</th>
				</tr>`)
		for _, ctr := range ctx.Server.GetControllers() {
			html.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%+v</td><td>%s</td><td align='center'>%d</td></tr>", ctr.Name, ctr.Methods, ctr.Path, (*stats)[ctr.Metric]))
		}
		html.WriteString("</table>\n")
		ctx.SendHTMLResponse(http.StatusOK, []byte(html.String()))
		return
	},
}
