package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// request/reply functions

// net/http.Client wrapper
type Client struct {
	http.Client
	addr  string
	debug bool
}

func NewClient(addr string, debug bool, timeout time.Duration) *Client {
	return &Client{
		addr: addr,
		Client: http.Client{
			Timeout: timeout,
		},
		debug: debug,
	}
}

func (c *Client) Debug(args ...interface{}) {
	if c.debug {
		log.Print(args...)
	}
}

func (c *Client) Debugf(format string, args ...interface{}) {
	if c.debug {
		log.Printf(format, args...)
	}
}

type Handler func(resp io.Reader) (err error)

// GET, POST (json), DELETE

func (c *Client) GET(path string, handle Handler) (err error) {

	var (
		reqs string = c.addr + path
		resp *http.Response
	)

	c.Debug("[GET] ", reqs)
	if resp, err = c.Get(reqs); err != nil {
		err = fmt.Errorf("request error: %v", err)
		return
	}

	c.Debug("response status: ", resp.StatusCode)
	defer resp.Body.Close()

	err = handle(resp.Body)
	return
}

func (c *Client) POST(path, body string, handle Handler) (err error) {

	var (
		reqs string = c.addr + path
		resp *http.Response
	)

	c.Debugf("[POST] %s, %s", reqs, body)
	resp, err = c.Post(reqs, "application/json", strings.NewReader(body))
	if err != nil {
		err = fmt.Errorf("request error: %v", err)
		return
	}

	c.Debug("response status: ", resp.Status)
	defer resp.Body.Close()

	err = handle(resp.Body)
	return
}

func (c *Client) DELETE(path string, handle Handler) (err error) {

	var (
		reqs string = c.addr + path
		resp *http.Response
		req  *http.Request
	)

	if req, err = http.NewRequest("DELETE", reqs, nil); err != nil {
		err = fmt.Errorf("request creating error: %v", err)
		return
	}

	c.Debug("[DELETE] ", reqs)
	if resp, err = c.Do(req); err != nil {
		err = fmt.Errorf("requset error: %v", err)
		return
	}

	c.Debug("response status: ", resp.StatusCode)
	defer resp.Body.Close()

	err = handle(resp.Body)
	return
}

// DELETE /daemon/close
func (c *Client) closeDaemon() error {
	return c.DELETE("/daemon/close", func(resp io.Reader) error {
		jr, err := readResponse(resp)
		if err != nil {
			return err
		}
		if jr.Status == 200 || jr.Status == 201 {
			fmt.Println(jr.Detail)
			return nil
		}
		return fmt.Errorf("failed: %s", jr.Detail)
	})
}

// GET /node
func (c *Client) getNodeInfo() error {
	return c.GET("/node", func(resp io.Reader) error {
		var nodeInfo NodeInfo
		err := json.NewDecoder(resp).Decode(&nodeInfo)
		if err != nil {
			return fmt.Errorf("error decoding resposne: %v", err)
		}
		printf("%-20s %v",
			"address:",
			nodeInfo.Address)
		printf("%-20s %v",
			"listeninig:",
			humanBool(nodeInfo.Listening))
		printf("%-20s %v",
			"public key:",
			nodeInfo.PubKey)
		return nil
	})
}

//
// list
//

func (c *Client) getSubscriptionsList() (subscriptions []Item, err error) {
	// list subscriptions
	//
	// GET /manager/nodes/:node_id/subscriptions
	// => []Item

	subscriptions, err = c.getList(c.addr+
		"/manager/nodes/stub/subscriptions", true)
	return
}

func (c *Client) listSubscriptions() error {
	subscriptions, err := c.getSubscriptionsList()
	if err != nil {
		return fmt.Errorf("error requesting subscriptions: %v", err)
	}
	// huminize the list
	if len(subscriptions) == 0 {
		fmt.Println("  there aren't subscriptions")
		return nil
	}
	for _, s := range subscriptions {
		printf("  %s %s",
			s.Address,
			s.PubKey)
	}
	return nil
}

func (c *Client) getSubscribersList() (subscribers []Item, err error) {
	// list subscribers
	//
	// GET /manager/nodes/:node_id/subscribers
	// => []Item

	subscribers, err = c.getList(c.addr+
		"/manager/nodes/stub/subscribers", true)
	return
}

// listSubscribers requests list of subscribers
func (c *Client) listSubscribers() error {
	subscribers, err := c.getSubscribersList()
	if err != nil {
		return fmt.Errorf("error requesting subscribers: %v", err)
	}

	// huminize the list
	if len(subscribers) == 0 {
		fmt.Println("  there aren't subscribers")
		return nil
	}
	for _, s := range subscribers {
		printf("  %s %s",
			s.Address,
			s.PubKey)
	}
	return nil
}

//
// add
//

func (c *Client) addSubscription(args string) error {
	// POST "/manager/nodes/:node_id/subscriptions"
	//   {"ip": "host:port", "pubKey": "theKey"}
	var reqp, reqb string // requset URL and request body
	switch ss := strings.Fields(args); len(ss) {
	case 0:
		return errors.New("to few arguments, want <host:port> [pub key]")
	case 1:
		reqp = c.addr + "/manager/nodes/stub/subscriptions"
		reqb = fmt.Sprintf(`{"ip":%q,"pubKey":""}`, ss[0])
	case 2:
		reqp = c.addr + "/manager/nodes/stub/subscriptions"
		reqb = fmt.Sprintf(`{"ip":%q,"pubKey":%q}`, ss[0], ss[1])
	default:
		return errors.New("to many arguments, want <host:port> [pub key]")
	}

	return c.POST(reqp, reqb, func(resp io.Reader) error {
		jr, err := readResponse(resp)
		if err != nil {
			return err
		}
		if jr.Status == 200 || jr.Status == 201 {
			fmt.Println(jr.Detail)
			return nil
		}
		return errors.New(jr.Detail)
	})

}

//
// remove (todo: DRY removeSubscriber + removeSubscription)
//

func (c *Client) removeSubscription(args string) error {
	// DELETE "/manager/nodes/:node_id/subscriptions/:subscription_id"

	var (
		reqs    string // remove using id
		address string // or remove using address

		err error
	)
	switch ss := strings.Fields(args); len(ss) {
	case 0:
		return errors.New("to few argumets, want: <id or ip:port>")
	case 1:
		if strings.Contains(ss[0], ":") {
			address = ss[0]
			break
		}
		reqs = c.addr + "/manager/nodes/stub/subscriptions/" +
			url.QueryEscape(ss[0])
	default:
		return errors.New("to many argumets, want: <id or ip:port>")
	}

	// request list of subscriptions and determine id by host:port
	if reqs == "" {
		var subscriptions []Item
		subscriptions, err = c.getSubscriptionsList()
		if err != nil {
			return fmt.Errorf("error requesting subscriptions list: %v", err)
		}
		for _, s := range subscriptions {
			if s.Address == address {
				reqs = c.addr + "/manager/nodes/stub/subscriptions/" +
					s.PubKey
				goto Request
			}
		}
		// not found
		printf("subscription %s not found", address)
		return nil
	}
Request:
	return c.DELETE(reqs, func(resp io.Reader) error {
		jr, err := readResponse(resp)
		if err != nil {
			return err
		}
		if jr.Status == 200 || jr.Status == 201 {
			fmt.Println(jr.Detail)
			return nil
		}
		return errors.New(jr.Detail)
	})

}

func (c *Client) removeSubscriber(args string) error {
	// DELETE "/manager/nodes/:node_id/subscribers/:subscriber_id

	var (
		reqs    string // remove using id
		address string // or remove using address

		err error
	)
	switch ss := strings.Fields(args); len(ss) {
	case 0:
		return errors.New("to few argumets, want: <id or ip:port>")
	case 1:
		if strings.Contains(ss[0], ":") {
			address = ss[0]
			break
		}
		reqs = c.addr + "/manager/nodes/stub/subscribers/" +
			url.QueryEscape(ss[0])
	default:
		return errors.New("to many argumets, want: <id or ip:port>")
	}

	// request list of subscribers and determine id by host:port
	if reqs == "" {
		var subscribers []Item
		subscribers, err = c.getSubscribersList()
		if err != nil {
			return fmt.Errorf("error requesting subscribers list: %v", err)
		}
		for _, s := range subscribers {
			if s.Address == address {
				reqs = c.addr + "/manager/nodes/stub/subscribers/" + s.PubKey
				goto Request
			}
		}
		// not found
		printf("connection %s not found", address)
		return nil
	}
Request:
	return c.DELETE(reqs, func(resp io.Reader) error {
		jr, err := readResponse(resp)
		if err != nil {
			return err
		}
		if jr.Status == 200 || jr.Status == 201 {
			fmt.Println(jr.Detail)
			return nil
		}
		return errors.New(jr.Detail)
	})

}

//
// stat
//

func (c *Client) getStat() error {
	// GET "/object1/_stat"

	return c.GET("/object1/_stat", func(resp io.Reader) error {
		// read stat
		var stat Statistic
		if err := json.NewDecoder(resp).Decode(&stat); err != nil {
			return fmt.Errorf("error decoding response: %v", err)
		}
		// print the stat
		fmt.Println("total objects:", stat.Total)
		fmt.Println("memory:       ", humanMemory(stat.Memory))
		return nil
	})
}

//
// helpers
//

func (c *Client) getList(url string, jerr bool) (li []Item, err error) {
	c.Debug("[GET] ", url)

	var resp *http.Response
	if resp, err = c.Get(url); err != nil {
		return
	}
	c.Debug("response status: ", resp.Status)

	defer resp.Body.Close()

	// on success we've got 200 or 201
	if !(resp.StatusCode == 200 || resp.StatusCode == 201) {
		// jerr indicate JSONResponse error report
		if !jerr {
			err = fmt.Errorf("invalid response status: %s", resp.Status)
			return
		}
		// error returns JSONResponse
		jr, e := readResponse(resp.Body)
		if e != nil {
			e = fmt.Errorf("error decoding response: %s", err.Error())
			return
		}
		err = fmt.Errorf("response error: %s", jr.Detail)
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&li)
	return
}
