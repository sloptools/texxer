package nodes

// GraphIO are the subgraph boundary nodes. The graph package evaluates them itself.
var GraphIO = map[string]Def{
	"input": {
		Category: "io",
		Inputs:   []string{"default"},
		Outputs:  []string{"out"},
		Params:   []Param{text("name", "")},
	},
	"output": {
		Category: "io",
		Inputs:   []string{"in"},
		Outputs:  []string{"out"},
		Params:   []Param{text("name", "out")},
	},
}
