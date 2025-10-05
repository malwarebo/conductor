package observability

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type TraceSpan struct {
	ID        string
	ParentID  string
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Tags      map[string]string
	Logs      []TraceLog
}

type TraceLog struct {
	Timestamp time.Time
	Message   string
	Fields    map[string]interface{}
}

type Tracer struct {
	spans map[string]*TraceSpan
	mu    sync.RWMutex
}

func CreateTracer() *Tracer {
	return &Tracer{
		spans: make(map[string]*TraceSpan),
	}
}

func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, string) {
	spanID := generateSpanID()
	parentID := getParentSpanID(ctx)

	span := &TraceSpan{
		ID:        spanID,
		ParentID:  parentID,
		Name:      name,
		StartTime: time.Now(),
		Tags:      make(map[string]string),
		Logs:      make([]TraceLog, 0),
	}

	t.mu.Lock()
	t.spans[spanID] = span
	t.mu.Unlock()

	newCtx := context.WithValue(ctx, "span_id", spanID)
	return newCtx, spanID
}

func (t *Tracer) FinishSpan(spanID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if span, exists := t.spans[spanID]; exists {
		span.EndTime = time.Now()
		span.Duration = span.EndTime.Sub(span.StartTime)
	}
}

func (t *Tracer) AddTag(spanID string, key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if span, exists := t.spans[spanID]; exists {
		span.Tags[key] = value
	}
}

func (t *Tracer) AddLog(spanID string, message string, fields map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if span, exists := t.spans[spanID]; exists {
		log := TraceLog{
			Timestamp: time.Now(),
			Message:   message,
			Fields:    fields,
		}
		span.Logs = append(span.Logs, log)
	}
}

func (t *Tracer) GetSpan(spanID string) *TraceSpan {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.spans[spanID]
}

func (t *Tracer) GetSpans() []*TraceSpan {
	t.mu.RLock()
	defer t.mu.RUnlock()

	spans := make([]*TraceSpan, 0, len(t.spans))
	for _, span := range t.spans {
		spans = append(spans, span)
	}
	return spans
}

func (t *Tracer) GetTraceSummary() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.spans) == 0 {
		return map[string]interface{}{}
	}

	totalSpans := len(t.spans)
	totalDuration := time.Duration(0)
	avgDuration := time.Duration(0)

	for _, span := range t.spans {
		totalDuration += span.Duration
	}

	if totalSpans > 0 {
		avgDuration = totalDuration / time.Duration(totalSpans)
	}

	return map[string]interface{}{
		"total_spans":    totalSpans,
		"total_duration": totalDuration.String(),
		"avg_duration":   avgDuration.String(),
	}
}

func generateSpanID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func getParentSpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value("span_id").(string); ok {
		return spanID
	}
	return ""
}
