package executor

import (
	"testing"

	"github.com/ataiva-software/runestone/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDAG(t *testing.T) {
	tests := []struct {
		name      string
		instances []config.ResourceInstance
		wantErr   bool
	}{
		{
			name: "simple DAG without dependencies",
			instances: []config.ResourceInstance{
				{
					ID:   "aws:s3:bucket.test1",
					Kind: "aws:s3:bucket",
					Name: "test1",
				},
				{
					ID:   "aws:s3:bucket.test2",
					Kind: "aws:s3:bucket",
					Name: "test2",
				},
			},
			wantErr: false,
		},
		{
			name: "DAG with explicit dependencies",
			instances: []config.ResourceInstance{
				{
					ID:   "module:vpc.network",
					Kind: "module:vpc",
					Name: "network",
				},
				{
					ID:        "aws:ec2:instance.web",
					Kind:      "aws:ec2:instance",
					Name:      "web",
					DependsOn: []string{"module:vpc.network"},
				},
			},
			wantErr: false,
		},
		{
			name: "DAG with missing dependency",
			instances: []config.ResourceInstance{
				{
					ID:        "aws:ec2:instance.web",
					Kind:      "aws:ec2:instance",
					Name:      "web",
					DependsOn: []string{"module:vpc.network"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dag, err := NewDAG(tt.instances)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, dag)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, dag)
			assert.Equal(t, len(tt.instances), len(dag.nodes))

			// Verify all instances are present
			for _, instance := range tt.instances {
				node, exists := dag.GetNode(instance.ID)
				assert.True(t, exists)
				assert.Equal(t, instance.ID, node.ID)
				assert.Equal(t, StatusPending, node.Status)
			}
		})
	}
}

func TestDAG_GetExecutionOrder(t *testing.T) {
	tests := []struct {
		name      string
		instances []config.ResourceInstance
		expected  [][]string
	}{
		{
			name: "independent resources",
			instances: []config.ResourceInstance{
				{ID: "aws:s3:bucket.test1", Kind: "aws:s3:bucket", Name: "test1"},
				{ID: "aws:s3:bucket.test2", Kind: "aws:s3:bucket", Name: "test2"},
			},
			expected: [][]string{
				{"aws:s3:bucket.test1", "aws:s3:bucket.test2"},
			},
		},
		{
			name: "linear dependency chain",
			instances: []config.ResourceInstance{
				{ID: "module:vpc.network", Kind: "module:vpc", Name: "network"},
				{
					ID:        "aws:ec2:instance.web",
					Kind:      "aws:ec2:instance",
					Name:      "web",
					DependsOn: []string{"module:vpc.network"},
				},
				{
					ID:        "aws:elb:loadbalancer.lb",
					Kind:      "aws:elb:loadbalancer",
					Name:      "lb",
					DependsOn: []string{"aws:ec2:instance.web"},
				},
			},
			expected: [][]string{
				{"module:vpc.network"},
				{"aws:ec2:instance.web"},
				{"aws:elb:loadbalancer.lb"},
			},
		},
		{
			name: "diamond dependency",
			instances: []config.ResourceInstance{
				{ID: "module:vpc.network", Kind: "module:vpc", Name: "network"},
				{
					ID:        "aws:ec2:instance.web1",
					Kind:      "aws:ec2:instance",
					Name:      "web1",
					DependsOn: []string{"module:vpc.network"},
				},
				{
					ID:        "aws:ec2:instance.web2",
					Kind:      "aws:ec2:instance",
					Name:      "web2",
					DependsOn: []string{"module:vpc.network"},
				},
				{
					ID:        "aws:elb:loadbalancer.lb",
					Kind:      "aws:elb:loadbalancer",
					Name:      "lb",
					DependsOn: []string{"aws:ec2:instance.web1", "aws:ec2:instance.web2"},
				},
			},
			expected: [][]string{
				{"module:vpc.network"},
				{"aws:ec2:instance.web1", "aws:ec2:instance.web2"},
				{"aws:elb:loadbalancer.lb"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dag, err := NewDAG(tt.instances)
			require.NoError(t, err)

			order := dag.GetExecutionOrder()
			assert.Equal(t, len(tt.expected), len(order))

			for i, expectedLevel := range tt.expected {
				assert.ElementsMatch(t, expectedLevel, order[i])
			}
		})
	}
}

func TestDAG_GetReadyNodes(t *testing.T) {
	instances := []config.ResourceInstance{
		{ID: "module:vpc.network", Kind: "module:vpc", Name: "network"},
		{
			ID:        "aws:ec2:instance.web",
			Kind:      "aws:ec2:instance",
			Name:      "web",
			DependsOn: []string{"module:vpc.network"},
		},
	}

	dag, err := NewDAG(instances)
	require.NoError(t, err)

	// Initially, only the module should be ready
	ready := dag.GetReadyNodes()
	assert.Len(t, ready, 1)
	assert.Equal(t, "module:vpc.network", ready[0].ID)

	// Mark the module as completed
	dag.SetNodeStatus("module:vpc.network", StatusCompleted, nil)

	// Now the EC2 instance should be ready
	ready = dag.GetReadyNodes()
	assert.Len(t, ready, 1)
	assert.Equal(t, "aws:ec2:instance.web", ready[0].ID)

	// Mark the EC2 instance as completed
	dag.SetNodeStatus("aws:ec2:instance.web", StatusCompleted, nil)

	// No nodes should be ready now
	ready = dag.GetReadyNodes()
	assert.Len(t, ready, 0)
}

func TestDAG_SetNodeStatus(t *testing.T) {
	instances := []config.ResourceInstance{
		{ID: "aws:s3:bucket.test", Kind: "aws:s3:bucket", Name: "test"},
	}

	dag, err := NewDAG(instances)
	require.NoError(t, err)

	// Initial status should be pending
	node, exists := dag.GetNode("aws:s3:bucket.test")
	assert.True(t, exists)
	assert.Equal(t, StatusPending, node.Status)
	assert.Nil(t, node.Error)

	// Update status to running
	dag.SetNodeStatus("aws:s3:bucket.test", StatusRunning, nil)
	node, exists = dag.GetNode("aws:s3:bucket.test")
	assert.True(t, exists)
	assert.Equal(t, StatusRunning, node.Status)
	assert.Nil(t, node.Error)

	// Update status to failed with error
	testErr := assert.AnError
	dag.SetNodeStatus("aws:s3:bucket.test", StatusFailed, testErr)
	node, exists = dag.GetNode("aws:s3:bucket.test")
	assert.True(t, exists)
	assert.Equal(t, StatusFailed, node.Status)
	assert.Equal(t, testErr, node.Error)
}

func TestDAG_IsComplete(t *testing.T) {
	instances := []config.ResourceInstance{
		{ID: "aws:s3:bucket.test1", Kind: "aws:s3:bucket", Name: "test1"},
		{ID: "aws:s3:bucket.test2", Kind: "aws:s3:bucket", Name: "test2"},
	}

	dag, err := NewDAG(instances)
	require.NoError(t, err)

	// Initially not complete
	assert.False(t, dag.IsComplete())

	// Mark one as completed
	dag.SetNodeStatus("aws:s3:bucket.test1", StatusCompleted, nil)
	assert.False(t, dag.IsComplete())

	// Mark the other as completed
	dag.SetNodeStatus("aws:s3:bucket.test2", StatusCompleted, nil)
	assert.True(t, dag.IsComplete())
}

func TestDAG_HasFailures(t *testing.T) {
	instances := []config.ResourceInstance{
		{ID: "aws:s3:bucket.test1", Kind: "aws:s3:bucket", Name: "test1"},
		{ID: "aws:s3:bucket.test2", Kind: "aws:s3:bucket", Name: "test2"},
	}

	dag, err := NewDAG(instances)
	require.NoError(t, err)

	// Initially no failures
	assert.False(t, dag.HasFailures())

	// Mark one as completed
	dag.SetNodeStatus("aws:s3:bucket.test1", StatusCompleted, nil)
	assert.False(t, dag.HasFailures())

	// Mark the other as failed
	dag.SetNodeStatus("aws:s3:bucket.test2", StatusFailed, assert.AnError)
	assert.True(t, dag.HasFailures())

	// Get failed nodes
	failed := dag.GetFailedNodes()
	assert.Len(t, failed, 1)
	assert.Equal(t, "aws:s3:bucket.test2", failed[0].ID)
}

func TestDAG_validateAcyclic(t *testing.T) {
	tests := []struct {
		name      string
		instances []config.ResourceInstance
		wantErr   bool
	}{
		{
			name: "acyclic graph",
			instances: []config.ResourceInstance{
				{ID: "a", Kind: "test", Name: "a"},
				{ID: "b", Kind: "test", Name: "b", DependsOn: []string{"a"}},
				{ID: "c", Kind: "test", Name: "c", DependsOn: []string{"b"}},
			},
			wantErr: false,
		},
		{
			name: "self-referencing cycle",
			instances: []config.ResourceInstance{
				{ID: "a", Kind: "test", Name: "a", DependsOn: []string{"a"}},
			},
			wantErr: true,
		},
		{
			name: "two-node cycle",
			instances: []config.ResourceInstance{
				{ID: "a", Kind: "test", Name: "a", DependsOn: []string{"b"}},
				{ID: "b", Kind: "test", Name: "b", DependsOn: []string{"a"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDAG(tt.instances)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "circular dependency")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
