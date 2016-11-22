package schema


type dynamicHref struct {
	Schema StructSchema
}

//
//func HrefDynamic(value interface{}) *HRef{
//	schema := ExtractSchema(value)
//	data:= encoder.Serialize(value)
//	return createHref(&dynamicHref{Schema:schema}, cipher.SumSHA256(data))
//}
