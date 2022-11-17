package main

import (
	"github.com/getkin/kin-openapi/openapi3"
	"reflect"
	"testing"
)

func Test_isOpenApiV3(t *testing.T) {
	type args struct {
		yaml map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Openapi V3", args{map[string]interface{}{"openapi": "3.0"}}, true},
		{"Swagger V2", args{map[string]interface{}{"swagger": "2.0"}}, false},
		{"something", args{map[string]interface{}{"foo": "2.0"}}, false},
		{"nil", args{map[string]interface{}{}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOpenApiV3(tt.args.yaml); got != tt.want {
				t.Errorf("isOpenApiV3() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isValidApiDefinition(t *testing.T) {
	type args struct {
		m map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"OpenAPI V3", args{map[string]interface{}{"openapi": "3.0.3"}}, true},
		{"Swagger V2", args{map[string]interface{}{"swagger": "2.0"}}, true},
		{"Something", args{map[string]interface{}{"foo": "3.0"}}, false},
		{"Empty", args{map[string]interface{}{}}, false},
		{"nil", args{nil}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidApiDefinition(tt.args.m); got != tt.want {
				t.Errorf("isValidApiDefinition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getOperations(t *testing.T) {
	type args struct {
		pathItem *openapi3.PathItem
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"GET Operation", args{&openapi3.PathItem{Get: openapi3.NewOperation()}}, []string{"GET"}},
		{"POST Operation", args{&openapi3.PathItem{Post: openapi3.NewOperation()}}, []string{"POST"}},
		{"NIL Operation", args{&openapi3.PathItem{}}, nil},
		{"PATCH Operation", args{&openapi3.PathItem{Patch: openapi3.NewOperation()}}, []string{"PATCH"}},
		{"GET and POST Operation", args{&openapi3.PathItem{Get: openapi3.NewOperation(), Post: openapi3.NewOperation()}}, []string{"GET", "POST"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getOperations(tt.args.pathItem); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getOperations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_splitPathString(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"Operation and path", args{"GET /test"}, []string{"GET", "/test"}},
		{"Operation only", args{"GET "}, nil},
		{"Path only", args{" /GET"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitPathString(tt.args.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitPathString() = %v, want %v", got, tt.want)
			}
		})
	}
}
