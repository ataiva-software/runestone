package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
	"gopkg.in/yaml.v3"
)

// Parser handles parsing and processing of Runestone configuration files
type Parser struct {
	variables map[string]interface{}
}

// NewParser creates a new configuration parser
func NewParser() *Parser {
	return &Parser{
		variables: make(map[string]interface{}),
	}
}

// ParseFile parses a Runestone configuration file
func (p *Parser) ParseFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return p.Parse(data)
}

// ParseFromString parses a Runestone configuration from a string
func (p *Parser) ParseFromString(configYAML string) (*Config, error) {
	return p.Parse([]byte(configYAML))
}

// Parse parses Runestone configuration from YAML data
func (p *Parser) Parse(data []byte) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Set up variables for expression evaluation
	p.variables = config.Variables
	if p.variables == nil {
		p.variables = make(map[string]interface{})
	}
	p.variables["environment"] = config.Environment
	p.variables["project"] = config.Project

	// Process expressions in the configuration
	if err := p.processExpressions(&config); err != nil {
		return nil, fmt.Errorf("failed to process expressions: %w", err)
	}

	return &config, nil
}

// processExpressions evaluates all expressions in the configuration
func (p *Parser) processExpressions(config *Config) error {
	// Process provider expressions directly (simpler approach)
	for name, provider := range config.Providers {
		// Process Region field
		if strings.Contains(provider.Region, "${") {
			if processed, err := p.evaluateExpression(provider.Region); err != nil {
				return fmt.Errorf("error processing provider %s region: %w", name, err)
			} else if processedStr, ok := processed.(string); ok {
				provider.Region = processedStr
			}
		}
		
		// Process Profile field
		if strings.Contains(provider.Profile, "${") {
			if processed, err := p.evaluateExpression(provider.Profile); err != nil {
				return fmt.Errorf("error processing provider %s profile: %w", name, err)
			} else if processedStr, ok := processed.(string); ok {
				provider.Profile = processedStr
			}
		}
		
		config.Providers[name] = provider
	}

	// Process module expressions using reflection
	for name, module := range config.Modules {
		if err := p.processValue(&module); err != nil {
			return fmt.Errorf("error processing module %s: %w", name, err)
		}
		config.Modules[name] = module
	}

	// Process resource expressions using reflection
	for i := range config.Resources {
		if err := p.processValue(&config.Resources[i]); err != nil {
			return fmt.Errorf("error processing resource %d: %w", i, err)
		}
	}

	return nil
}

// processValue recursively processes expressions in any value
func (p *Parser) processValue(v interface{}) error {
	visited := make(map[uintptr]bool)
	return p.processValueReflectWithVisited(reflect.ValueOf(v), visited)
}

// processValueReflect handles the actual reflection processing
func (p *Parser) processValueReflect(val reflect.Value) error {
	visited := make(map[uintptr]bool)
	return p.processValueReflectWithVisited(val, visited)
}

// processValueReflectWithVisited handles the actual reflection processing with cycle detection
func (p *Parser) processValueReflectWithVisited(val reflect.Value, visited map[uintptr]bool) error {
	if !val.IsValid() {
		return nil
	}

	// Handle pointers
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		// Check for cycles
		ptr := val.Pointer()
		if visited[ptr] {
			return nil // Already processed
		}
		visited[ptr] = true
		return p.processValueReflectWithVisited(val.Elem(), visited)
	}

	// Handle interfaces
	if val.Kind() == reflect.Interface {
		if val.IsNil() {
			return nil
		}
		return p.processValueReflectWithVisited(val.Elem(), visited)
	}

	switch val.Kind() {
	case reflect.String:
		if val.CanSet() {
			str := val.String()
			if processed, err := p.evaluateExpression(str); err != nil {
				return err
			} else if processed != str {
				// If the result is still a string, set it directly
				if processedStr, ok := processed.(string); ok {
					val.SetString(processedStr)
				} else {
					// If it's not a string, we need to replace the entire value
					// This happens when ${variable} evaluates to a non-string type
					if val.CanSet() {
						val.Set(reflect.ValueOf(processed))
					}
				}
			}
		}
	case reflect.Map:
		if val.IsNil() {
			return nil
		}
		// Check for cycles
		ptr := val.Pointer()
		if visited[ptr] {
			return nil // Already processed
		}
		visited[ptr] = true
		
		// For maps, we need to process each value and potentially update the map
		for _, key := range val.MapKeys() {
			mapVal := val.MapIndex(key)
			if mapVal.IsValid() && mapVal.CanInterface() {
				originalValue := mapVal.Interface()
				
				// Process strings for expressions
				if str, ok := originalValue.(string); ok {
					if processed, err := p.evaluateExpression(str); err != nil {
						return err
					} else if processed != str {
						// Set the processed value (could be any type)
						val.SetMapIndex(key, reflect.ValueOf(processed))
					}
				} else {
					// For non-string values, recursively process them
					newVal := reflect.New(mapVal.Type()).Elem()
					newVal.Set(mapVal)
					if err := p.processValueReflectWithVisited(newVal, visited); err != nil {
						return err
					}
					val.SetMapIndex(key, newVal)
				}
			}
		}
	case reflect.Slice:
		if val.IsNil() {
			return nil
		}
		// Check for cycles
		ptr := val.Pointer()
		if visited[ptr] {
			return nil // Already processed
		}
		visited[ptr] = true
		
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i)
			if elem.IsValid() && elem.CanSet() {
				if err := p.processValueReflectWithVisited(elem, visited); err != nil {
					return err
				}
			}
		}
	case reflect.Struct:
		// For structs, use the struct's address for cycle detection
		if val.CanAddr() {
			ptr := val.Addr().Pointer()
			if visited[ptr] {
				return nil // Already processed
			}
			visited[ptr] = true
		}
		
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			fieldType := val.Type().Field(i)
			
			// Skip unexported fields
			if !fieldType.IsExported() {
				continue
			}
			
			if field.IsValid() && field.CanSet() {
				if err := p.processValueReflectWithVisited(field, visited); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// evaluateExpression evaluates expressions in strings
func (p *Parser) evaluateExpression(input string) (interface{}, error) {
	if !strings.Contains(input, "${") {
		return input, nil
	}

	// Check if the entire string is a single expression
	if strings.HasPrefix(input, "${") && strings.HasSuffix(input, "}") && strings.Count(input, "${") == 1 {
		exprStr := input[2 : len(input)-1]
		return p.evaluateExpr(exprStr)
	}

	// Handle multiple expressions or mixed content
	result := input
	for {
		start := strings.Index(result, "${")
		if start == -1 {
			break
		}

		end := strings.Index(result[start:], "}")
		if end == -1 {
			return nil, fmt.Errorf("unclosed expression in: %s", input)
		}
		end += start

		exprStr := result[start+2 : end]
		value, err := p.evaluateExpr(exprStr)
		if err != nil {
			return nil, fmt.Errorf("error evaluating expression '%s': %w", exprStr, err)
		}

		result = result[:start] + fmt.Sprintf("%v", value) + result[end+1:]
	}

	return result, nil
}

// evaluateExpr evaluates a single expression
func (p *Parser) evaluateExpr(exprStr string) (interface{}, error) {
	// Handle simple variable references
	if val, exists := p.variables[exprStr]; exists {
		return val, nil
	}

	// For variables that might not exist yet (like 'index' during initial parsing),
	// return the expression as-is to be evaluated later during expansion
	if isSimpleVariable(exprStr) {
		return "${" + exprStr + "}", nil
	}

	program, err := expr.Compile(exprStr, expr.Env(p.variables))
	if err != nil {
		// If compilation fails due to unknown variables, return the expression as-is
		// This will be re-evaluated later during resource expansion
		return "${" + exprStr + "}", nil
	}

	result, err := expr.Run(program, p.variables)
	if err != nil {
		// If execution fails, return the expression as-is for later evaluation
		return "${" + exprStr + "}", nil
	}

	return result, nil
}

// isSimpleVariable checks if the expression is just a simple variable name
func isSimpleVariable(expr string) bool {
	// Simple heuristic: if it contains no operators or spaces, it's likely a variable
	return !strings.ContainsAny(expr, " +-*/()[]{}=<>!&|?:")
}

// ExpandResources expands resources with count and for_each into individual instances
func (p *Parser) ExpandResources(resources []Resource) ([]ResourceInstance, error) {
	var instances []ResourceInstance

	for _, resource := range resources {
		expanded, err := p.expandResource(resource)
		if err != nil {
			return nil, fmt.Errorf("error expanding resource %s: %w", resource.Name, err)
		}
		instances = append(instances, expanded...)
	}

	return instances, nil
}

// expandResource expands a single resource based on count or for_each
func (p *Parser) expandResource(resource Resource) ([]ResourceInstance, error) {
	var instances []ResourceInstance

	// Handle count
	if resource.Count != nil {
		count, err := p.resolveCount(resource.Count)
		if err != nil {
			return nil, fmt.Errorf("error resolving count: %w", err)
		}

		for i := 0; i < count; i++ {
			instance, err := p.createInstance(resource, map[string]interface{}{"index": i})
			if err != nil {
				return nil, err
			}
			instances = append(instances, instance)
		}
		return instances, nil
	}

	// Handle for_each
	if resource.ForEach != nil {
		items, err := p.resolveForEach(resource.ForEach)
		if err != nil {
			return nil, fmt.Errorf("error resolving for_each: %w", err)
		}

		for _, item := range items {
			vars := map[string]interface{}{}
			switch v := item.(type) {
			case string:
				vars["region"] = v // Common case for regions
			default:
				vars["item"] = v
			}

			instance, err := p.createInstance(resource, vars)
			if err != nil {
				return nil, err
			}
			instances = append(instances, instance)
		}
		return instances, nil
	}

	// Single instance
	instance, err := p.createInstance(resource, nil)
	if err != nil {
		return nil, err
	}
	instances = append(instances, instance)

	return instances, nil
}

// createInstance creates a resource instance with variable substitution
func (p *Parser) createInstance(resource Resource, vars map[string]interface{}) (ResourceInstance, error) {
	// Create a copy of variables with instance-specific vars
	instanceVars := make(map[string]interface{})
	for k, v := range p.variables {
		instanceVars[k] = v
	}
	for k, v := range vars {
		instanceVars[k] = v
	}

	// Create a temporary parser with instance variables
	tempParser := &Parser{variables: instanceVars}

	// Process the resource with instance variables
	resourceCopy := resource
	
	// Process Name field directly
	if strings.Contains(resourceCopy.Name, "${") {
		if processed, err := tempParser.evaluateExpression(resourceCopy.Name); err != nil {
			return ResourceInstance{}, fmt.Errorf("error processing resource name: %w", err)
		} else if processedStr, ok := processed.(string); ok {
			resourceCopy.Name = processedStr
		}
	}
	
	// Process other fields using reflection
	if err := tempParser.processValue(&resourceCopy); err != nil {
		return ResourceInstance{}, err
	}

	instance := ResourceInstance{
		ID:          fmt.Sprintf("%s.%s", resourceCopy.Kind, resourceCopy.Name),
		Kind:        resourceCopy.Kind,
		Name:        resourceCopy.Name,
		Properties:  resourceCopy.Properties,
		DriftPolicy: resourceCopy.DriftPolicy,
		DependsOn:   resourceCopy.DependsOn,
	}

	return instance, nil
}

// resolveCount resolves a count value (int or expression)
func (p *Parser) resolveCount(count interface{}) (int, error) {
	switch v := count.(type) {
	case int:
		return v, nil
	case string:
		result, err := p.evaluateExpr(v)
		if err != nil {
			return 0, err
		}
		if intVal, ok := result.(int); ok {
			return intVal, nil
		}
		if strVal, ok := result.(string); ok {
			return strconv.Atoi(strVal)
		}
		return 0, fmt.Errorf("count expression must evaluate to an integer")
	default:
		return 0, fmt.Errorf("count must be an integer or expression")
	}
}

// resolveForEach resolves a for_each value (array or expression)
func (p *Parser) resolveForEach(forEach interface{}) ([]interface{}, error) {
	switch v := forEach.(type) {
	case []interface{}:
		return v, nil
	case string:
		// Check if it's an expression
		if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
			exprStr := v[2 : len(v)-1]
			result, err := p.evaluateExpr(exprStr)
			if err != nil {
				return nil, err
			}
			if slice, ok := result.([]interface{}); ok {
				return slice, nil
			}
			return nil, fmt.Errorf("for_each expression must evaluate to an array")
		}
		// If it's not an expression, treat it as a single-item array
		return []interface{}{v}, nil
	default:
		return nil, fmt.Errorf("for_each must be an array or expression")
	}
}
