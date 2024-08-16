package asl

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
)

type ChoiceState struct {
	Type           StateType
	Matches        []*Match
	InputPath      Path
	OutputPath     Path
	ResultPath     Path
	Parameters     string
	ResultSelector string
	Retry          *Retry
	Catch          *Catch
	Next           string
	End            bool
}

func (c *ChoiceState) Equals(cmp types.Comparable) bool {
	//TODO implement me
	panic("implement me")
}

func NewEmptyChoice() *ChoiceState {
	return &ChoiceState{
		Type: Choice,
	}
}

func (c *ChoiceState) ParseFrom(jsonData []byte) (State, error) {
	/*choices, _, _, errChoice := jsonparser.Get(value, "Choices")
	if errChoice != nil {
		return errChoice
	}
	matches := make([]Match, 0)

	_, errArr := jsonparser.ArrayEach(choices, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		matchVariable, _, _, errMatch := jsonparser.Get(value, "Variable")
		if errMatch != nil {
			return
		}
		matchCondition, _, _, errMatch := jsonparser.Get(value, "NumericEquals") // TODO: only checks NumericEquals
		if errMatch != nil {
			return
		}
		num, errNum := strconv.Atoi(string(matchCondition))
		if errNum != nil {
			return
		}
		matchNext, _, _, errMatch := jsonparser.Get(value, "Next")
		if errMatch != nil {
			return
		}
		m := Match{
			Variable:  string(matchVariable),
			Operation: NewEqCondition(NewParam(string(matchVariable)), NewValue(num)),
			Next:      string(matchNext),
		}
		matches = append(matches, m)
	})
	if errArr != nil {
		return errArr
	}

	defaultMatch, _, _, errChoice := jsonparser.Get(value, "Default")
	if errChoice != nil {
		return err
	}*/
	return nil, nil
}

func (c *ChoiceState) GetNext() (string, bool) {
	if c.End == false {
		return c.Next, true
	}
	return "", false
}

func (c *ChoiceState) GetType() StateType {
	return Choice
}

type Match struct {
	Variable  string
	Operation string // FIXME: come up with a better type (Do not use fc.Condition, or you will have an import cycle)
	Next      string
}

func (m *Match) Equals(cmp types.Comparable) bool {
	m2 := cmp.(*Match)
	return m.Next == m2.Next &&
		m.Operation == m2.Operation &&
		m.Variable == m2.Variable
}

// FIXME: improve
func (c *ChoiceState) String() string {
	return fmt.Sprintf("%v", c.Matches)
}
