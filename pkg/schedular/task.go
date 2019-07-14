package schedular

import (
	"reflect"
	"fmt"
	"time"
	"crypto/sha1"

    log "github.com/sirupsen/logrus"    
)

type Function interface{}
type FuncParam interface{}

type FunctionID string
type TaskID string

type TaskFunction struct {
	Name string

	Function Function
	Params []FuncParam
}

func (tf *TaskFunction) Run() {
	tf.Function(tf.Params...)
}

func (tf *TaskFunc) GetFunctionID() FunctionID {
	hash := sha1.New()

	_, _ = io.WriteString(hash, tf.Name)
	_, _ = io.WriteString(hash, fmt.Sprintf("%+v", tf.Params))

	return FunctionID(fmt.Sprintf("%x", hash.Sum(nil)))
}

type TaskFunctionRegister struct {
	Functions map[FunctionID]TaskFunction
}

func (tfr *TaskFunctionRegister) NewTaskFunctionRegister() *TaskFunctionRegister {
	return &TaskFunctionRegister{
		Functions: make(map[FunctionID]TaskFunction)
	}
}

func (tfr *TaskFunctionRegister) AddFunction(function Function, params ...FuncParam) (FunctionID, error) {
	var funcID FunctionID

	funcValue := reflect.ValueOf(function)
	if funcValue.Kind() != reflect.Func {
		return funcID, fmt.Errorf("The function provided is not of type Function")
	}

	funcType := reflect.TypeOf(function)
	err := validateParamTypes(function, ...params)
	if err != nil {
		return funcID, err
	}

	tf := TaskFunction{
		Function: function,
		Name: funcType.Name()
		Params: params,
	}

	funcID = tf.GetFunctionID()
	trf.Functions[funcID] = tf

	return funcID, nil
}

type Schedule struct {
	IsRecurring bool
	LastRun     time.Time
	NextRun     time.Time
	Duration    time.Duration
}

type Task struct {
	Schedule Schedule

	FunctionID FunctionID
	id TaskID
}

func NewTask(schedule Schedule, funcID FunctionID) *Task {
	return &Task{
		Schedule schedule,

		FunctionID funcID,
	}
}

func (task *Task) getTaskID() TaskID {
	hash := sha1.New()

	_, _ = io.WriteString(hash, string(task.FunctionID))
	_, _ = io.WriteString(hash, fmt.Sprintf("%+v", tf.Schedule))

	return TaskID(fmt.Sprintf("%x", hash.Sum(nil)))
}


func (task *Task) GetTaskID() TaskID {
	if string(task.id) == "" {
		task.id = task.getTaskID()
	}

	return task.id
}

func (task *Task) IsDue() bool {
	return time.Now() == task.Schedule.NextRun || time.Now().After(task.Schedule.NextRun)
}

type TaskMap map[TaskID]*Task

func NewTaskMap() TaskMap {
	return make(TaskMap)
}

func (tMap *TaskMap) AddTask(schedule Schedule, funcID FunctionID) {
	task := NewTask(schedule, funcID)

	tMap[task.GetTaskID()] = task
}

func validateParamTypes(function Function, params ...FuncParam) error {
	funcType := reflect.TypeOf(function)
	if funcType.Kind() != reflect.Func {
		return fmt.Errorf("Provided function is not a valid function")
	}

	if funcType.NumIn() != len(params) {
		return fmt.Errorf("Parameters not valid required %d given %d", funcType.NumIn(), len(params))
	}

	for i, param := range params {
		typ := funcType.In(i)
		if typ != reflect.TypeOf(param) {
			return fmt.Errorf("Parameter type for param %s is not valid", in.Name())
		}
	}

	return nil
}
