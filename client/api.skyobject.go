package client

import (
	"fmt"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/gui"
	"github.com/skycoin/skycoin/src/cipher"
)

type schemaApi struct {
	container    skyobject.ISkyObjects
	synchronizer skyobject.ISynchronizer
}

type objectLink struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func SkyObjectsAPI(container skyobject.ISkyObjects, synchronizer skyobject.ISynchronizer) *schemaApi {
	return &schemaApi{container:container, synchronizer : synchronizer}
}

func (api *schemaApi) Register(router *gui.Router) {
	router.GET("/object1/_stat", api.Statistic)
	router.GET("/object1/_schemas", api.Schemas)
	router.GET("/object1/:schema/schema", api.Schema)
	router.GET("/object1/:schema/list", api.ObjectsBySchema)
	router.GET("/object1/:schema/list/:id", api.Object)
	//router.GET("/object1/:schema/list/:id", sch.Object)
	router.GET("/object1/data/:id", api.Object)
	router.POST("/object1/data/:id/sync", api.SyncObject)
}

func (api *schemaApi) ObjectsBySchema(ctx *gui.Context) error {
	schemaName := *ctx.Param("schema")
	schemaKey, _ := api.container.GetSchemaKey(schemaName)
	fmt.Println(schemaKey)
	keys := api.container.GetAllBySchema(schemaKey)
	fmt.Println(keys)
	res := []objectLink{}
	for _, k := range keys {
		ref := skyobject.HashObject{Ref:k}
		ref.SetData(k[:])
		res = append(res, objectLink{ID:k.Hex(), Name:ref.String(api.container)})
	}
	return ctx.JSON(200, res)
}

func (api *schemaApi) Object(ctx *gui.Context) error {
	id, err := cipher.SHA256FromHex(*ctx.Param("id"))
	if (err != nil) {
		return ctx.ErrNotFound(err.Error())
	}
	fmt.Println("id", id)

	item := api.container.LoadFields(id)
	fmt.Println("item: ", item)
	return ctx.JSON(200, item)
}

func (api *schemaApi) Schemas(ctx *gui.Context) error {
	return ctx.JSON(200, api.container.GetSchemas())
}

func (api *schemaApi) Schema(ctx *gui.Context) error {
	objectName := *ctx.Param("schema")
	schema := api.container.GetSchema(objectName)
	fmt.Println("Schema", schema)
	return ctx.JSON(200, schema)
}

func (api *schemaApi) Statistic(ctx *gui.Context) error {
	stat := api.container.Statistic()
	return ctx.JSON(200, stat)
}

func (api *schemaApi) SyncObject(ctx *gui.Context) error {
	id, err := cipher.SHA256FromHex(*ctx.Param("id"))
	if (err != nil) {
		return ctx.ErrNotFound(err.Error())
	}
	//TODO: Validate for root type
	root := skyobject.HashRoot{Ref:id}
	fmt.Println("Sync object")
	res := <-api.synchronizer.Sync(root)
	return ctx.JSON(200, res)
}
