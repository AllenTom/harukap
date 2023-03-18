package task

import (
	"errors"
	"fmt"
	"github.com/allentom/haruka"
	"reflect"
)

type TaskModule struct {
	Pool               *TaskPool
	Converter          []interface{}
	ListHandler        haruka.RequestHandler
	GetTaskByIdHandler haruka.RequestHandler
	ErrorHandler       func(context *haruka.Context, err error)
}

func NewTaskModule() *TaskModule {
	module := &TaskModule{
		Pool:      NewTaskPool(),
		Converter: []interface{}{},
	}
	module.ListHandler = func(context *haruka.Context) {
		data, err := module.SerializerTemplateList()
		if err != nil {
			module.ErrorHandler(context, err)
			return
		}
		context.JSON(haruka.JSON{
			"success": true,
			"data":    data,
		})
	}
	module.GetTaskByIdHandler = func(context *haruka.Context) {
		id := context.GetQueryString("id")
		task := module.Pool.GetTaskById(id)
		if task == nil {
			module.ErrorHandler(context, errors.New(fmt.Sprintf("task id = %s not found", id)))
			return
		}
		data, err := module.SerializerTemplate(task)
		if err != nil {
			module.ErrorHandler(context, err)
			return
		}
		context.JSON(haruka.JSON{
			"success": true,
			"data":    data,
		})
	}
	return module
}

type Template struct {
	Id           string      `json:"id"`
	Type         string      `json:"type"`
	Status       string      `json:"status"`
	Created      string      `json:"created"`
	Err          string      `json:"err,omitempty"`
	Output       interface{} `json:"output,omitempty"`
	SubTask      []*Template `json:"subTask,omitempty"`
	StartTime    string      `json:"startTime,omitempty"`
	EndTime      string      `json:"endTime,omitempty"`
	Duration     uint        `json:"duration,omitempty"`
	ParentTaskId string      `json:"parentTaskId,omitempty"`
}

func (t *TaskModule) AddConverter(converters ...interface{}) {
	t.Converter = append(t.Converter, converters...)
}
func (t *TaskModule) SerializerTemplateList() (interface{}, error) {
	list := make([]interface{}, 0)
	for _, task := range t.Pool.Tasks {
		tp, err := t.SerializerTemplate(task)
		if err != nil {
			return nil, err
		}
		list = append(list, tp)
	}
	return list, nil
}
func (t *TaskModule) SerializerTemplate(data Task) (interface{}, error) {
	template := &Template{
		Id:           data.GetId(),
		Type:         data.GetType(),
		Status:       data.GetStatus(),
		Created:      data.GetCreated().Format("2006-01-02 15:04:05"),
		ParentTaskId: data.GetParentTaskId(),
	}
	if data.Error() != nil {
		template.Err = data.Error().Error()
	}
	output, err := data.Output()
	if err != nil {
		return nil, err
	}
	template.Output, err = t.SerializerTemplateOutput(output)
	if err != nil {
		return nil, err
	}
	if data.SubTask() != nil && len(data.SubTask()) > 0 {
		template.SubTask = make([]*Template, 0)
		for _, subTask := range data.SubTask() {
			subTemplate, err := t.SerializerTemplate(subTask)
			if err != nil {
				return nil, err
			}
			template.SubTask = append(template.SubTask, subTemplate.(*Template))
		}
	}
	if !data.GetStartTime().IsZero() {
		template.StartTime = data.GetStartTime().Format("2006-01-02 15:04:05")
	}
	if !data.GetEndTime().IsZero() {
		template.EndTime = data.GetEndTime().Format("2006-01-02 15:04:05")
		template.Duration = uint(data.GetEndTime().Sub(data.GetStartTime()).Milliseconds())
	}
	return template, nil
}
func (t *TaskModule) SerializerTemplateOutput(data interface{}) (interface{}, error) {
	dataTypeRef := reflect.TypeOf(data)
	var dataConverter interface{} = nil
	for _, converter := range t.Converter {
		converterTypeRef := reflect.TypeOf(converter)
		if converterTypeRef.In(0) == dataTypeRef {
			dataConverter = converter
		}
	}
	if dataConverter == nil {
		return data, nil
	}
	dataConverterValueRef := reflect.ValueOf(dataConverter)
	resultValues := dataConverterValueRef.Call([]reflect.Value{reflect.ValueOf(data)})
	if resultValues[1].Interface() == nil {
		return resultValues[0].Interface(), nil
	}
	return resultValues[0].Interface(), resultValues[0].Interface().(error)
}
