package openapi

import (
	"errors"
	"fmt"

	"github.com/daarxwalker/enjin/app"
	"github.com/daarxwalker/enjin/app/routespec"

	openapigo "github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
)

var _ app.Spec = (*Spec)(nil)

type Spec struct {
	reflector *openapi3.Reflector
}

func New(options ...Option) *Spec {
	reflector := new(openapi3.Reflector)
	reflector.Spec = &openapi3.Spec{Openapi: "3.1.0"}
	for _, option := range options {
		option(reflector)
	}
	return &Spec{reflector: reflector}
}

func (s *Spec) AddOperation(method, pattern string, operation routespec.Operation) error {
	op, err := s.reflector.NewOperationContext(method, pattern)
	if err != nil {
		return fmt.Errorf("create operation %s %s failed: %w", method, pattern, err)
	}
	s.applyInfo(op, operation)
	if operation.Request != nil {
		op.AddReqStructure(operation.Request)
	}
	for _, response := range operation.Responses {
		op.AddRespStructure(response.Body, func(cu *openapigo.ContentUnit) {
			cu.HTTPStatus = response.Status
			if response.ContentType != "" {
				cu.ContentType = response.ContentType
			}
			if response.Description != "" {
				cu.Description = response.Description
			}
		})
	}
	if err = s.applyParams(op, operation.Params); err != nil {
		return err
	}
	for _, security := range operation.Security {
		op.AddSecurity(security.Name, security.Scopes...)
	}
	if err = s.reflector.AddOperation(op); err != nil {
		return fmt.Errorf("add operation %s %s failed: %w", method, pattern, err)
	}
	return nil
}

func (s *Spec) MarshalJSON() ([]byte, error) {
	return s.reflector.Spec.MarshalJSON()
}

func (s *Spec) applyInfo(op openapigo.OperationContext, operation routespec.Operation) {
	if operation.ID != "" {
		op.SetID(operation.ID)
	}
	if operation.Summary != "" {
		op.SetSummary(operation.Summary)
	}
	if operation.Description != "" {
		op.SetDescription(operation.Description)
	}
	if len(operation.Tags) > 0 {
		op.SetTags(operation.Tags...)
	}
	if operation.Deprecated {
		op.SetIsDeprecated(true)
	}
}

func (s *Spec) applyParams(op openapigo.OperationContext, params []routespec.Param) error {
	if len(params) == 0 {
		return nil
	}
	exposer, ok := op.(openapi3.OperationExposer)
	if !ok {
		return errors.New("operation does not expose openapi3 operation")
	}
	target := exposer.Operation()
	for _, param := range params {
		parameter := (&openapi3.Parameter{}).
			WithName(param.Name).
			WithIn(openapi3.ParameterIn(param.In)).
			WithRequired(param.Required).
			WithDescription(param.Description).
			WithSchema(openapi3.SchemaOrRef{
				Schema: (&openapi3.Schema{}).WithType(openapi3.SchemaTypeString),
			})
		target.Parameters = append(target.Parameters, parameter.ToParameterOrRef())
	}
	return nil
}
