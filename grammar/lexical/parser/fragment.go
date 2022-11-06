package parser

import (
	"fmt"

	spec "github.com/nihei9/vartan/spec/grammar"
)

type incompleteFragment struct {
	kind spec.LexKindName
	root *rootNode
}

func CompleteFragments(fragments map[spec.LexKindName]CPTree) error {
	if len(fragments) == 0 {
		return nil
	}

	completeFragments := map[spec.LexKindName]CPTree{}
	incompleteFragments := []*incompleteFragment{}
	for kind, tree := range fragments {
		root, ok := tree.(*rootNode)
		if !ok {
			return fmt.Errorf("CompleteFragments can take only *rootNode: %T", tree)
		}
		if root.incomplete() {
			incompleteFragments = append(incompleteFragments, &incompleteFragment{
				kind: kind,
				root: root,
			})
		} else {
			completeFragments[kind] = root
		}
	}
	for len(incompleteFragments) > 0 {
		lastIncompCount := len(incompleteFragments)
		remainingFragments := []*incompleteFragment{}
		for _, e := range incompleteFragments {
			complete, err := ApplyFragments(e.root, completeFragments)
			if err != nil {
				return err
			}
			if !complete {
				remainingFragments = append(remainingFragments, e)
			} else {
				completeFragments[e.kind] = e.root
			}
		}
		incompleteFragments = remainingFragments
		if len(incompleteFragments) == lastIncompCount {
			return ParseErr
		}
	}

	return nil
}

func ApplyFragments(t CPTree, fragments map[spec.LexKindName]CPTree) (bool, error) {
	root, ok := t.(*rootNode)
	if !ok {
		return false, fmt.Errorf("ApplyFragments can take only *rootNode type: %T", t)
	}

	for name, frag := range fragments {
		err := root.applyFragment(name, frag)
		if err != nil {
			return false, err
		}
	}

	return !root.incomplete(), nil
}
