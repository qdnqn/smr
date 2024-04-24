package objects

import (
	"encoding/json"
	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
	"reflect"
	"smr/pkg/database"
	"smr/pkg/logger"
	"time"
)

func New() *Object {
	return &Object{
		definition: map[string]any{},
		exists:     false,
		changed:    false,
		created:    time.Now(),
		updated:    time.Now(),
	}
}

func ConvertToMap(jsonData []byte) (map[string]any, error) {
	data := map[string]any{}
	err := json.Unmarshal(jsonData, &data)

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (obj *Object) Add(registryObjects map[string]Object, db *badger.DB, format database.FormatStructure, data string) error {
	err := database.Put(db, format.ToString(), string(data))

	if err != nil {
		return err
	}

	format.Key = "updated"
	err = database.Put(db, format.ToString(), time.Now().Format(time.RFC3339))

	timeNow := time.Now()

	if err == nil {
		obj.updated = timeNow
	} else {
		return err
	}

	format.Key = "created"
	err = database.Put(db, format.ToString(), time.Now().Format(time.RFC3339))

	if err == nil {
		obj.created = timeNow
	} else {
		return err
	}

	return err
}

func (obj *Object) Update(registryObjects map[string]Object, db *badger.DB, format database.FormatStructure, data string) error {
	err := database.Put(db, format.ToString(), string(data))

	if err != nil {
		return err
	}

	format.Key = "updated"
	err = database.Put(db, format.ToString(), time.Now().Format(time.RFC3339))

	if err == nil {
		obj.updated = time.Now()
	}

	return err
}

func (obj *Object) Find(registryObjects map[string]Object, db *badger.DB, format database.FormatStructure) error {
	val, err := database.Get(db, format.ToString())

	if err == nil {
		data := make(map[string]any)
		err = json.Unmarshal([]byte(val), &data)

		if err != nil {
			return err
		}

		obj.definition = data
	} else {
		return err
	}

	format.Key = "created"

	val, err = database.Get(db, format.ToString())

	if err == nil {
		obj.created, err = time.Parse(time.RFC3339, val)

		if err != nil {
			return err
		}
	} else {
		return err
	}

	format.Key = "updated"

	val, err = database.Get(db, format.ToString())

	if err == nil {
		obj.created, err = time.Parse(time.RFC3339, val)

		if err != nil {
			return err
		}
	} else {
		return err
	}

	obj.changed = false
	obj.exists = true

	return nil
}

func (obj *Object) Remove(registryObjects map[string]Object, db *badger.DB, format database.FormatStructure) {
}

func (obj *Object) Diff(definition string) bool {
	data := make(map[string]any)
	err := json.Unmarshal([]byte(definition), &data)

	if err != nil {
		logger.Log.Error("failed to marshal json to map", zap.String("json", definition))
		return true
	}

	if reflect.DeepEqual(obj.definition, data) {
		obj.changed = false
	} else {
		obj.changed = true
	}

	return obj.changed
}

func (obj *Object) Exists() bool {
	return obj.exists
}

func (obj *Object) ChangeDetected() bool {
	return obj.changed
}