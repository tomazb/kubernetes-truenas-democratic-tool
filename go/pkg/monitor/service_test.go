package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockK8sClient is a mock implementation of k8s.Client
type MockK8sClient struct {
	mock.Mock
}

func (m *MockK8sClient) ListPersistentVolumes(ctx context.Context) ([]interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockK8sClient) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]interface{}, error) {
	args := m.Called(ctx, namespace)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockK8sClient) ListVolumeSnapshots(ctx context.Context, namespace string) ([]interface{}, error) {
	args := m.Called(ctx, namespace)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockK8sClient) TestConnection(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockTruenasClient is a mock implementation of truenas.Client
type MockTruenasClient struct {
	mock.Mock
}

func (m *MockTruenasClient) ListVolumes(ctx context.Context) ([]interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockTruenasClient) ListSnapshots(ctx context.Context) ([]interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockTruenasClient) TestConnection(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockMetricsExporter is a mock implementation of metrics.Exporter
type MockMetricsExporter struct {
	mock.Mock
}

func (m *MockMetricsExporter) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMetricsExporter) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMetricsExporter) SetOrphanedPVsCount(count float64) {
	m.Called(count)
}

func (m *MockMetricsExporter) SetOrphanedPVCsCount(count float64) {
	m.Called(count)
}

func (m *MockMetricsExporter) SetOrphanedSnapshotsCount(count float64) {
	m.Called(count)
}

func (m *MockMetricsExporter) SetScanDuration(duration float64) {
	m.Called(duration)
}

func TestNewService(t *testing.T) {
	// Setup
	mockK8s := &MockK8sClient{}
	mockTruenas := &MockTruenasClient{}
	mockMetrics := &MockMetricsExporter{}
	logger := zap.NewNop()

	config := Config{
		K8sClient:       mockK8s,
		TruenasClient:   mockTruenas,
		MetricsExporter: mockMetrics,
		Logger:          logger,
		ScanInterval:    5 * time.Minute,
	}

	// Execute
	service := NewService(config)

	// Verify
	assert.NotNil(t, service)
	assert.Equal(t, mockK8s, service.k8sClient)
	assert.Equal(t, mockTruenas, service.truenasClient)
	assert.Equal(t, mockMetrics, service.metricsExporter)
	assert.Equal(t, logger, service.logger)
	assert.Equal(t, 5*time.Minute, service.scanInterval)
	assert.False(t, service.running)
}

func TestServiceStartStop(t *testing.T) {
	// Setup
	mockK8s := &MockK8sClient{}
	mockTruenas := &MockTruenasClient{}
	mockMetrics := &MockMetricsExporter{}
	logger := zap.NewNop()

	config := Config{
		K8sClient:       mockK8s,
		TruenasClient:   mockTruenas,
		MetricsExporter: mockMetrics,
		Logger:          logger,
		ScanInterval:    100 * time.Millisecond, // Short interval for testing
	}

	service := NewService(config)

	// Setup mock expectations
	mockMetrics.On("Start").Return(nil)
	mockMetrics.On("Stop").Return(nil)
	mockK8s.On("ListPersistentVolumes", mock.Anything).Return([]interface{}{}, nil)
	mockK8s.On("ListPersistentVolumeClaims", mock.Anything, "").Return([]interface{}{}, nil)
	mockK8s.On("ListVolumeSnapshots", mock.Anything, "").Return([]interface{}{}, nil)
	mockTruenas.On("ListVolumes", mock.Anything).Return([]interface{}{}, nil)
	mockTruenas.On("ListSnapshots", mock.Anything).Return([]interface{}{}, nil)
	mockMetrics.On("SetOrphanedPVsCount", mock.Anything).Return()
	mockMetrics.On("SetOrphanedPVCsCount", mock.Anything).Return()
	mockMetrics.On("SetOrphanedSnapshotsCount", mock.Anything).Return()
	mockMetrics.On("SetScanDuration", mock.Anything).Return()

	ctx := context.Background()

	// Test Start
	err := service.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, service.running)

	// Wait a bit for the monitoring loop to run
	time.Sleep(200 * time.Millisecond)

	// Test Stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = service.Stop(stopCtx)
	assert.NoError(t, err)
	assert.False(t, service.running)

	// Verify mock expectations
	mockMetrics.AssertExpectations(t)
	mockK8s.AssertExpectations(t)
	mockTruenas.AssertExpectations(t)
}

func TestServiceDoubleStart(t *testing.T) {
	// Setup
	mockK8s := &MockK8sClient{}
	mockTruenas := &MockTruenasClient{}
	mockMetrics := &MockMetricsExporter{}
	logger := zap.NewNop()

	config := Config{
		K8sClient:       mockK8s,
		TruenasClient:   mockTruenas,
		MetricsExporter: mockMetrics,
		Logger:          logger,
		ScanInterval:    5 * time.Minute,
	}

	service := NewService(config)

	// Setup mock expectations
	mockMetrics.On("Start").Return(nil)

	ctx := context.Background()

	// First start should succeed
	err := service.Start(ctx)
	assert.NoError(t, err)

	// Second start should fail
	err = service.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Cleanup
	mockMetrics.On("Stop").Return(nil)
	service.Stop(context.Background())
}

func TestPerformScan(t *testing.T) {
	// Setup
	mockK8s := &MockK8sClient{}
	mockTruenas := &MockTruenasClient{}
	mockMetrics := &MockMetricsExporter{}
	logger := zap.NewNop()

	config := Config{
		K8sClient:       mockK8s,
		TruenasClient:   mockTruenas,
		MetricsExporter: mockMetrics,
		Logger:          logger,
		ScanInterval:    5 * time.Minute,
	}

	service := NewService(config)

	// Setup mock expectations
	mockK8s.On("ListPersistentVolumes", mock.Anything).Return([]interface{}{
		map[string]interface{}{
			"metadata": map[string]interface{}{"name": "pv-1"},
		},
	}, nil)
	mockK8s.On("ListPersistentVolumeClaims", mock.Anything, "").Return([]interface{}{
		map[string]interface{}{
			"metadata": map[string]interface{}{"name": "pvc-1"},
		},
	}, nil)
	mockK8s.On("ListVolumeSnapshots", mock.Anything, "").Return([]interface{}{}, nil)
	mockTruenas.On("ListVolumes", mock.Anything).Return([]interface{}{
		map[string]interface{}{"name": "volume-1"},
	}, nil)
	mockTruenas.On("ListSnapshots", mock.Anything).Return([]interface{}{}, nil)
	mockMetrics.On("SetOrphanedPVsCount", mock.Anything).Return()
	mockMetrics.On("SetOrphanedPVCsCount", mock.Anything).Return()
	mockMetrics.On("SetOrphanedSnapshotsCount", mock.Anything).Return()
	mockMetrics.On("SetScanDuration", mock.Anything).Return()

	// Execute
	ctx := context.Background()
	service.performScan(ctx)

	// Verify mock expectations
	mockK8s.AssertExpectations(t)
	mockTruenas.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}