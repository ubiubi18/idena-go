package log

import "testing"

func TestNewContextPreservesPrefixAndNormalizesSuffix(t *testing.T) {
	got := newContext([]interface{}{"component", "node"}, []interface{}{"height"})
	want := []interface{}{
		"component", "node",
		"height", nil,
		errorKey, "Normalized odd number of arguments by adding nil",
	}

	if len(got) != len(want) {
		t.Fatalf("context length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("context[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestCtxToArrayContainsEveryPair(t *testing.T) {
	ctx := Ctx{"component": "node", "height": uint64(10)}
	got := ctx.toArray()
	if len(got) != len(ctx)*2 {
		t.Fatalf("context length = %d, want %d", len(got), len(ctx)*2)
	}

	pairs := make(map[string]interface{}, len(ctx))
	for i := 0; i < len(got); i += 2 {
		key, ok := got[i].(string)
		if !ok {
			t.Fatalf("context key %v is not a string", got[i])
		}
		pairs[key] = got[i+1]
	}
	for key, want := range ctx {
		if got := pairs[key]; got != want {
			t.Fatalf("context[%q] = %v, want %v", key, got, want)
		}
	}
}
