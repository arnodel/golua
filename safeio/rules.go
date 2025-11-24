package safeio

import (
	"strings"
)

// FSAccessEffect represents an effect on the filesystem, i.e. a set of
// FSActions explicitly allowed and a set of FSActions explicitly denied.
type FSAccessEffect struct {
	AllowedActions FSAction
	DeniedActions  FSAction
}

// MergeWith combines two FSAccessEffect instances and returns a new
// FSAccessEffect allowing actions allowed by either effect and denying all
// actions denied by either effect.
func (e FSAccessEffect) MergeWith(e1 FSAccessEffect) FSAccessEffect {
	e.AllowedActions |= e1.AllowedActions
	e.DeniedActions |= e1.DeniedActions
	return e
}

// ChainWIth combines two FSAccessEffect instances and returns a new
// FSAccessEffect allowing only actions allowed in both effects and denying all
// actions denied by either effect.
func (e FSAccessEffect) ChainWith(e1 FSAccessEffect) FSAccessEffect {
	e.AllowedActions &= e1.AllowedActions
	e.DeniedActions |= e1.DeniedActions
	return e
}

// Allows returns true only if all actions are allowed by the effect and none
// are denied by the effect.
func (e FSAccessEffect) Allows(actions FSAction) bool {
	return e.AllowedActions&actions == actions && e.DeniedActions&actions == 0
}

// FSAccessRule knows how to allow or deny actions on a certain file path.
type FSAccessRule interface {
	GetFSAccessEffect(path string, requested FSAction) FSAccessEffect
}

// FSAccessRuleset allows grouping rules together.
type FSAccessRuleset struct {
	Rules []FSAccessRule
}

// GetFSAccessEffect returns Deny if any of its rules returns Deny, otherwise
// returns Allow if any of its rules returns Allow, otherwise returns None.
func (s FSAccessRuleset) GetFSAccessEffect(path string, requested FSAction) (effect FSAccessEffect) {
	for _, r := range s.Rules {
		effect = effect.ChainWith(r.GetFSAccessEffect(path, requested))
	}
	return
}

// FSAccessRulechain chains rules, i.e. for an action to be allowed it needs to
// be allowed by all members of the chain.
type FSAccessRulechain struct {
	Rules []FSAccessRule
}

func (r FSAccessRulechain) GetFSAccessEffect(path string, requested FSAction) (effect FSAccessEffect) {
	effect.AllowedActions = AllFileActions
	for _, r := range r.Rules {
		effect = effect.ChainWith(r.GetFSAccessEffect(path, requested))
	}
	return
}

type PrefixFSAccessRule struct {
	Prefix string
	Effect FSAccessEffect
}

func (r PrefixFSAccessRule) GetFSAccessEffect(path string, actions FSAction) (effect FSAccessEffect) {
	if !strings.HasPrefix(path, r.Prefix) {
		// If the path does start with r.Prefix, there is no effect from this rule.
		return
	}
	effect.AllowedActions = r.Effect.AllowedActions & actions
	effect.DeniedActions = r.Effect.DeniedActions & actions
	return
}

// MergeFSAccessRules returns an FSAccessRule representing all the rules passed
// in.  It discards nil rules and flattens rulesets.
func MergeFSAccessRules(rules ...FSAccessRule) FSAccessRule {
	var mergedRules []FSAccessRule
	for _, r := range rules {
		if r == nil {
			continue
		}
		if s, ok := r.(FSAccessRuleset); ok {
			mergedRules = append(mergedRules, s.Rules...)
		} else {
			mergedRules = append(mergedRules, r)
		}
	}
	switch len(mergedRules) {
	case 0:
		return nil
	case 1:
		return mergedRules[0]
	default:
		return FSAccessRuleset{Rules: mergedRules}
	}
}

// ChainFSAccessRules returns an FSAccessRule chaining all the rules passed in.
// It discards nil rules.
func ChainFSAccessRules(rules ...FSAccessRule) FSAccessRule {
	var chain []FSAccessRule
	for _, r := range rules {
		if r == nil {
			continue
		}
		chain = append(chain, r)
	}
	switch len(chain) {
	case 0:
		return nil
	case 1:
		return chain[0]
	default:
		return FSAccessRulechain{Rules: chain}
	}
}
