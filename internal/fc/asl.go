package fc

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/asl"
	"github.com/grussorusso/serverledge/internal/function"
)

// FromASL parses a AWS State Language specification file and returns a Function Composition with the corresponding Serverledge Dag
// The name of the composition should not be the file name by default, to avoid problems when adding the same composition multiple times.
func FromASL(name string, aslSrc []byte) (*FunctionComposition, error) {
	stateMachine, err := asl.ParseFrom(name, aslSrc)
	if err != nil {
		return nil, fmt.Errorf("could not parse the ASL file: %v", err)
	}
	dag, err := FromStateMachine(stateMachine)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ASL State Machine to Serverledge DAG: %v", err)
	}

	// we do not care whether function names are duplicate, we handle this in the composition
	funcNames := stateMachine.GetFunctionNames()
	functions := make([]*function.Function, 0)
	for _, f := range funcNames {
		funcObj, ok := function.GetFunction(f)
		if !ok {
			return nil, fmt.Errorf("function does not exists")
		}
		functions = append(functions, funcObj)
	}

	comp := NewFC(stateMachine.Name, *dag, functions, true)
	return &comp, nil
}

/* ============== Build from ASL States =================== */

// BuildFromTaskState adds a SimpleNode to the previous Node. The simple node will have id as specified by the name parameter
func BuildFromTaskState(builder *DagBuilder, t *asl.TaskState, name string) (*DagBuilder, error) {
	f, found := function.GetFunction(t.Resource) // Could have been used t.GetResources()[0], but it is better to avoid the array dereference
	if !found {
		return nil, fmt.Errorf("non existing function in composition: %s", t.Resource)
	}
	builder = builder.AddSimpleNodeWithId(f, name)
	fmt.Printf("Added simple node with f: %s\n", f.Name)
	return builder, nil
}

// BuildFromChoiceState adds a ChoiceNode as defined in the ChoiceState, connects it to the previous Node, and TERMINATES the DAG
func BuildFromChoiceState(builder *DagBuilder, c *asl.ChoiceState, name string, entireSM *asl.StateMachine) (*Dag, error) {
	conds, err := BuildConditionFromRule(c.Choices)
	if err != nil {
		return nil, err
	}
	branchBuilder := builder.AddChoiceNode(conds...)

	// the choice state has two or more StateMachine(s) in it, one for each branch
	i := 0
	for branchBuilder.HasNextBranch() {
		var nextState string
		if i < len(conds)-1 {
			// choice branches
			nextState = c.Choices[i].GetNextState()
		} else {
			// we add one more branch to the ChoiceNode to handle the default branch
			nextState = c.Default
		}
		sm, errBranch := GetBranchForChoiceFromStates(entireSM, nextState, i)
		if errBranch != nil {
			return nil, errBranch
		}
		branchBuilder = branchBuilder.NextBranch(sm, errBranch)
		i++
	}

	return branchBuilder.EndChoiceAndBuild()
}

// BuildConditionFromRule creates a condition from a rule
func BuildConditionFromRule(rules []asl.ChoiceRule) ([]Condition, error) {
	conds := make([]Condition, 0)

	for i, rule := range rules {
		switch t := rule.(type) {
		case *asl.BooleanExpression:
			condition, err := buildBooleanExpr(t)
			if err != nil {
				return []Condition{}, fmt.Errorf("failed to build boolean expression %d: %v", i, err)
			}
			conds = append(conds, condition)
			break
		case *asl.DataTestExpression:
			condition, err := buildTestExpr(t.Test)
			if err != nil {
				return []Condition{}, fmt.Errorf("failed to build data test expression %d: %v", i, err)
			}
			conds = append(conds, condition)
			break
		default:
			return []Condition{}, fmt.Errorf("this is not a ChoiceRule: %v", rule)
		}
	}

	// this is for the default branch
	conds = append(conds, NewConstCondition(true))
	return conds, nil
}

func buildBooleanExpr(b *asl.BooleanExpression) (Condition, error) {
	var condition Condition
	switch t := b.Formula.(type) {
	case *asl.AndFormula:
		andConditions := make([]Condition, 0)
		for i, andExpr := range t.And {
			testExpr, err := buildTestExpr(andExpr)
			if err != nil {
				return NewConstCondition(false), fmt.Errorf("failed to build AND test expression %d %v:\n %v", i, t, err)
			}
			andConditions = append(andConditions, testExpr)
		}
		condition = NewAnd(andConditions...)
		break
	case *asl.OrFormula:
		orConditions := make([]Condition, 0)
		for i, orExpr := range t.Or {
			testExpr, err := buildTestExpr(orExpr)
			if err != nil {
				return NewConstCondition(false), fmt.Errorf("failed to build OR test expression %d %v:\n %v", i, t, err)
			}
			orConditions = append(orConditions, testExpr)
		}
		condition = NewAnd(orConditions...)
		break
	case *asl.NotFormula:
		testExpr, err := buildTestExpr(t.Not)
		if err != nil {
			return NewConstCondition(false), fmt.Errorf("failed to build NOT test expression %v:\n %v", t, err)
		}
		condition = NewNot(testExpr)
		break
	default:
		condition = NewConstCondition(false)
		break
	}
	return condition, nil
}

func buildTestExpr(t *asl.TestExpression) (Condition, error) {
	var param, val *ParamOrValue
	// the Variable could be a parameter or a value, like "true", 1, etc.
	if asl.IsReferencePath(t.Variable) {
		param = NewParam(asl.RemoveDollar(t.Variable))
	} else {
		param = NewValue(t.Variable)
	}
	// The operand could be a constant or another parameter
	operand := t.ComparisonOperator.Operand
	if asl.IsReferencePath(operand) {
		operandPath, ok := operand.(string)
		if !ok {
			return NewConstCondition(false), fmt.Errorf("invalid comparison operator operand: it should have been a string")
		}
		val = NewParam(asl.RemoveDollar(operandPath))
	} else {
		val = NewValue(operand)
	}
	var condition Condition
	switch t.ComparisonOperator.Kind {
	case "StringEquals":
		condition = NewEqParamCondition(param, val)
		break
	case "StringEqualsPath":
	case "StringLessThan":
	case "StringLessThanPath":
	case "StringGreaterThan":
	case "StringGreaterThanPath":
	case "StringLessThanEquals":
	case "StringLessThanEqualsPath":
	case "StringGreaterThanEquals":
	case "StringGreaterThanEqualsPath":
	case "StringMatches":
		return NewConstCondition(false), fmt.Errorf("not implemented")
	case "NumericEquals":
		condition = NewEqParamCondition(param, val)
		break
	case "NumericEqualsPath":
		return NewConstCondition(false), fmt.Errorf("not implemented")
	case "NumericLessThan":
		condition = NewSmallerParamCondition(param, val)
		break
	case "NumericLessThanPath":
		return NewConstCondition(false), fmt.Errorf("not implemented")
	case "NumericGreaterThan":
		condition = NewGreaterParamCondition(param, val)
		break
	case "NumericGreaterThanPath":
		return NewConstCondition(false), fmt.Errorf("not implemented")
	case "NumericLessThanEquals":
		condition = NewOr(NewSmallerParamCondition(param, val), NewEqParamCondition(param, val))
		break
	case "NumericLessThanEqualsPath":
		return NewConstCondition(false), fmt.Errorf("not implemented")
	case "NumericGreaterThanEquals":
		condition = NewOr(NewGreaterParamCondition(param, val), NewEqParamCondition(param, val))
		break
	case "NumericGreaterThanEqualsPath":
		return NewConstCondition(false), fmt.Errorf("not implemented")
	case "BooleanEquals":
		condition = NewEqCondition(param, true)
		break
	case "BooleanEqualsPath":
		return NewConstCondition(false), fmt.Errorf("not implemented")
	case "TimestampEquals":
	case "TimestampEqualsPath":
	case "TimestampLessThan":
	case "TimestampLessThanPath":
	case "TimestampGreaterThan":
	case "TimestampGreaterThanPath":
	case "TimestampLessThanEquals":
	case "TimestampLessThanEqualsPath":
	case "TimestampGreaterThanEquals":
	case "TimestampGreaterThanEqualsPath":
		return NewConstCondition(false), fmt.Errorf("not implemented")
	case "IsNull":
		condition = NewEqCondition(param, NewValue(nil))
		break
	case "IsPresent":
		condition = NewNot(NewEqCondition(param, NewValue(nil)))
		break
	case "IsNumeric":
		break
	case "IsString":
		break
	case "IsBoolean":
		condition = NewOr(NewEqCondition(param, true), NewEqCondition(param, false))
	case "IsTimestamp":
		return NewConstCondition(false), fmt.Errorf("not implemented")
	}
	return condition, nil
}

func GetBranchForChoiceFromStates(sm *asl.StateMachine, nextState string, branchIndex int) (*Dag, error) {
	return DagBuildingLoop(sm, sm.States[nextState], nextState)
}

// BuildFromParallelState adds a FanOutNode and a FanInNode and as many branches as defined in the ParallelState
func BuildFromParallelState(builder *DagBuilder, c *asl.ParallelState, name string) (*DagBuilder, error) {
	// TODO: implement me
	return builder, nil
}

// BuildFromMapState is not compatible with Serverledge at the moment
func BuildFromMapState(builder *DagBuilder, c *asl.MapState, name string) (*DagBuilder, error) {
	// TODO: implement me
	// TODO: implement MapNode
	panic("not compatible with serverledge currently")
	// return builder, nil
}

// BuildFromPassState adds a SimpleNode with an identity function
func BuildFromPassState(builder *DagBuilder, p *asl.PassState, name string) (*DagBuilder, error) {
	// TODO: implement me
	return builder, nil
}

// BuildFromWaitState adds a Simple node with a sleep function for the specified time as described in the WaitState
func BuildFromWaitState(builder *DagBuilder, w *asl.WaitState, name string) (*DagBuilder, error) {
	// TODO: implement me
	return builder, nil
}

// BuildFromSucceedState is not fully compatible with serverledge, but it adds an EndNode
func BuildFromSucceedState(builder *DagBuilder, s *asl.SucceedState, name string) (*DagBuilder, error) {
	// TODO: implement me
	return builder, nil
}

// BuildFromFailState is not fully compatible with serverledge, but it adds an EndNode
func BuildFromFailState(builder *DagBuilder, s *asl.FailState, name string) (*DagBuilder, error) {
	// TODO: implement me
	return builder, nil
}
