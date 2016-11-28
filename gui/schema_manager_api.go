package gui

import (
	"github.com/skycoin/cxo/nodeManager"
)

type schemaApi struct {

}

//RegisterNodeManagerHandlers - create routes for NodeManager
func RegisterSchemaHandlers(router *Router, shm *nodeManager.Manager) {
	// enclose shm into SkyhashManager to be able to add methods
	//lshm := SkyhashManager{Manager: shm}

	sch := &schemaApi{}

	router.GET("/object1/:object/list", sch.List)
	router.GET("/object1/:object/schema", sch.Schema)
	//router.POST("/manager", lshm._StartManager)
	//router.DELETE("/manager", lshm._StopManager)

}

func (api *schemaApi) List(ctx *Context) error {
	objectName := *ctx.Param("object")

	return ctx.Text(200, objectName)
}

func (api *schemaApi) Schema(ctx *Context) error {
	return ctx.Text(200, "Object schema!")
}
