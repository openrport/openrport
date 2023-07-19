package rules

import (
	"reflect"
	"testing"
)

func TestShouldCloneActionList(t *testing.T) {
	notifyList := NotifyList{"template1", "template2"}
	ignoreList := IgnoreList{"spec1", "spec2"}
	logMessage := LogMessage("log message")

	action1 := Action{
		NotifyList: &notifyList,
		IgnoreList: &ignoreList,
		LogMessage: logMessage,
	}

	action2 := Action{
		NotifyList: &notifyList,
		IgnoreList: &ignoreList,
		LogMessage: logMessage,
	}

	actionList := ActionList{action1, action2}

	clonedActionList := actionList.Clone()

	// Check if the length of clonedActionList is the same
	if len(actionList) != len(clonedActionList) {
		t.Errorf("Cloned ActionList has a different length")
	}

	// Check if each Action in the original list is cloned correctly
	for i := 0; i < len(actionList); i++ {
		if !reflect.DeepEqual(actionList[i], clonedActionList[i]) {
			t.Errorf("Action at index %d not cloned correctly", i)
		}
	}
}
