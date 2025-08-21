package executor

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/ataiva-software/runestone/internal/config"
)

// DAGNode represents a node in the dependency graph
type DAGNode struct {
	ID           string
	Instance     config.ResourceInstance
	Dependencies []string
	Dependents   []string
	Status       NodeStatus
	Error        error
}

// NodeStatus represents the execution status of a node
type NodeStatus string

const (
	StatusPending   NodeStatus = "pending"
	StatusReady     NodeStatus = "ready"
	StatusRunning   NodeStatus = "running"
	StatusCompleted NodeStatus = "completed"
	StatusFailed    NodeStatus = "failed"
)

// DAG represents a directed acyclic graph of resources
type DAG struct {
	nodes map[string]*DAGNode
	mutex sync.RWMutex
}

// NewDAG creates a new DAG from resource instances
func NewDAG(instances []config.ResourceInstance) (*DAG, error) {
	dag := &DAG{
		nodes: make(map[string]*DAGNode),
	}

	// Create nodes
	for _, instance := range instances {
		node := &DAGNode{
			ID:           instance.ID,
			Instance:     instance,
			Dependencies: make([]string, 0),
			Dependents:   make([]string, 0),
			Status:       StatusPending,
		}
		dag.nodes[instance.ID] = node
	}

	// Build dependency relationships
	for _, node := range dag.nodes {
		for _, depID := range node.Instance.DependsOn {
			if depNode, exists := dag.nodes[depID]; exists {
				node.Dependencies = append(node.Dependencies, depID)
				depNode.Dependents = append(depNode.Dependents, node.ID)
			} else {
				return nil, fmt.Errorf("dependency %s not found for resource %s", depID, node.ID)
			}
		}

		// Infer dependencies from resource kinds (e.g., modules before resources)
		if err := dag.inferDependencies(node); err != nil {
			return nil, err
		}
	}

	// Validate that the graph is acyclic
	if err := dag.validateAcyclic(); err != nil {
		return nil, err
	}

	return dag, nil
}

// inferDependencies infers implicit dependencies between resources
func (d *DAG) inferDependencies(node *DAGNode) error {
	// Module resources should be created before other resources that might depend on them
	if strings.HasPrefix(node.Instance.Kind, "module:") {
		return nil // Modules typically don't depend on other resources
	}

	// Look for potential dependencies based on naming patterns or resource types
	for _, otherNode := range d.nodes {
		if otherNode.ID == node.ID {
			continue
		}

		// If this is a module, other resources might depend on it
		if strings.HasPrefix(otherNode.Instance.Kind, "module:") {
			// Check if the resource name suggests it depends on the module
			if d.shouldDependOn(node, otherNode) {
				node.Dependencies = append(node.Dependencies, otherNode.ID)
				otherNode.Dependents = append(otherNode.Dependents, node.ID)
			}
		}
	}

	return nil
}

// shouldDependOn determines if one resource should depend on another
func (d *DAG) shouldDependOn(dependent, dependency *DAGNode) bool {
	// Simple heuristic: if the dependent resource name contains the dependency name
	// This is a basic implementation - in practice, you'd want more sophisticated logic
	if strings.HasPrefix(dependency.Instance.Kind, "module:") {
		// Check if the dependent resource might use outputs from the module
		return strings.Contains(dependent.Instance.Name, strings.TrimPrefix(dependency.Instance.Kind, "module:"))
	}
	return false
}

// validateAcyclic checks if the graph contains cycles
func (d *DAG) validateAcyclic() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for nodeID := range d.nodes {
		if !visited[nodeID] {
			if d.hasCycle(nodeID, visited, recStack) {
				return fmt.Errorf("circular dependency detected involving resource %s", nodeID)
			}
		}
	}

	return nil
}

// hasCycle performs DFS to detect cycles
func (d *DAG) hasCycle(nodeID string, visited, recStack map[string]bool) bool {
	visited[nodeID] = true
	recStack[nodeID] = true

	node := d.nodes[nodeID]
	for _, depID := range node.Dependencies {
		if !visited[depID] {
			if d.hasCycle(depID, visited, recStack) {
				return true
			}
		} else if recStack[depID] {
			return true
		}
	}

	recStack[nodeID] = false
	return false
}

// GetExecutionOrder returns the topological order for execution
func (d *DAG) GetExecutionOrder() [][]string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	var levels [][]string
	processed := make(map[string]bool)
	
	for len(processed) < len(d.nodes) {
		var currentLevel []string
		
		// Find nodes with no unprocessed dependencies
		for nodeID, node := range d.nodes {
			if processed[nodeID] {
				continue
			}
			
			canExecute := true
			for _, depID := range node.Dependencies {
				if !processed[depID] {
					canExecute = false
					break
				}
			}
			
			if canExecute {
				currentLevel = append(currentLevel, nodeID)
			}
		}
		
		if len(currentLevel) == 0 {
			// This shouldn't happen if the graph is acyclic
			break
		}
		
		// Sort for deterministic output
		sort.Strings(currentLevel)
		levels = append(levels, currentLevel)
		
		// Mark current level as processed
		for _, nodeID := range currentLevel {
			processed[nodeID] = true
		}
	}
	
	return levels
}

// GetReadyNodes returns nodes that are ready to execute
func (d *DAG) GetReadyNodes() []*DAGNode {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	var ready []*DAGNode
	for _, node := range d.nodes {
		if node.Status == StatusPending {
			canExecute := true
			for _, depID := range node.Dependencies {
				depNode := d.nodes[depID]
				if depNode.Status != StatusCompleted {
					canExecute = false
					break
				}
			}
			if canExecute {
				ready = append(ready, node)
			}
		}
	}

	return ready
}

// SetNodeStatus updates the status of a node
func (d *DAG) SetNodeStatus(nodeID string, status NodeStatus, err error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if node, exists := d.nodes[nodeID]; exists {
		node.Status = status
		node.Error = err
	}
}

// GetNode returns a node by ID
func (d *DAG) GetNode(nodeID string) (*DAGNode, bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	node, exists := d.nodes[nodeID]
	return node, exists
}

// GetAllNodes returns all nodes in the DAG
func (d *DAG) GetAllNodes() map[string]*DAGNode {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	result := make(map[string]*DAGNode)
	for id, node := range d.nodes {
		result[id] = node
	}
	return result
}

// IsComplete returns true if all nodes have completed (successfully or with error)
func (d *DAG) IsComplete() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for _, node := range d.nodes {
		if node.Status != StatusCompleted && node.Status != StatusFailed {
			return false
		}
	}
	return true
}

// HasFailures returns true if any node has failed
func (d *DAG) HasFailures() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for _, node := range d.nodes {
		if node.Status == StatusFailed {
			return true
		}
	}
	return false
}

// GetFailedNodes returns all nodes that have failed
func (d *DAG) GetFailedNodes() []*DAGNode {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	var failed []*DAGNode
	for _, node := range d.nodes {
		if node.Status == StatusFailed {
			failed = append(failed, node)
		}
	}
	return failed
}
