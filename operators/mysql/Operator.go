package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/simplecontainer/smr/pkg/httpcontract"
	"github.com/simplecontainer/smr/pkg/operators"
	"reflect"
)

func (operator *Operator) Run(operation string, args ...interface{}) httpcontract.ResponseOperator {
	reflected := reflect.TypeOf(operator)
	reflectedValue := reflect.ValueOf(operator)

	for i := 0; i < reflected.NumMethod(); i++ {
		method := reflected.Method(i)

		if operation == method.Name {
			inputs := make([]reflect.Value, len(args))

			for i, _ := range args {
				inputs[i] = reflect.ValueOf(args[i])
			}

			returnValue := reflectedValue.MethodByName(operation).Call(inputs)

			return returnValue[0].Interface().(httpcontract.ResponseOperator)
		}
	}

	return httpcontract.ResponseOperator{
		HttpStatus:       400,
		Explanation:      "server doesn't support requested functionality",
		ErrorExplanation: "implementation is missing",
		Error:            true,
		Success:          false,
		Data:             nil,
	}
}

func (operator *Operator) ListSupported(args ...interface{}) httpcontract.ResponseOperator {
	reflected := reflect.TypeOf(operator)

	supportedOperations := map[string]any{}
	supportedOperations["SupportedOperations"] = []string{}

OUTER:
	for i := 0; i < reflected.NumMethod(); i++ {
		method := reflected.Method(i)
		for _, forbiddenOperator := range invalidOperators {
			if forbiddenOperator == method.Name {
				continue OUTER
			}
		}

		supportedOperations["SupportedOperations"] = append(supportedOperations["SupportedOperations"].([]string), method.Name)
	}

	return httpcontract.ResponseOperator{
		HttpStatus:       200,
		Explanation:      "",
		ErrorExplanation: "",
		Error:            true,
		Success:          false,
		Data:             supportedOperations,
	}
}

func (operator *Operator) DatabaseReady(request operators.Request) httpcontract.ResponseOperator {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/?timeout=5s", request.Data["username"], request.Data["password"], request.Data["ip"], request.Data["port"]))
	if err != nil {
		return httpcontract.ResponseOperator{
			HttpStatus:       400,
			Explanation:      "database connection can't be opened",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
			Data:             nil,
		}
	}

	defer db.Close()
	err = db.Ping()

	if err != nil {
		return httpcontract.ResponseOperator{
			HttpStatus:       400,
			Explanation:      "database can't be pinged",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
			Data:             nil,
		}
	}

	return httpcontract.ResponseOperator{
		HttpStatus:       200,
		Explanation:      "database connection is ready",
		ErrorExplanation: "",
		Error:            false,
		Success:          true,
		Data:             nil,
	}
}

// Exported
var Mysql Operator
