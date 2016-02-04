package handler

import (
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
	"github.com/oursky/skygear/router"
	"github.com/oursky/skygear/skydb"
	"github.com/oursky/skygear/skyerr"
)

type schemaResponse struct {
	Schemas map[string]schemaFieldList `json:"record_types"`
}

type schemaFieldList struct {
	Fields []schemaField `mapstructure:"fields" json:"fields"`
}

func (s schemaFieldList) Len() int {
	return len(s.Fields)
}

func (s schemaFieldList) Swap(i, j int) {
	s.Fields[i], s.Fields[j] = s.Fields[j], s.Fields[i]
}

func (s schemaFieldList) Less(i, j int) bool {
	return strings.Compare(s.Fields[i].Name, s.Fields[j].Name) < 0
}

type schemaField struct {
	Name     string `mapstructure:"name" json:"name"`
	TypeName string `mapstructure:"type" json:"type"`
}

func (resp *schemaResponse) Encode(data map[string]skydb.RecordSchema) {
	resp.Schemas = make(map[string]schemaFieldList)
	for recordType, schema := range data {
		fieldList := schemaFieldList{}
		for fieldName, val := range schema {
			if strings.HasPrefix(fieldName, "_") {
				continue
			}

			fieldList.Fields = append(fieldList.Fields, schemaField{
				Name:     fieldName,
				TypeName: val.ToSimpleName(),
			})
		}
		sort.Sort(fieldList)
		resp.Schemas[recordType] = fieldList
	}
}

/*
SchemaRenameHandler handles the action of renaming column
curl -X POST -H "Content-Type: application/json" \
  -d @- http://localhost:3000/ <<EOF
{
	"access_token":"ee41c969-cc1f-422b-985d-ddb2217b90f8",
	"action":"schema:rename",
	"database_id":"_public",
	"record_type":"student",
	"item_type":"field",
	"item_name":"score",
	"new_name":"exam_score"
}
EOF
*/
type SchemaRenameHandler struct {
	DevOnly       router.Processor `preprocessor:"dev_only"`
	DBConn        router.Processor `preprocessor:"dbconn"`
	InjectDB      router.Processor `preprocessor:"inject_db"`
	preprocessors []router.Processor
}

func (h *SchemaRenameHandler) Setup() {
	h.preprocessors = []router.Processor{
		h.DevOnly,
		h.DBConn,
		h.InjectDB,
	}
}

func (h *SchemaRenameHandler) GetPreprocessors() []router.Processor {
	return h.preprocessors
}

type schemaRenamePayload struct {
	RecordType string `mapstructure:"record_type"`
	OldName    string `mapstructure:"item_name"`
	NewName    string `mapstructure:"new_name"`
}

func (payload *schemaRenamePayload) Decode(data map[string]interface{}) skyerr.Error {
	if err := mapstructure.Decode(data, payload); err != nil {
		return skyerr.NewError(skyerr.BadRequest, "fails to decode the request payload")
	}
	return payload.Validate()
}

func (payload *schemaRenamePayload) Validate() skyerr.Error {
	if payload.RecordType == "" || payload.OldName == "" || payload.NewName == "" {
		return skyerr.NewError(skyerr.InvalidArgument, "data in the specified request is invalid")
	}
	if strings.HasPrefix(payload.RecordType, "_") ||
		strings.HasPrefix(payload.OldName, "_") ||
		strings.HasPrefix(payload.NewName, "_") {
		return skyerr.NewError(skyerr.InvalidArgument, "attempts to change reserved key")
	}
	return nil
}

func (h *SchemaRenameHandler) Handle(rpayload *router.Payload, response *router.Response) {
	payload := &schemaRenamePayload{}
	skyErr := payload.Decode(rpayload.Data)
	if skyErr != nil {
		response.Err = skyErr
		return
	}

	db := rpayload.Database

	if err := db.RenameSchema(payload.RecordType, payload.OldName, payload.NewName); err != nil {
		response.Err = skyerr.NewError(skyerr.ResourceNotFound, err.Error())
		return
	}

	results, err := db.GetRecordSchemas()
	if err != nil {
		response.Err = skyerr.NewError(skyerr.UnexpectedError, err.Error())
		return
	}

	resp := &schemaResponse{}
	resp.Encode(results)

	response.Result = resp
}

/*
SchemaDeleteHandler handles the action of deleting column
curl -X POST -H "Content-Type: application/json" \
  -d @- http://localhost:3000/ <<EOF
{
	"access_token":"ee41c969-cc1f-422b-985d-ddb2217b90f8",
	"action":"schema:delete",
	"database_id":"_public",
	"record_type":"student",
	"item_type":"field",
	"item_name":"score"
}
EOF
*/
type SchemaDeleteHandler struct {
	DevOnly       router.Processor `preprocessor:"dev_only"`
	DBConn        router.Processor `preprocessor:"dbconn"`
	InjectDB      router.Processor `preprocessor:"inject_db"`
	preprocessors []router.Processor
}

func (h *SchemaDeleteHandler) Setup() {
	h.preprocessors = []router.Processor{
		h.DevOnly,
		h.DBConn,
		h.InjectDB,
	}
}

func (h *SchemaDeleteHandler) GetPreprocessors() []router.Processor {
	return h.preprocessors
}

type schemaDeletePayload struct {
	RecordType string `mapstructure:"record_type"`
	ColumnName string `mapstructure:"item_name"`
}

func (payload *schemaDeletePayload) Decode(data map[string]interface{}) skyerr.Error {
	if err := mapstructure.Decode(data, payload); err != nil {
		return skyerr.NewError(skyerr.BadRequest, "fails to decode the request payload")
	}
	return payload.Validate()
}

func (payload *schemaDeletePayload) Validate() skyerr.Error {
	if payload.RecordType == "" || payload.ColumnName == "" {
		return skyerr.NewError(skyerr.InvalidArgument, "data in the specified request is invalid")
	}
	if strings.HasPrefix(payload.RecordType, "_") ||
		strings.HasPrefix(payload.ColumnName, "_") {
		return skyerr.NewError(skyerr.InvalidArgument, "attempts to change reserved key")
	}
	return nil
}

func (h *SchemaDeleteHandler) Handle(rpayload *router.Payload, response *router.Response) {
	payload := &schemaDeletePayload{}
	skyErr := payload.Decode(rpayload.Data)
	if skyErr != nil {
		response.Err = skyErr
		return
	}

	db := rpayload.Database

	if err := db.DeleteSchema(payload.RecordType, payload.ColumnName); err != nil {
		response.Err = skyerr.NewError(skyerr.ResourceNotFound, err.Error())
		return
	}

	results, err := db.GetRecordSchemas()
	if err != nil {
		response.Err = skyerr.NewError(skyerr.UnexpectedError, err.Error())
		return
	}

	resp := &schemaResponse{}
	resp.Encode(results)

	response.Result = resp
}

/*
SchemaCreateHandler handles the action of creating new columns
curl -X POST -H "Content-Type: application/json" \
  -d @- http://localhost:3000/ <<EOF
{
	"access_token":"ee41c969-cc1f-422b-985d-ddb2217b90f8",
	"action":"schema:create",
	"database_id":"_public",
	"record_types":{
		"student": {
			"fields":[
				{"name": "age", 	"type": "number"},
				{"name": "nickname" "type": "string"}
			]
		}
	}
}
EOF
*/
type SchemaCreateHandler struct {
	DevOnly       router.Processor `preprocessor:"dev_only"`
	DBConn        router.Processor `preprocessor:"dbconn"`
	InjectDB      router.Processor `preprocessor:"inject_db"`
	preprocessors []router.Processor
}

func (h *SchemaCreateHandler) Setup() {
	h.preprocessors = []router.Processor{
		h.DevOnly,
		h.DBConn,
		h.InjectDB,
	}
}

func (h *SchemaCreateHandler) GetPreprocessors() []router.Processor {
	return h.preprocessors
}

type schemaCreatePayload struct {
	RawSchemas map[string]schemaFieldList `mapstructure:"record_types"`

	Schemas map[string]skydb.RecordSchema
}

func (payload *schemaCreatePayload) Decode(data map[string]interface{}) skyerr.Error {
	if err := mapstructure.Decode(data, payload); err != nil {
		return skyerr.NewError(skyerr.BadRequest, "fails to decode the request payload")
	}

	payload.Schemas = make(map[string]skydb.RecordSchema)
	for recordType, schema := range payload.RawSchemas {
		payload.Schemas[recordType] = make(skydb.RecordSchema)
		for _, field := range schema.Fields {
			var err error
			payload.Schemas[recordType][field.Name], err = skydb.SimpleNameToFieldType(field.TypeName)
			if err != nil {
				return skyerr.NewError(skyerr.InvalidArgument, "unexpected field type")
			}
		}
	}

	return payload.Validate()
}

func (payload *schemaCreatePayload) Validate() skyerr.Error {
	for recordType, schema := range payload.Schemas {
		if strings.HasPrefix(recordType, "_") {
			return skyerr.NewError(skyerr.InvalidArgument, "attempts to create reserved table")
		}
		for fieldName := range schema {
			if strings.HasPrefix(fieldName, "_") {
				return skyerr.NewError(skyerr.InvalidArgument, "attempts to create reserved field")
			}
		}
	}
	return nil
}

func (h *SchemaCreateHandler) Handle(rpayload *router.Payload, response *router.Response) {
	log.Debugf("%+v\n", rpayload)

	payload := &schemaCreatePayload{}
	skyErr := payload.Decode(rpayload.Data)
	if skyErr != nil {
		response.Err = skyErr
		return
	}

	db := rpayload.Database

	for recordType, recordSchema := range payload.Schemas {
		err := db.Extend(recordType, recordSchema)
		if err != nil {
			response.Err = skyerr.NewError(skyerr.IncompatibleSchema, err.Error())
			return
		}
	}

	results, err := db.GetRecordSchemas()
	if err != nil {
		response.Err = skyerr.NewError(skyerr.UnexpectedError, err.Error())
		return
	}

	resp := &schemaResponse{}
	resp.Encode(results)

	response.Result = resp
}

/*
SchemaFetchHandler handles the action of returing information of record schema
curl -X POST -H "Content-Type: application/json" \
  -d @- http://localhost:3000/ <<EOF
{
	"access_token":"ee41c969-cc1f-422b-985d-ddb2217b90f8",
	"action":"schema:fetch",
	"database_id":"_public"
}
EOF
*/
type SchemaFetchHandler struct {
	DevOnly       router.Processor `preprocessor:"dev_only"`
	DBConn        router.Processor `preprocessor:"dbconn"`
	InjectDB      router.Processor `preprocessor:"inject_db"`
	preprocessors []router.Processor
}

func (h *SchemaFetchHandler) Setup() {
	h.preprocessors = []router.Processor{
		h.DevOnly,
		h.DBConn,
		h.InjectDB,
	}
}

func (h *SchemaFetchHandler) GetPreprocessors() []router.Processor {
	return h.preprocessors
}

func (h *SchemaFetchHandler) Handle(rpayload *router.Payload, response *router.Response) {
	db := rpayload.Database

	results, err := db.GetRecordSchemas()
	if err != nil {
		response.Err = skyerr.NewError(skyerr.UnexpectedError, err.Error())
		return
	}

	resp := &schemaResponse{}
	resp.Encode(results)

	response.Result = resp
}
