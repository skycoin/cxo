package main

//
// import (
// 	"fmt"
// 	"github.com/skycoin/cxo/data"
// 	"github.com/skycoin/cxo/skyobject"
// )
//
// // Parent type.
// type Parent struct {
// 	Name     string
// 	Children skyobject.HashArray
// }
//
// // Child type.
// type Child struct {
// 	Name string
// }
//
// func main() {
// 	container := skyobject.SkyObjects(data.NewDB())
// 	container.RegisterSchema(Parent{})
// 	container.RegisterSchema(Child{})
//
// 	getParentRef := func() skyobject.ObjRef {
// 		children := []Child{{"Ben"}, {"James"}, {"Ryan"}}
// 		childrenRef := skyobject.NewArray(children)
// 		container.Save(&childrenRef)
//
// 		parent := Parent{Name: "Bob", Children: childrenRef}
// 		parentRef := skyobject.NewObject(parent)
// 		container.Save(&parentRef)
//
// 		return container.GetObjRef(parentRef.Ref)
// 	}
//
// 	parentRef := getParentRef()
//
// 	// Retrieve the 'Children' field of the Parent object as an 'ObjRef'.
// 	childrenRef, e := parentRef.GetFieldAsObj("Children")
// 	if e != nil {
// 		fmt.Println("Unable to retrieve field. Error:", e)
// 	}
//
// 	// We know that 'childrenRef' references a HashArray, not a HashObject.
// 	// We can retrieve all the array elements using the `GetValuesAsObjArray` member function of 'childrenRef'.
// 	childRefArray, e := childrenRef.GetValuesAsObjArray()
// 	if e != nil {
// 		fmt.Println("Unable to retrieve elements. Error:", e)
// 	}
//
// 	// 'childRefArray' is of type '[]ObjRef'.
// 	// We can now deserialize all the child references into child objects.
// 	fmt.Println("Children:")
//
// 	for _, childRef := range childRefArray {
// 		var child Child
// 		childRef.Deserialize(&child)
//
// 		fmt.Println(" -", child.Name)
// 	}
//
// 	// Schemas.
// 	fmt.Println("Schemas:")
// 	schemas := container.GetSchemas()
// 	for _, schema := range schemas {
// 		fmt.Println("-", schema.Name)
// 	}
//
// 	// Inspect.
// 	fmt.Println("Inspect:")
// 	container.Inspect()
//
// 	func() {
// 		// Get all keys of objects with Schema type 'Child'.
// 		schemaKey, ok := container.GetSchemaKey("Child")
// 		if ok == false {
// 			fmt.Println("Woops! This Schema doesn't exist in the container.")
// 			return
// 		}
// 		childrenKeys := container.GetAllBySchema(schemaKey)
//
// 		// Get all children.
// 		fmt.Println("Children:")
//
// 		for _, childKey := range childrenKeys {
// 			var child Child
// 			childRef := container.GetObjRef(childKey)
// 			childRef.Deserialize(&child)
//
// 			fmt.Println(" -", child.Name)
// 		}
// 	}()
// }
