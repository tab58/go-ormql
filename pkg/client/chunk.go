package client

import (
	"context"
	"fmt"

	"github.com/tab58/go-ormql/pkg/cypher"
	"github.com/tab58/go-ormql/pkg/driver"
	"github.com/tab58/go-ormql/pkg/translate"
	"github.com/vektah/gqlparser/v2/ast"
)

// buildChunkPlan creates a new TranslationPlan with the given chunk params
// applied to both WriteStatements and ReadStatement (preserving original queries).
func buildChunkPlan(plan translate.TranslationPlan, params map[string]any) translate.TranslationPlan {
	cp := translate.TranslationPlan{
		ReadStatement:   cypher.Statement{Query: plan.ReadStatement.Query, Params: params},
		WriteStatements: make([]cypher.Statement, len(plan.WriteStatements)),
	}
	for i, ws := range plan.WriteStatements {
		cp.WriteStatements[i] = cypher.Statement{Query: ws.Query, Params: params}
	}
	return cp
}

// chunkParams inspects the params map for []any values exceeding batchSize.
// Returns a slice of param maps — one per chunk.
// If no list exceeds batchSize, returns a single-element slice (the original params).
func chunkParams(params map[string]any, batchSize int) []map[string]any {
	if params == nil {
		return []map[string]any{params}
	}

	// Find the longest []any value
	maxLen := 0
	listKeys := map[string][]any{}
	for k, v := range params {
		if list, ok := v.([]any); ok {
			listKeys[k] = list
			if len(list) > maxLen {
				maxLen = len(list)
			}
		}
	}

	// No list exceeds batchSize — return single chunk
	if maxLen <= batchSize {
		return []map[string]any{params}
	}

	// Calculate number of chunks
	numChunks := (maxLen + batchSize - 1) / batchSize

	chunks := make([]map[string]any, numChunks)
	for i := range numChunks {
		chunk := make(map[string]any, len(params))
		start := i * batchSize
		end := start + batchSize

		// Copy non-list params unchanged
		for k, v := range params {
			if _, isList := listKeys[k]; !isList {
				chunk[k] = v
			}
		}

		// Slice each list param for this chunk
		for k, list := range listKeys {
			if start >= len(list) {
				chunk[k] = []any{}
			} else {
				hi := end
				if hi > len(list) {
					hi = len(list)
				}
				chunk[k] = list[start:hi]
			}
		}

		chunks[i] = chunk
	}

	return chunks
}

// aggregateResults merges chunk result data maps into one.
// Concatenates []any, sums int64/float64, recurses map[string]any, last-wins otherwise.
func aggregateResults(results []map[string]any) map[string]any {
	if len(results) == 0 {
		return nil
	}
	if len(results) == 1 {
		return results[0]
	}

	agg := make(map[string]any)
	for _, r := range results {
		for k, v := range r {
			existing, exists := agg[k]
			if !exists {
				agg[k] = v
				continue
			}
			agg[k] = mergeValue(existing, v)
		}
	}
	return agg
}

// mergeValue combines two values according to aggregation rules:
// []any: concatenate, int64/float64: sum, map[string]any: recurse, other: last-wins.
func mergeValue(existing, incoming any) any {
	switch ev := existing.(type) {
	case []any:
		if iv, ok := incoming.([]any); ok {
			return append(ev, iv...)
		}
		return incoming
	case int64:
		if iv, ok := incoming.(int64); ok {
			return ev + iv
		}
		return incoming
	case float64:
		if iv, ok := incoming.(float64); ok {
			return ev + iv
		}
		return incoming
	case map[string]any:
		iv, ok := incoming.(map[string]any)
		if !ok {
			return incoming
		}
		merged := make(map[string]any, len(ev))
		for k, v := range ev {
			merged[k] = v
		}
		for k, v := range iv {
			if prev, exists := merged[k]; exists {
				merged[k] = mergeValue(prev, v)
			} else {
				merged[k] = v
			}
		}
		return merged
	default:
		return incoming
	}
}

// executeChunk runs a single TranslationPlan against the driver.
// Handles WriteStatements (FOREACH writes) then ReadStatement.
func (c *Client) executeChunk(ctx context.Context, plan translate.TranslationPlan, op *ast.OperationDefinition) (*Result, error) {
	// Execute WriteStatements (FOREACH writes for merge mutations) before ReadStatement
	for _, ws := range plan.WriteStatements {
		if _, err := c.drv.ExecuteWrite(ctx, ws); err != nil {
			return nil, fmt.Errorf("write statement failed: %w", err)
		}
	}

	// Execute ReadStatement — queries use Execute, mutations use ExecuteWrite
	var drvResult driver.Result
	var err error
	switch op.Operation {
	case ast.Mutation:
		drvResult, err = c.drv.ExecuteWrite(ctx, plan.ReadStatement)
	default:
		drvResult, err = c.drv.Execute(ctx, plan.ReadStatement)
	}
	if err != nil {
		return nil, err
	}

	return &Result{data: extractResultData(drvResult)}, nil
}

// extractResultData extracts the response map from the first driver record's "data" column.
// Returns nil if no records, no "data" column, or the value is not a map.
func extractResultData(result driver.Result) map[string]any {
	if len(result.Records) == 0 {
		return nil
	}
	d, ok := result.Records[0].Values[resultDataKey]
	if !ok {
		return nil
	}
	m, ok := d.(map[string]any)
	if !ok {
		return nil
	}
	return m
}
