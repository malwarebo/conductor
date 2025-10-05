package utils

import (
	"context"
	"sync"
	"time"
)

type HealthStatus int

const (
	StatusHealthy HealthStatus = iota
	StatusUnhealthy
	StatusUnknown
)

type HealthChecker struct {
	checkFunc func(ctx context.Context) error
	interval  time.Duration
	timeout   time.Duration
	status    HealthStatus
	lastCheck time.Time
	mutex     sync.RWMutex
	stopChan  chan struct{}
}

func CreateHealthChecker(checkFunc func(ctx context.Context) error, interval, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		checkFunc: checkFunc,
		interval:  interval,
		timeout:   timeout,
		status:    StatusUnknown,
		stopChan:  make(chan struct{}),
	}
}

func (hc *HealthChecker) Start() {
	go hc.run()
}

func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
}

func (hc *HealthChecker) run() {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.performCheck()
		case <-hc.stopChan:
			return
		}
	}
}

func (hc *HealthChecker) performCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	err := hc.checkFunc(ctx)

	hc.mutex.Lock()
	hc.lastCheck = time.Now()
	if err != nil {
		hc.status = StatusUnhealthy
	} else {
		hc.status = StatusHealthy
	}
	hc.mutex.Unlock()
}

func (hc *HealthChecker) GetStatus() HealthStatus {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()
	return hc.status
}

func (hc *HealthChecker) IsHealthy() bool {
	return hc.GetStatus() == StatusHealthy
}

func (hc *HealthChecker) GetLastCheck() time.Time {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()
	return hc.lastCheck
}
