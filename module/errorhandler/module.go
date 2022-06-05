package errorhandler

import (
	"github.com/allentom/haruka"
	"reflect"
)

type ErrorModule struct {
	Handlers []ErrorHandler
}

func NewErrorModule() *ErrorModule {
	return &ErrorModule{
		Handlers: []ErrorHandler{},
	}
}

type ErrorHandler struct {
	Match       interface{}
	Code        string
	Status      int
	ErrorRender func(ctx *haruka.Context, err error) interface{}
}

func (m *ErrorModule) RegisterHandler(handler ErrorHandler) {
	m.Handlers = append(m.Handlers, handler)
}

func (m *ErrorModule) GetHandlerByType(err error) *ErrorHandler {
	errTypeRef := reflect.TypeOf(err)
	for _, handler := range m.Handlers {
		matchType := reflect.TypeOf(handler.Match)
		if errTypeRef == matchType {
			return &handler
		}
	}
	return nil
}
func (m *ErrorModule) RaiseHttpError(context *haruka.Context, err error) {
	handler := m.GetHandlerByType(err)
	if handler == nil {
		context.JSONWithStatus(haruka.JSON{
			"success": false,
			"err":     err.Error(),
			"code":    "9999",
		}, 500)
		return
	}
	if handler.ErrorRender != nil {
		handler.ErrorRender(context, err)
		return
	}
	context.JSONWithStatus(haruka.JSON{
		"success": false,
		"err":     err.Error(),
		"code":    handler.Code,
	}, handler.Status)
}
