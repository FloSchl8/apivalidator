package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/invopop/yaml"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) domready(ctx context.Context) {
	runtime.EventsOn(ctx, "notification", func(optionalData ...interface{}) {
		if len(optionalData) == 0 {
			return
		}
		data, ok := optionalData[0].(map[string]interface{})
		if !ok {
			runtime.LogError(ctx, "Failed to parse optionalData on notification")
			return
		}

		parsedAsJSON, err := json.Marshal(data)
		if err != nil {
			runtime.LogErrorf(ctx, "Failed to generate JSON %v", err)
			return
		}

		reqData := struct {
			DialogType string `json:"type"`
			Title      string `json:"title"`
			Message    string `json:"message"`
		}{}
		if err := json.Unmarshal(parsedAsJSON, &reqData); err != nil {
			runtime.LogErrorf(ctx, "Failed to parse generated JSON %v", err)
			return
		}

		_, _ = runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.DialogType(reqData.DialogType),
			Title:   reqData.Title,
			Message: reqData.Message,
		})
	})
}

func (a *App) Validate(spec string, payload string, path string) map[string][]string {

	if spec == "" || payload == "" || path == "" {
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Error",
			Message: "Missing input",
		})
		return nil
	}

	var apiSpecMap = map[string]interface{}{}

	if err := yaml.Unmarshal([]byte(spec), &apiSpecMap); err != nil {
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Error",
			Message: "Unable to load definition",
		})
		return nil
	}

	if !isValidApiDefinition(apiSpecMap) {
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Error",
			Message: "No valid definition. Definition must contain 'swagger' or 'openApi'",
		})
		return nil
	}

	var api *openapi3.T
	if isOpenApiV3(apiSpecMap) {
		api = loadOpenApiDoc(spec)
	} else {
		api = loadSwaggerDoc(spec)
	}

	router, err := gorillamux.NewRouter(api)
	if err != nil {
		log.Fatal(err)
	}

	splittedPath := splitPathString(path)

	var combinedPath = ""
	if len(api.Servers) > 0 {
		combinedPath = api.Servers[0].URL + splittedPath[1]
	} else {
		combinedPath = splittedPath[1]
	}

	request, err := http.NewRequest(splittedPath[0], combinedPath, strings.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Fatal(err)
	}

	route, params, err := router.FindRoute(request)
	if err != nil {
		log.Fatal(err)
	}

	requestValidationInput := &openapi3filter.RequestValidationInput{
		Request:    request,
		PathParams: params,
		Route:      route,
		Options:    &openapi3filter.Options{MultiError: true, AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
	}

	err = openapi3filter.ValidateRequest(a.ctx, requestValidationInput)

	if err != nil {
		return getErrors(err.(openapi3.MultiError))
	}

	return nil
}

func getErrors(me openapi3.MultiError) map[string][]string {
	issues := make(map[string][]string)
	for _, e := range me {
		const prefixBody = "@body"
		switch e := e.(type) {
		case *openapi3.SchemaError:
			field := prefixBody
			if path := e.JSONPointer(); len(path) > 0 {
				field = fmt.Sprintf("%s.%s", field, strings.Join(path, "."))
			}
			issues[field] = append(issues[field], e.Error())
			break
		case *openapi3filter.RequestError:
			if e.Parameter != nil {
				prefix := e.Parameter.In
				name := fmt.Sprintf("%s.%s", prefix, e.Parameter.Name)
				issues[name] = append(issues[name], e.Error())
				continue
			}
			// check if invalid HTTP parameter
			if e.Parameter != nil {
				prefix := e.Parameter.In
				name := fmt.Sprintf("%s.%s", prefix, e.Parameter.Name)
				issues[name] = append(issues[name], e.Error())
				continue
			}

			if err, ok := e.Err.(openapi3.MultiError); ok {
				for k, v := range getErrors(err) {
					issues[k] = append(issues[k], v...)
				}
				continue
			}
			// check if requestBody
			if e.RequestBody != nil {
				issues[prefixBody] = append(issues[prefixBody], e.Error())
				continue
			}
		default:
			break
		}
	}
	return issues
}

func (a *App) LoadPaths(spec string) []string {
	if spec == "" {
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Error",
			Message: "Missing input",
		})
		return nil
	}

	var apiSpecMap = map[string]interface{}{}

	if err := yaml.Unmarshal([]byte(spec), &apiSpecMap); err != nil {
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Error",
			Message: "Unable to parse definition",
		})
		return nil
	}

	if !isValidApiDefinition(apiSpecMap) {
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Error",
			Message: "No valid definition. Definition must contain 'swagger' or 'openApi'",
		})
		return nil
	}

	var api *openapi3.T

	if isOpenApiV3(apiSpecMap) {
		api = loadOpenApiDoc(spec)
	} else {
		api = loadSwaggerDoc(spec)
	}

	return loadApiPaths(api)
}

func loadApiPaths(api *openapi3.T) []string {
	var paths []string

	var sortedPaths []string
	for s, _ := range api.Paths {
		sortedPaths = append(sortedPaths, s)
	}
	sort.Strings(sortedPaths)

	for _, path := range sortedPaths {
		for _, op := range getOperations(api.Paths[path]) {
			paths = append(paths, op+" "+path)
		}
	}
	return paths
}

func loadOpenApiDoc(openApiSpec string) *openapi3.T {
	loader := openapi3.NewLoader()

	if openApiV3, err := loader.LoadFromData([]byte(openApiSpec)); err != nil {
		log.Fatal(err)
	} else {
		return openApiV3
	}
	return nil
}

func loadSwaggerDoc(swaggerSpec string) *openapi3.T {
	swaggerApi := openapi2.T{}

	if err := yaml.Unmarshal([]byte(swaggerSpec), &swaggerApi); err != nil {
		log.Fatal(err)
	}
	if openApiV3, err := openapi2conv.ToV3(&swaggerApi); err != nil {
		log.Fatal(err)
	} else {
		return openApiV3
	}
	return nil
}

func getOperations(pathItem *openapi3.PathItem) []string {
	var operations []string

	elem := reflect.ValueOf(pathItem).Elem()
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		name := elem.Type().Field(i).Name
		typ := field.Type()
		if typ == reflect.TypeOf(openapi3.NewOperation()) && !field.IsNil() {
			operations = append(operations, strings.ToUpper(name))
		}
	}

	return operations
}

func isOpenApiV3(m map[string]interface{}) bool {
	if m["openapi"] != nil {
		return true
	}
	return false
}

func isValidApiDefinition(m map[string]interface{}) bool {
	return m["openapi"] != nil || m["swagger"] != nil
}

func splitPathString(path string) []string {
	if splittedPath := strings.Split(path, " "); len(splittedPath) == 2 && splittedPath[0] != "" && splittedPath[1] != "" {
		return splittedPath
	} else {
		return nil
	}
}
