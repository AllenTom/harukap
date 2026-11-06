package task

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/allentom/haruka"
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
	HasSubTask   bool        `json:"hasSubTask"`
	SubTaskCount int         `json:"subTaskCount"`
	SubTask      []*Template `json:"subTask,omitempty"`
	StartTime    string      `json:"startTime,omitempty"`
	EndTime      string      `json:"endTime,omitempty"`
	Duration     uint        `json:"duration,omitempty"`
	ParentTaskId string      `json:"parentTaskId,omitempty"`
}

func (t *TaskModule) AddConverter(converters ...interface{}) {
	t.Converter = append(t.Converter, converters...)
}

// SerializerTemplateList serializes root tasks for list APIs.
// Nested subtasks are summarized with hasSubTask/subTaskCount, never expanded.
func (t *TaskModule) SerializerTemplateList() (interface{}, error) {
	list := make([]interface{}, 0)
	for _, task := range t.Pool.Tasks {
		tp, err := t.serializeTemplate(task, false)
		if err != nil {
			return nil, err
		}
		list = append(list, tp)
	}
	return list, nil
}
func formatTaskTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// SerializerTemplate serializes a single task for detail APIs,
// expanding only the direct children (one level).
func (t *TaskModule) SerializerTemplate(data Task) (interface{}, error) {
	return t.serializeTemplate(data, true)
}

func (t *TaskModule) serializeTemplate(data Task, includeDirectChildren bool) (*Template, error) {
	template := &Template{
		Id:           data.GetId(),
		Type:         data.GetType(),
		Status:       data.GetStatus(),
		Created:      formatTaskTime(data.GetCreated()),
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
	subTasks := data.SubTask()
	template.SubTaskCount = len(subTasks)
	template.HasSubTask = template.SubTaskCount > 0
	if includeDirectChildren && template.HasSubTask {
		template.SubTask = make([]*Template, 0, len(subTasks))
		for _, subTask := range subTasks {
			subTemplate, err := t.serializeTemplate(subTask, false)
			if err != nil {
				return nil, err
			}
			template.SubTask = append(template.SubTask, subTemplate)
		}
	}
	if !data.GetStartTime().IsZero() {
		template.StartTime = formatTaskTime(data.GetStartTime())
	}
	if !data.GetEndTime().IsZero() {
		template.EndTime = formatTaskTime(data.GetEndTime())
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
