package client

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/robertkrimen/otto"
)

const (
	ctrlcTimeout = time.Second
)

type Client struct {
	ctrlcAt    time.Time
	gui        *gocui.Gui
	connection io.Writer
	ot         *otto.Otto
}

func (self *Client) Close() {
	self.gui.Close()
}

func (self *Client) handleLine(g *gocui.Gui, v *gocui.View) (err error) {
	line, err := v.Line(0)
	if err != nil {
		return
	}
	v.Clear()
	v.SetCursor(0, 0)
	if len(line) > 0 {
		line = strings.TrimSpace(line)
		line = line[:len(line)-1]
		if line[0] == '/' {
			result, e := self.ot.Run(line[1:])
			if e != nil {
				self.Outputf("Error executing %#v: %v", line[1:], e)
				return
			}
			self.Outputf("%v", result)
			return
		}
		if self.connection != nil {
			fmt.Fprintln(self.connection, line)
		}
		self.Outputf("Nowhere to send %#v\n", line)
	}
	return
}

func (self *Client) connect(host string) (err error) {
	addr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return
	}
	_, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		return
	}
	return
}

func (self *Client) bindOtto() {
	self.ot.Set("connect", func(call otto.FunctionCall) (result otto.Value) {
		if err := self.connect(call.Argument(0).String()); err != nil {
			result, _ = otto.ToValue(fmt.Errorf("Error connecting to %#v: %v", call.Argument(0).String(), err))
			return
		}
		result, _ = otto.ToValue(fmt.Sprintf("Connected to %#v", call.Argument(0).String()))
		return
	})
}

func (self *Client) Run() {
	if err := self.gui.Init(); err != nil {
		log.Panicln(err)
	}
	self.gui.SetLayout(self.layout)
	if err := self.gui.SetKeybinding("", gocui.KeyEnter, 0, self.handleLine); err != nil {
		log.Panicln(err)
	}
	if err := self.gui.SetKeybinding("", gocui.KeyCtrlC, 0, self.ctrlc); err != nil {
		log.Panicln(err)
	}
	self.gui.ShowCursor = true
	self.bindOtto()
	err := self.gui.MainLoop()
	if err != nil && err != gocui.ErrorQuit {
		log.Panicln(err)
	}
}

func New() (result *Client) {
	result = &Client{
		gui: gocui.NewGui(),
		ot:  otto.New(),
	}
	return
}

func (self *Client) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if _, err := g.SetView("output", 0, 0, maxX-1, maxY-5); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	if _, err := g.SetView("input", 0, maxY-6, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	g.SetCurrentView("input")
	if v := g.View("input"); v != nil {
		v.Editable = true
	}
	return nil
}

func (self *Client) Outputf(format string, params ...interface{}) {
	if v := self.gui.View("output"); v != nil {
		fmt.Fprintf(v, format, params...)
	}
}

func (self *Client) ctrlc(g *gocui.Gui, v *gocui.View) error {
	if time.Now().Sub(self.ctrlcAt) < ctrlcTimeout {
		return gocui.ErrorQuit
	}
	self.ctrlcAt = time.Now()
	self.Outputf("Press C-c again within %v to quit", ctrlcTimeout)
	return nil
}
