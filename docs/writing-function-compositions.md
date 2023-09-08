# Writing functions compositions

Serverledge accepts DAGs defined with two types of syntax: YAML based and JSON based. Furthermore you can also define DAGs
with the DagBuilder APIs.

There are 4 types of nodes that you can Add in the Dag:
- SimpleNode: a node that wraps a function. This is the only node that executes user-defined functions.
- ChoiceNode: a node with N conditions that sends input to the first branches that evaluates the condition to true
- FanOutNode: a node with N outputs, that copies or Scatters the input throw all outputs, and the subsequent functions are run in parallel
-  FanInNode: a node with N inputs, that waits the termination of all previous functions and then merge the results in one output. Fails after a specified timeout.

Three special node are always present and pre-built when using the APIs:
- StartNode: a node from which the dag starts executing. Only one Startnode can be
- EndNode: the final node of the dag. Multiple nodes can chain to the EndNode.
- ErrorNode: a node that terminates the dag with failure and explains why that occurred. Failing ChoiceNode or FanInNodes automatically invoke the ErrorNode.

## AFCL - Abstract Function Composition Language


## AWS State Language for Amazon Step Function

## Go DagBuilder APIs

It is possible to use the builder APIs to build complex serverless DAG in the Go Language.
Here is an example of a Dag made by two simpleNodes and a ChoiceNode, with N alternative conditions

N := 4
function := function.Function{...}
condition := make([]Condition, N)
condition[0] = Condition{...}
...
condition[N-1] = Condition{...}

NewDagBuilder().
    AddSimpleNode(&function).
    AddSimpleNode(&function2).
    AddChoiceNode(conditions).
    ForEach(NewDagBuilder().
            AddSimpleNode(&function).
            Build()).
    EndChoiceNode().
    Build()