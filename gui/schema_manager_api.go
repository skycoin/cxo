package gui

import (
	"github.com/skycoin/cxo/skyobject"
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
)

type schemaApi struct {
	sm skyobject.ISkyObjects
}

type objectLink struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

//RegisterNodeManagerHandlers - create routes for NodeManager
func RegisterSchemaHandlers(router *Router, schemaProvider skyobject.ISkyObjects) {
	// enclose shm into SkyhashManager to be able to add methods
	//lshm := SkyhashManager{Manager: shm}

	sch := &schemaApi{sm:schemaProvider}

	router.GET("/object1/_stat", sch.Statistic)
	router.GET("/object1/_schemas", sch.Schemas)
	router.GET("/object1/:schema/schema", sch.Schema)
	router.GET("/object1/:schema/list", sch.ObjectsBySchema)
	router.GET("/object1/:schema/list/:id", sch.Object)
	router.GET("/object1/data/:id", sch.Object)
}

func (api *schemaApi) ObjectsBySchema(ctx *Context) error {
	schemaName := *ctx.Param("schema")
	schemaKey, _ := api.sm.GetSchemaKey(schemaName)
	fmt.Println(schemaKey)
	keys := api.sm.GetAllBySchema(schemaKey)
	res := []objectLink{}
	for _, k := range keys {
		ref := skyobject.HashLink{Ref:k}
		ref.SetData(k[:])
		res = append(res, objectLink{ID:k.Hex(), Name:ref.String(api.sm)})
	}
	return ctx.JSON(200, res)
}

func (api *schemaApi) Object(ctx *Context) error {
	id, err := cipher.SHA256FromHex(*ctx.Param("id"))
	if (err != nil) {
		return ctx.ErrNotFound(err.Error())
	}
	fmt.Println("id", id)

	item := api.sm.LoadFields(id)
	////item["ID"] = *ctx.Param("id")
	//
	return ctx.JSON(200, item)
	//fmt.Println("Getting object")
	//return ctx.JSON(200, "")
}

func (api *schemaApi) Schemas(ctx *Context) error {
	return ctx.JSON(200, api.sm.GetSchemas())
}

func (api *schemaApi) Schema(ctx *Context) error {
	objectName := *ctx.Param("schema")
	schema := api.sm.GetSchema(objectName)
	fmt.Println("Schema", schema)
	return ctx.JSON(200, schema)
}

func (api *schemaApi) Statistic(ctx *Context) error {
	stat := api.sm.Statistic()
	return ctx.JSON(200, stat)
	//return ctx.JSON(200, "")
}
