package tool

import (
	"context"
	"testing"
)

func TestStaticToolImplementsTool(t *testing.T) {
	var testTool Tool = StaticTool{
		ToolSpec: Spec{
			Name:        "finish",
			Description: "finish task",
			Parameters: map[string]ParameterSpec{
				"final_answer": {Type: TypeString, Required: true},
			},
			Required: []string{"final_answer"},
		},
		Result: Result{Status: StatusOK, Message: "done", Finished: true, Final: true},
	}

	if testTool.Spec().Name != "finish" {
		t.Fatalf("unexpected tool name: %q", testTool.Spec().Name)
	}

	result, err := testTool.Execute(context.Background(), Call{ToolName: "finish"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Finished {
		t.Fatalf("expected finished result")
	}
}

func TestSpecCarriesParameters(t *testing.T) {
	spec := FilesystemTool{}.Spec()
	if spec.Parameters["operation"].Type != TypeString {
		t.Fatalf("unexpected operation type: %+v", spec.Parameters["operation"])
	}
	if len(spec.Required) == 0 {
		t.Fatalf("expected required fields")
	}
}
