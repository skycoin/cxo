package gui

import (
	"encoding/json"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/skycoin/src/cipher"
)

type SkyhashManager struct {
	node.Node
}

//RegisterNodeManagerHandlers - create routes for NodeManager
func RegisterNodeManagerHandlers(router *Router, shm node.Node) {
	// enclose shm into SkyhashManager to be able to add methods
	lshm := SkyhashManager{shm}

	// OBSOLETE

	router.GET("/manager", lshm.ManagerInfo)
	router.POST("/manager", lshm.StartManager)
	router.DELETE("/manager", lshm.StopManager)

	router.GET("/manager/nodes", lshm.ListNodes)
	router.POST("/manager/nodes", lshm.AddNode)

	router.GET("/manager/nodes/:node_id", lshm.GetNodeByID)
	router.DELETE("/manager/nodes/:node_id", lshm.TerminateNodeByID)

	// TODO: rid out of nodes/node_id

	router.GET("/manager/nodes/:node_id/subscriptions", lshm.ListSubscriptions)
	router.POST("/manager/nodes/:node_id/subscriptions", lshm.AddSubscription)

	router.GET("/manager/nodes/:node_id/subscriptions/:subscription_id", lshm.GetSubscriptionByID)
	router.DELETE("/manager/nodes/:node_id/subscriptions/:subscription_id", lshm.TerminateSubscriptionByID)

	router.GET("/manager/nodes/:node_id/subscribers", lshm.ListSubscribers)

	router.GET("/manager/nodes/:node_id/subscribers/:subscriber_id", lshm.GetSubscriberByID)
	router.DELETE("/manager/nodes/:node_id/subscribers/:subscriber_id", lshm.TerminateSubscriberByID)

}

func (self *SkyhashManager) ManagerInfo(ctx *Context) error {
	return ctx.ErrInternal("OBSOLETE")
}
func (self *SkyhashManager) StartManager(ctx *Context) error {
	return ctx.ErrInternal("OBSOLETE")
}
func (self *SkyhashManager) StopManager(ctx *Context) error {
	return ctx.ErrInternal("OBSOLETE")
}

func (self *SkyhashManager) ListNodes(ctx *Context) error {
	return ctx.ErrInternal("OBSOLETE")
}

func (self *SkyhashManager) AddNode(ctx *Context) error {
	return ctx.ErrInternal("OBSOLETE")
}

func (self *SkyhashManager) GetNodeByID(ctx *Context) error {
	return ctx.ErrInternal("OBSOLETE")
}

func (self *SkyhashManager) TerminateNodeByID(ctx *Context) error {
	return ctx.ErrInternal("OBSOLETE")
}

func (self *SkyhashManager) ListSubscriptions(ctx *Context) error {

	var subscriptions []node.Connection

	for c := range self.Outgoing().List() {
		subscriptions = append(subscriptions, c)
	}

	return ctx.JSON(200, subscriptions)
}

func (self *SkyhashManager) AddSubscription(ctx *Context) error {
	// declare configuration struct of new subscription
	var newSubscriptionConfig struct {
		IP     string `json:"ip"` // host:port
		PubKey string `json:"pubKey"`
	}

	// decode configuration of the new subscription to be established
	err := json.NewDecoder(ctx.Request.Body).Decode(&newSubscriptionConfig)
	if err != nil {
		return ctx.ErrInvalidRequest(err.Error())
	}

	// If the pubKey parameter is an empty cipher.PubKey{}, we will connect to that node
	// for any PubKey it communicates us it has.
	// For a specific match, you have to provide a specific pubKey.
	desired := cipher.PubKey{}

	if newSubscriptionConfig.PubKey != "" {
		desired, err = cipher.PubKeyFromHex(
			newSubscriptionConfig.PubKey)
		if err != nil {
			return ctx.ErrInvalidRequest(
				err.Error(),
				"pubKey",
				newSubscriptionConfig.PubKey)
		}
	}
	err = self.Outgoing().Connect(newSubscriptionConfig.IP, desired)
	if err != nil {
		return ctx.ErrInternal(err.Error())
	}
	response := JSONResponse{
		Code:   "created",
		Status: 201,
		Detail: "The subscription has been initiated", // may be
	}
	return ctx.JSON(201, response)
}

func (self *SkyhashManager) GetSubscriptionByID(ctx *Context) error {
	subscriptionPubKey, err := ctx.PubKeyFromParam("subscription_id")
	if err != nil {
		return ctx.ErrInvalidRequest(err.Error())
	}

	for c := range self.Outgoing().List() {
		if c.Pub == *subscriptionPubKey {
			ctx.JSON(200, c)
			return nil
		}
	}
	return ctx.ErrNotFound(err.Error(), "pubKey", subscriptionPubKey.Hex())
}

func (self *SkyhashManager) TerminateSubscriptionByID(ctx *Context) error {
	subscriptionPubKey, err := ctx.PubKeyFromParam("subscription_id")
	if err != nil {
		return ctx.ErrInvalidRequest(err.Error())
	}
	self.Outgoing().Terminate(*subscriptionPubKey)
	return ctx.JSON(200, JSONResponse{
		Code:   "terminated",
		Status: 200,
		Detail: "The subscription has been terminated", // or not found
	})
}

func (self *SkyhashManager) ListSubscribers(ctx *Context) error {
	var subscribers []node.Connection
	for c := range self.Incoming().List() {
		subscribers = append(subscribers, c)
	}
	return ctx.JSON(200, subscribers)
}

func (self *SkyhashManager) GetSubscriberByID(ctx *Context) error {
	pub, err := ctx.PubKeyFromParam("subscriber_id")
	if err != nil {
		return ctx.ErrInvalidRequest(err.Error())
	}
	for c := range self.Incoming().List() {
		if c.Pub == *pub {
			return ctx.JSON(200, c)
		}
	}
	return ctx.ErrNotFound(err.Error(), "pubKey", pub.Hex())
}

func (self *SkyhashManager) TerminateSubscriberByID(ctx *Context) error {
	pub, err := ctx.PubKeyFromParam("subscriber_id")
	if err != nil {
		return ctx.ErrInvalidRequest(err.Error())
	}
	self.Incoming().Terminate(*pub)
	return ctx.JSON(200, JSONResponse{
		Code:   "terminated",
		Status: 200,
		Detail: "The subscriber has been terminated", // or not found
	})
}
