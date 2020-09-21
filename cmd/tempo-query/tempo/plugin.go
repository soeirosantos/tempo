package tempo

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"

	jaeger "github.com/jaegertracing/jaeger/model"
	jaeger_spanstore "github.com/jaegertracing/jaeger/storage/spanstore"

	ot_pdata "go.opentelemetry.io/collector/consumer/pdata"
	ot_jaeger "go.opentelemetry.io/collector/translator/trace/jaeger"
)

type Backend struct {
	tempoEndpoint string
}

func New(cfg *Config) *Backend {
	return &Backend{
		tempoEndpoint: "http://" + cfg.Backend + "/api/traces/",
	}
}

func (b *Backend) GetDependencies(endTs time.Time, lookback time.Duration) ([]jaeger.DependencyLink, error) {
	return nil, nil
}
func (b *Backend) GetTrace(ctx context.Context, traceID jaeger.TraceID) (*jaeger.Trace, error) {

	hexID := fmt.Sprintf("%016x%016x", traceID.High, traceID.Low)

	resp, err := http.Get(b.tempoEndpoint + hexID)
	if err != nil {
		return nil, fmt.Errorf("failed get to tempo %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, jaeger_spanstore.ErrTraceNotFound
	}

	out := &tempopb.Trace{}
	unmarshaller := &jsonpb.Unmarshaler{}
	err = unmarshaller.Unmarshal(resp.Body, out)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal trace json %w", err)
	}
	resp.Body.Close()

	otTrace := ot_pdata.TracesFromOtlp(out.Batches)
	jaegerBatches, err := ot_jaeger.InternalTracesToJaegerProto(otTrace)

	if err != nil {
		return nil, fmt.Errorf("error translating to jaegerBatches %v: %w", hexID, err)
	}

	jaegerTrace := &jaeger.Trace{
		Spans:      []*jaeger.Span{},
		ProcessMap: []jaeger.Trace_ProcessMapping{},
	}

	for _, batch := range jaegerBatches {
		// otel proto conversion doesn't set jaeger spans for some reason.
		for _, s := range batch.Spans {
			s.Process = batch.Process
		}

		jaegerTrace.Spans = append(jaegerTrace.Spans, batch.Spans...)
		jaegerTrace.ProcessMap = append(jaegerTrace.ProcessMap, jaeger.Trace_ProcessMapping{
			Process:   *batch.Process,
			ProcessID: batch.Process.ServiceName,
		})
	}

	return jaegerTrace, nil
}

func (b *Backend) GetServices(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (b *Backend) GetOperations(ctx context.Context, query jaeger_spanstore.OperationQueryParameters) ([]jaeger_spanstore.Operation, error) {
	return nil, nil
}
func (b *Backend) FindTraces(ctx context.Context, query *jaeger_spanstore.TraceQueryParameters) ([]*jaeger.Trace, error) {
	return nil, nil
}
func (b *Backend) FindTraceIDs(ctx context.Context, query *jaeger_spanstore.TraceQueryParameters) ([]jaeger.TraceID, error) {
	return nil, nil
}
func (b *Backend) WriteSpan(span *jaeger.Span) error {
	return nil
}