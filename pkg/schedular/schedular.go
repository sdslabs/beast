package schedular

import (
	"reflect"
	"fmt"

    log "github.com/sirupsen/logrus"    
)

type Function interface{}

type TaskFunction struct {
	Function Function
	Name string
	ParamTypes map[string]reflect.Type
}

type TaskFunctionRegister struct {
	Functions map[string]TaskFunction
}

func (tfr *TaskFunctionRegister) NewTaskFunctionRegister() *TaskFunctionRegister {
	return &TaskFunctionRegister{
		Functions: make(map[string]TaskFunction)
	}
}

func (tfr *TaskFunctionRegister) AddFunc(function Function) error {
	funcValue := reflect.ValueOf(function)
	if funcValue.Kind() != reflect.Func {
		return fmt.Errorf("Provide function as an argument.")
	}

	funcType := reflect.TypeOf(function)
	paramTypes, err := resolveParamTypes(function)
	if err != nil {
		return fmt.Errorf("Error occured while resolving parameter types.")
	}

	trf.Functions[funcType.Name()] = TaskFunction{
		Function: function,
		Name: funcType.Name()
		ParamTypes: paramTypes,
	}
}

type Schedular struct {
    Register TaskFunctionRegister
}

func resolveParamTypes(function Function) (map[string]reflect.Type, error) {
	funcType := reflect.TypeOf(function)
	if funcValue.Kind() != reflect.Func {
		return nil, fmt.Errorf("Provide function as an argument.")
	}

	paramTypes := make(map[string]reflect.Type)

	for i := 0; i < funcType.NumIn(); i++ {
		typ := funcType.In(i)
		paramTypes[in.Name()] = typ
	}

	return paramTypes, nil
}

